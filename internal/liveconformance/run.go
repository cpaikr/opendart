package liveconformance

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"

	"github.com/cpaikr/opendart/internal/liveprobe"
)

// Run executes a previously preflighted plan. Constructing the plan is what
// guarantees that the credential is not read until all offline gates pass.
func (plan *Plan) Run(ctx context.Context) (Report, error) {
	client := liveprobe.NewSequentialHTTPClient(RequestTimeout)
	return execute(ctx, plan, dependencies{
		do: client.Do,
		credential: func() (string, error) {
			value := os.Getenv(CredentialSource)
			if len(value) != 40 {
				return "", errors.New("credential is unavailable")
			}
			return value, nil
		},
		now: time.Now,
		wait: func(duration time.Duration) error {
			timer := time.NewTimer(duration)
			defer timer.Stop()
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-timer.C:
				return nil
			}
		},
	})
}

func execute(ctx context.Context, plan *Plan, deps dependencies) (Report, error) {
	report := newReport(deps.now(), plan.requestBudget)
	credential, err := deps.credential()
	if err != nil || len(credential) != 40 {
		return failReport(report, credential, Failure{Code: "credential-unavailable", Stage: "credential"})
	}
	for index, prepared := range plan.cases {
		if report.RequestBudget.Used >= report.RequestBudget.Ceiling {
			return failReport(report, credential, caseFailure("request-budget-exhausted", "budget", prepared))
		}
		result, failure := executeCase(ctx, plan.specification, deps.do, credential, prepared)
		report.RequestBudget.Used++
		report.Cases = append(report.Cases, result)
		if failure != nil {
			return failReport(report, credential, *failure)
		}
		if index+1 < len(plan.cases) {
			if err := deps.wait(RequestPacing); err != nil {
				return failReport(report, credential, caseFailure("pacing-interrupted", "pacing", prepared))
			}
		}
	}
	report.Outcome = "passed"
	if err := validateReport(report, credential); err != nil {
		return failReport(report, credential, Failure{Code: "report-sanitization", Stage: "report"})
	}
	return report, nil
}

func executeCase(ctx context.Context, spec specification, do func(*http.Request) (*http.Response, error), credential string, prepared preparedCase) (CaseResult, *Failure) {
	result := CaseResult{
		CaseID:             prepared.definition.ID,
		OperationID:        prepared.operation.OperationID,
		LogicalOperationID: prepared.operation.LogicalOperationID,
		Method:             prepared.operation.Method,
		Path:               prepared.operation.Path,
		Representation:     prepared.operation.PrimaryRepresentation,
		AssertionID:        prepared.definition.Assertion,
		Outcome:            "failed",
		SchemaLocation:     prepared.operation.Path + "#responses/default/content/" + escapePointer(prepared.operation.PrimaryRepresentation),
	}
	requestURL, err := trustedRequestURL(prepared, credential)
	if err != nil {
		failure := caseFailure("request-construction", "request", prepared)
		return result, &failure
	}
	requestContext, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, prepared.operation.Method, requestURL.String(), nil)
	if err != nil {
		failure := caseFailure("request-construction", "request", prepared)
		return result, &failure
	}
	request.Header.Set("Accept", prepared.operation.PrimaryRepresentation)
	request.Header.Set("User-Agent", "opendart-live-conformance/1.0")
	response, err := do(request)
	if err != nil || response == nil {
		failure := caseFailure("transport-failure", "request", prepared)
		return result, &failure
	}
	result.HTTPStatus = response.StatusCode
	body, err := readBoundedBody(response.Body)
	if err != nil {
		failure := caseFailure("bounded-body-failure", "response", prepared)
		return result, &failure
	}
	result.BodyBytes = len(body)
	digest := sha256.Sum256(body)
	result.BodySHA256 = hex.EncodeToString(digest[:])
	contentType := response.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		failure := caseFailure("invalid-media-type", "response", prepared)
		return result, &failure
	}
	result.MediaType = strings.ToLower(mediaType)
	if err := spec.ValidateResponse(prepared.operation.Method, prepared.operation.Path, contentType, response.StatusCode, body); err != nil {
		failure := caseFailure("openapi-response-validation", "response", prepared)
		return result, &failure
	}
	parsed, err := parseResponse(result.MediaType, body)
	if err != nil {
		failure := caseFailure("representation-parse", "response", prepared)
		return result, &failure
	}
	result.APIStatus = parsed.APIStatus
	if response.StatusCode != http.StatusOK || result.MediaType != prepared.operation.PrimaryRepresentation || structuredRepresentation(result.MediaType) && parsed.APIStatus != "000" {
		failure := caseFailure("unsuccessful-envelope", "response", prepared)
		return result, &failure
	}
	evidence, passed := prepared.assertion.Check(parsed)
	result.Comparison = evidence
	if !passed {
		failure := caseFailure("assertion-failed", "assertion", prepared)
		return result, &failure
	}
	result.Outcome = "passed"
	return result, nil
}

func trustedRequestURL(prepared preparedCase, credential string) (*url.URL, error) {
	base, err := url.Parse(TrustedServer)
	if err != nil {
		return nil, err
	}
	if !strings.HasPrefix(prepared.operation.Path, "/") || strings.Contains(prepared.operation.Path, "?") || strings.Contains(prepared.operation.Path, "#") {
		return nil, errors.New("operation path is invalid")
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + prepared.operation.Path
	query := cloneValues(prepared.query)
	query.Set("crtfc_key", credential)
	base.RawQuery = query.Encode()
	if base.Scheme != "https" || base.Host != "opendart.fss.or.kr" || !strings.HasPrefix(base.Path, "/api/") || base.User != nil {
		return nil, errors.New("request target is not trusted")
	}
	return base, nil
}

func readBoundedBody(body io.ReadCloser) ([]byte, error) {
	if body == nil {
		return nil, errors.New("body is missing")
	}
	value, readErr := io.ReadAll(io.LimitReader(body, MaximumBodyBytes+1))
	closeErr := body.Close()
	if readErr != nil || closeErr != nil || len(value) == 0 || len(value) > MaximumBodyBytes {
		return nil, errors.New("body is invalid")
	}
	return value, nil
}

func parseResponse(representation string, body []byte) (Response, error) {
	switch representation {
	case "application/json":
		return parseJSON(body)
	case "application/xml":
		return parseXML(body)
	case "application/zip":
		return parseZIP(body)
	default:
		return Response{}, errors.New("unsupported representation")
	}
}

func parseJSON(body []byte) (Response, error) {
	var value map[string]any
	if err := json.Unmarshal(body, &value); err != nil || value == nil {
		return Response{}, errors.New("JSON is invalid")
	}
	status, _ := value["status"].(string)
	if status == "" {
		return Response{}, errors.New("JSON status is missing")
	}
	return Response{Representation: "application/json", APIStatus: status, JSON: value}, nil
}

func parseXML(body []byte) (Response, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	stack := make([]string, 0)
	text := make([]strings.Builder, 0)
	values := make(map[string][]string)
	rootSeen := false
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return Response{}, errors.New("XML is invalid")
		}
		switch current := token.(type) {
		case xml.StartElement:
			if len(stack) == 0 {
				if rootSeen {
					return Response{}, errors.New("XML has multiple roots")
				}
				rootSeen = true
			}
			stack = append(stack, current.Name.Local)
			text = append(text, strings.Builder{})
		case xml.CharData:
			if len(text) > 0 {
				text[len(text)-1].Write([]byte(current))
			}
		case xml.EndElement:
			if len(stack) == 0 || stack[len(stack)-1] != current.Name.Local {
				return Response{}, errors.New("XML nesting is invalid")
			}
			path := strings.Join(stack, "/")
			if value := strings.TrimSpace(text[len(text)-1].String()); value != "" {
				values[path] = append(values[path], value)
			}
			stack = stack[:len(stack)-1]
			text = text[:len(text)-1]
		}
	}
	if !rootSeen || len(stack) != 0 {
		return Response{}, errors.New("XML root is missing")
	}
	status := firstElementValue(values, "status")
	if status == "" {
		return Response{}, errors.New("XML status is missing")
	}
	return Response{Representation: "application/xml", APIStatus: status, XMLValues: values}, nil
}

func parseZIP(body []byte) (Response, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil || len(reader.File) == 0 {
		return Response{}, errors.New("ZIP is invalid")
	}
	summary := ArchiveSummary{Entries: len(reader.File)}
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		summary.ExpandedBytes += int(file.UncompressedSize64)
		extension := strings.ToLower(file.Name)
		if strings.HasSuffix(extension, ".xml") || strings.HasSuffix(extension, ".xbrl") {
			summary.XMLDocuments++
		}
	}
	if summary.XMLDocuments == 0 || summary.ExpandedBytes == 0 {
		return Response{}, errors.New("ZIP has no meaningful XML content")
	}
	return Response{Representation: "application/zip", Archive: summary}, nil
}

func firstElementValue(values map[string][]string, element string) string {
	for path, candidates := range values {
		if (path == element || strings.HasSuffix(path, "/"+element)) && len(candidates) > 0 {
			return candidates[0]
		}
	}
	return ""
}

func structuredRepresentation(representation string) bool {
	return representation == "application/json" || representation == "application/xml"
}

func newReport(now time.Time, requestCeiling int) Report {
	return Report{
		SchemaVersion:       ReportSchemaVersion,
		Kind:                ReportKind,
		Outcome:             "failed",
		ObservedAt:          now.UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z"),
		CredentialSource:    CredentialSource,
		CredentialPersisted: false,
		RequestBudget:       RequestBudget{Ceiling: requestCeiling},
		Cases:               []CaseResult{},
	}
}

func failReport(report Report, credential string, failure Failure) (Report, error) {
	report.Outcome = "failed"
	report.Failure = &failure
	if err := validateReport(report, credential); err != nil {
		report = Report{
			SchemaVersion:       ReportSchemaVersion,
			Kind:                ReportKind,
			Outcome:             "failed",
			ObservedAt:          report.ObservedAt,
			CredentialSource:    CredentialSource,
			CredentialPersisted: false,
			RequestBudget:       RequestBudget{},
			Cases:               []CaseResult{},
			Failure:             &Failure{Code: "report-sanitization", Stage: "report"},
		}
	}
	return report, &Error{Failure: *report.Failure}
}

func validateReport(report Report, credential string) error {
	if report.SchemaVersion != ReportSchemaVersion || report.Kind != ReportKind || report.CredentialSource != CredentialSource || report.CredentialPersisted || report.RequestBudget.Ceiling < 0 || report.RequestBudget.Ceiling > AbsoluteRequestLimit || report.RequestBudget.Used < 0 || report.RequestBudget.Used > report.RequestBudget.Ceiling || len(report.Cases) != report.RequestBudget.Used {
		return errors.New("report invariants failed")
	}
	if _, err := time.Parse("2006-01-02T15:04:05.000Z", report.ObservedAt); err != nil {
		return errors.New("report timestamp is invalid")
	}
	if report.Outcome == "passed" {
		if report.Failure != nil || report.RequestBudget.Used != report.RequestBudget.Ceiling {
			return errors.New("passed report is incomplete")
		}
		for _, result := range report.Cases {
			if result.Outcome != "passed" {
				return errors.New("passed report contains a failed case")
			}
		}
	} else if report.Outcome != "failed" || report.Failure == nil {
		return errors.New("failed report has no failure")
	} else if len(report.Cases) > 0 && report.Failure.CaseID != "" && report.Cases[len(report.Cases)-1].CaseID != report.Failure.CaseID {
		return errors.New("failed report does not identify the last case")
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return err
	}
	if credential != "" && bytes.Contains(encoded, []byte(credential)) {
		return errors.New("report contains credential")
	}
	return nil
}

func caseFailure(code, stage string, prepared preparedCase) Failure {
	return Failure{Code: code, Stage: stage, CaseID: prepared.definition.ID, Operation: prepared.operation.Identity()}
}

func cloneValues(source url.Values) url.Values {
	result := make(url.Values, len(source))
	for name, values := range source {
		result[name] = append([]string(nil), values...)
	}
	return result
}

func escapePointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}
