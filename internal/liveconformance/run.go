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
	"unicode/utf8"

	"github.com/cpaikr/opendart/internal/liveprobe"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const maximumXMLDepth = 128

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
	report := newReport(deps.now(), plan.requestBudget, plan.discoveryBudget)
	credential, err := deps.credential()
	if err != nil || len(credential) != 40 {
		return failReport(report, credential, Failure{Code: "credential-unavailable", Stage: "credential"})
	}
	resolvedCases, discoveryFailure := executeDiscoveries(ctx, plan, deps, credential, &report)
	if discoveryFailure != nil {
		return failReport(report, credential, *discoveryFailure)
	}
	for index, prepared := range resolvedCases {
		if report.RequestBudget.Used >= report.RequestBudget.Ceiling {
			return failReport(report, credential, caseFailure("request-budget-exhausted", "budget", prepared))
		}
		result, _, failure := executeCase(ctx, plan.specification, deps.do, credential, prepared)
		report.RequestBudget.Used++
		report.Cases = append(report.Cases, result)
		if failure != nil {
			return failReport(report, credential, *failure)
		}
		if index+1 < len(resolvedCases) {
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

func executeCase(ctx context.Context, spec specification, do func(*http.Request) (*http.Response, error), credential string, prepared preparedCase) (CaseResult, Response, *Failure) {
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
		return result, Response{}, &failure
	}
	requestContext, cancel := context.WithTimeout(ctx, RequestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, prepared.operation.Method, requestURL.String(), nil)
	if err != nil {
		failure := caseFailure("request-construction", "request", prepared)
		return result, Response{}, &failure
	}
	request.Header.Set("Accept", prepared.operation.PrimaryRepresentation)
	request.Header.Set("User-Agent", "opendart-live-conformance/1.0")
	response, err := do(request)
	if err != nil {
		if response != nil && response.Body != nil {
			_ = response.Body.Close()
		}
		failure := caseFailure("transport-failure", "request", prepared)
		return result, Response{}, &failure
	}
	if response == nil {
		failure := caseFailure("transport-failure", "request", prepared)
		return result, Response{}, &failure
	}
	result.HTTPStatus = response.StatusCode
	body, err := readBoundedBody(response.Body)
	if err != nil {
		failure := caseFailure("bounded-body-failure", "response", prepared)
		return result, Response{}, &failure
	}
	result.BodyBytes = len(body)
	digest := sha256.Sum256(body)
	result.BodySHA256 = hex.EncodeToString(digest[:])
	contentType := response.Header.Get("Content-Type")
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		failure := caseFailure("invalid-media-type", "response", prepared)
		return result, Response{}, &failure
	}
	normalizedMediaType := strings.ToLower(mediaType)
	canonicalMediaType := canonicalResponseMediaType(prepared.operation.PrimaryRepresentation, normalizedMediaType, body)
	if !allowedRepresentation(canonicalMediaType) {
		failure := caseFailure("invalid-media-type", "response", prepared)
		return result, Response{}, &failure
	}
	result.MediaType = canonicalMediaType
	if err := spec.ValidateResponse(prepared.operation.Method, prepared.operation.Path, canonicalMediaType, response.StatusCode, body); err != nil {
		failure := caseFailure("openapi-response-validation", "response", prepared)
		return result, Response{}, &failure
	}
	parsed, err := parseResponse(result.MediaType, body)
	if err != nil {
		failure := caseFailure("representation-parse", "response", prepared)
		return result, parsed, &failure
	}
	result.APIStatus = parsed.APIStatus
	validAPIStatus := parsed.APIStatus == "000" || prepared.allowEmptyDiscovery && parsed.APIStatus == "013"
	if response.StatusCode != http.StatusOK || result.MediaType != prepared.operation.PrimaryRepresentation || structuredRepresentation(result.MediaType) && !validAPIStatus {
		failure := caseFailure("unsuccessful-envelope", "response", prepared)
		return result, parsed, &failure
	}
	evidence, passed := prepared.assertion.Check(parsed)
	result.Comparison = evidence
	if !passed {
		failure := caseFailure("assertion-failed", "assertion", prepared)
		return result, parsed, &failure
	}
	result.Outcome = "passed"
	return result, parsed, nil
}

func canonicalResponseMediaType(expected, observed string, body []byte) string {
	if observed == expected {
		return observed
	}
	if expected == "application/zip" && (observed == "application/x-msdownload" || observed == "application/octet-stream") && len(body) >= 4 && bytes.Equal(body[:4], []byte{'P', 'K', 3, 4}) {
		return "application/zip"
	}
	return observed
}

func trustedRequestURL(prepared preparedCase, credential string) (*url.URL, error) {
	base, err := url.Parse(TrustedServer)
	if err != nil {
		return nil, err
	}
	if !validOperationPath(prepared.operation.Path) {
		return nil, errors.New("operation path is invalid")
	}
	base.Path = strings.TrimSuffix(base.Path, "/") + prepared.operation.Path
	query := cloneValues(prepared.query)
	query.Set("crtfc_key", credential)
	base.RawQuery = query.Encode()
	if base.Scheme != "https" || base.Host != "opendart.fss.or.kr" || !strings.HasPrefix(base.Path, "/api/") || !validOperationPath(strings.TrimPrefix(base.Path, "/api")) || base.User != nil {
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
	root, values, err := parseXMLValues(body)
	if err != nil || root != "result" {
		return Response{}, errors.New("XML is invalid")
	}
	records, err := parseXMLRecords(body)
	if err != nil {
		return Response{}, errors.New("XML is invalid")
	}
	status := firstElementValue(values, "result/status")
	if status == "" {
		return Response{}, errors.New("XML status is missing")
	}
	return Response{Representation: "application/xml", APIStatus: status, XMLValues: values, XMLRecords: records}, nil
}

func parseXMLRecords(body []byte) (map[string][]map[string]string, error) {
	records, err := parseXMLRecordsDecoded(body)
	if err == nil || utf8.Valid(body) {
		return records, err
	}
	decoded, _, decodeErr := transform.Bytes(korean.EUCKR.NewDecoder(), body)
	if decodeErr != nil || !utf8.Valid(decoded) || bytes.ContainsRune(decoded, utf8.RuneError) {
		return nil, errors.New("XML encoding is invalid")
	}
	return parseXMLRecordsDecoded(decoded)
}

func parseXMLRecordsDecoded(body []byte) (map[string][]map[string]string, error) {
	type frame struct {
		name   string
		text   strings.Builder
		fields map[string]string
	}
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = koreanCharsetReader
	stack := make([]frame, 0)
	records := make(map[string][]map[string]string)
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return nil, errors.New("XML is invalid")
		}
		switch current := token.(type) {
		case xml.StartElement:
			if len(stack) >= maximumXMLDepth {
				return nil, errors.New("XML nesting is too deep")
			}
			stack = append(stack, frame{name: current.Name.Local, fields: make(map[string]string)})
		case xml.CharData:
			if len(stack) > 0 {
				stack[len(stack)-1].text.Write([]byte(current))
			}
		case xml.EndElement:
			if len(stack) == 0 || stack[len(stack)-1].name != current.Name.Local {
				return nil, errors.New("XML nesting is invalid")
			}
			last := stack[len(stack)-1]
			pathParts := make([]string, len(stack))
			for index := range stack {
				pathParts[index] = stack[index].name
			}
			path := strings.Join(pathParts, "/")
			if last.name == "list" && (path == "result/list" || path == "result/group/list") {
				records[path] = append(records[path], last.fields)
			}
			stack = stack[:len(stack)-1]
			if len(stack) > 0 {
				if value := strings.TrimSpace(last.text.String()); value != "" {
					stack[len(stack)-1].fields[last.name] = value
				}
			}
		}
	}
	if len(stack) != 0 {
		return nil, errors.New("XML root is missing")
	}
	return records, nil
}

func parseXMLValues(body []byte) (string, map[string][]string, error) {
	root, values, err := parseXMLValuesDecoded(body)
	if err == nil {
		if xmlValuesContainRuneError(values) {
			return "", nil, errors.New("XML encoding is invalid")
		}
		return root, values, nil
	}
	if utf8.Valid(body) {
		return root, values, err
	}
	decoded, _, decodeErr := transform.Bytes(korean.EUCKR.NewDecoder(), body)
	if decodeErr != nil || !utf8.Valid(decoded) || bytes.ContainsRune(decoded, utf8.RuneError) {
		return "", nil, errors.New("XML encoding is invalid")
	}
	return parseXMLValuesDecoded(decoded)
}

func xmlValuesContainRuneError(values map[string][]string) bool {
	for _, candidates := range values {
		for _, value := range candidates {
			if strings.ContainsRune(value, utf8.RuneError) {
				return true
			}
		}
	}
	return false
}

func parseXMLValuesDecoded(body []byte) (string, map[string][]string, error) {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	decoder.CharsetReader = koreanCharsetReader
	stack := make([]string, 0)
	text := make([]strings.Builder, 0)
	values := make(map[string][]string)
	root := ""
	for {
		token, err := decoder.Token()
		if errors.Is(err, io.EOF) {
			break
		}
		if err != nil {
			return "", nil, errors.New("XML is invalid")
		}
		switch current := token.(type) {
		case xml.StartElement:
			if len(stack) >= maximumXMLDepth {
				return "", nil, errors.New("XML nesting is too deep")
			}
			if len(stack) == 0 {
				if root != "" {
					return "", nil, errors.New("XML has multiple roots")
				}
				root = current.Name.Local
			}
			stack = append(stack, current.Name.Local)
			text = append(text, strings.Builder{})
		case xml.CharData:
			if len(text) > 0 {
				text[len(text)-1].Write([]byte(current))
			} else if strings.TrimSpace(string(current)) != "" {
				return "", nil, errors.New("XML has text outside its root")
			}
		case xml.EndElement:
			if len(stack) == 0 || stack[len(stack)-1] != current.Name.Local {
				return "", nil, errors.New("XML nesting is invalid")
			}
			path := strings.Join(stack, "/")
			if value := strings.TrimSpace(text[len(text)-1].String()); value != "" {
				values[path] = append(values[path], value)
			}
			stack = stack[:len(stack)-1]
			text = text[:len(text)-1]
		}
	}
	if root == "" || len(stack) != 0 {
		return "", nil, errors.New("XML root is missing")
	}
	return root, values, nil
}

func koreanCharsetReader(label string, input io.Reader) (io.Reader, error) {
	switch strings.ToLower(strings.TrimSpace(label)) {
	case "utf-8", "utf8", "us-ascii":
		return input, nil
	case "euc-kr", "ks_c_5601-1987", "cp949":
		return transform.NewReader(input, korean.EUCKR.NewDecoder()), nil
	default:
		return nil, errors.New("XML encoding is unsupported")
	}
}

func parseZIP(body []byte) (Response, error) {
	return parseZIPWithLimit(body, MaximumArchiveBytes)
}

func parseZIPWithLimit(body []byte, expansionLimit int) (Response, error) {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil || len(reader.File) == 0 || expansionLimit <= 0 {
		return Response{}, errors.New("ZIP is invalid")
	}
	summary := ArchiveSummary{Entries: len(reader.File)}
	documents := make([]ArchiveDocument, 0)
	for _, file := range reader.File {
		if file.FileInfo().IsDir() {
			continue
		}
		entry, err := file.Open()
		if err != nil {
			return Response{}, errors.New("ZIP entry is invalid")
		}
		remaining := expansionLimit - summary.ExpandedBytes
		content, readErr := io.ReadAll(io.LimitReader(entry, int64(remaining)+1))
		closeErr := entry.Close()
		if readErr != nil || closeErr != nil || len(content) > remaining {
			return Response{}, errors.New("ZIP expansion is invalid")
		}
		summary.ExpandedBytes += len(content)
		extension := strings.ToLower(file.Name)
		if strings.HasSuffix(extension, ".xml") || strings.HasSuffix(extension, ".xbrl") {
			root, values, parseErr := parseXMLValues(content)
			if parseErr == nil {
				documents = append(documents, ArchiveDocument{Root: root, XMLValues: values})
				summary.XMLDocuments++
			}
		}
	}
	if summary.XMLDocuments == 0 || summary.ExpandedBytes == 0 {
		return Response{}, errors.New("ZIP has no meaningful XML content")
	}
	return Response{Representation: "application/zip", Archive: summary, ArchiveDocuments: documents}, nil
}

func firstElementValue(values map[string][]string, path string) string {
	candidates := values[path]
	if len(candidates) > 0 {
		return candidates[0]
	}
	return ""
}

func structuredRepresentation(representation string) bool {
	return representation == "application/json" || representation == "application/xml"
}

func newReport(now time.Time, requestCeiling int, discoveryCeiling ...int) Report {
	discoveryRequests := 0
	if len(discoveryCeiling) == 1 {
		discoveryRequests = discoveryCeiling[0]
	}
	return Report{
		SchemaVersion:       ReportSchemaVersion,
		Kind:                ReportKind,
		Outcome:             "failed",
		ObservedAt:          now.UTC().Truncate(time.Millisecond).Format("2006-01-02T15:04:05.000Z"),
		CredentialSource:    CredentialSource,
		CredentialPersisted: false,
		RequestBudget:       RequestBudget{Ceiling: requestCeiling, DiscoveryCeiling: discoveryRequests},
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
	primaryCeiling := report.RequestBudget.Ceiling - report.RequestBudget.DiscoveryCeiling
	metadataValid := report.SchemaVersion == ReportSchemaVersion && report.Kind == ReportKind && report.CredentialSource == CredentialSource && !report.CredentialPersisted
	budgetValid := report.RequestBudget.Ceiling >= 0 && report.RequestBudget.Ceiling <= AbsoluteRequestLimit &&
		report.RequestBudget.DiscoveryCeiling >= 0 && report.RequestBudget.DiscoveryCeiling <= report.RequestBudget.Ceiling &&
		report.RequestBudget.Used >= 0 && report.RequestBudget.Used <= report.RequestBudget.Ceiling &&
		report.RequestBudget.DiscoveryUsed >= 0 && report.RequestBudget.DiscoveryUsed <= report.RequestBudget.DiscoveryCeiling
	caseCountValid := len(report.Cases) == report.RequestBudget.Used-report.RequestBudget.DiscoveryUsed && len(report.Cases) <= primaryCeiling
	if !metadataValid || !budgetValid || !caseCountValid {
		return errors.New("report invariants failed")
	}
	if _, err := time.Parse("2006-01-02T15:04:05.000Z", report.ObservedAt); err != nil {
		return errors.New("report timestamp is invalid")
	}
	if report.Outcome == "passed" {
		if report.Failure != nil || len(report.Cases) != primaryCeiling {
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
	if len(encoded) == 0 || len(encoded) > MaximumReportBytes {
		return errors.New("report size is invalid")
	}
	if credential != "" && bytes.Contains(encoded, []byte(credential)) {
		return errors.New("report contains credential")
	}
	return validateAllowlistedFields(report)
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
