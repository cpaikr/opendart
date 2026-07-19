package liveconformance

import (
	"archive/zip"
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"slices"
	"strings"
	"testing"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

const testCredential = "0123456789012345678901234567890123456789"

func TestPreflightRejectsNilSpecification(t *testing.T) {
	_, err := Preflight(nil, nil, nil)
	var conformanceError *Error
	if !errors.As(err, &conformanceError) || conformanceError.Failure.Code != "operation-enumeration" {
		t.Fatalf("error = %v", err)
	}
}

func TestPreflightAndExecuteRepresentativeJSONXMLAndZIP(t *testing.T) {
	spec := representativeSpecification()
	plan, err := Preflight(spec, representativeCases(), representativeAssertions())
	if err != nil {
		t.Fatal(err)
	}

	requests := make([]*http.Request, 0, 3)
	waits := 0
	report, err := execute(context.Background(), plan, dependencies{
		credential: func() (string, error) { return testCredential, nil },
		now:        func() time.Time { return time.Date(2026, 7, 19, 1, 2, 3, 456000000, time.UTC) },
		wait: func(duration time.Duration) error {
			if duration != RequestPacing {
				t.Fatalf("pacing = %s", duration)
			}
			waits++
			return nil
		},
		do: func(request *http.Request) (*http.Response, error) {
			requests = append(requests, request)
			if request.URL.Scheme != "https" || request.URL.Host != "opendart.fss.or.kr" || request.URL.Query().Get("crtfc_key") != testCredential {
				t.Fatalf("request target = %s", request.URL.Redacted())
			}
			switch request.URL.Path {
			case "/api/company.json":
				return response("application/json;charset=UTF-8", []byte(`{"status":"000","message":"OK","corp_code":"00126380"}`)), nil
			case "/api/company.xml":
				return response("application/xml;charset=UTF-8", []byte(`<result><status>000</status><message>OK</message><corp_code>00126380</corp_code></result>`)), nil
			case "/api/document.xml":
				return response("application/zip", zipBody(t)), nil
			default:
				t.Fatalf("unexpected path %q", request.URL.Path)
				return nil, nil
			}
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "passed" || report.RequestBudget != (RequestBudget{Ceiling: 3, Used: 3}) || len(report.Cases) != 3 {
		t.Fatalf("report = %#v", report)
	}
	if waits != 2 || len(requests) != 3 || spec.responseValidations != 3 {
		t.Fatalf("waits=%d requests=%d validations=%d", waits, len(requests), spec.responseValidations)
	}
	if got := []string{report.Cases[0].Representation, report.Cases[1].Representation, report.Cases[2].Representation}; !slices.Equal(got, []string{"application/json", "application/xml", "application/zip"}) {
		t.Fatalf("representations = %#v", got)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte(testCredential)) || bytes.Contains(encoded, []byte("crtfc_key")) || bytes.Contains(encoded, []byte("?")) {
		t.Fatalf("report contains request material: %s", encoded)
	}
}

func TestPreflightFailsBeforeExecutionForCoverageParametersAndTrust(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*fakeSpecification, *[]Case)
		code   string
	}{
		{name: "missing coverage", code: "incomplete-operation-coverage", mutate: func(_ *fakeSpecification, cases *[]Case) { *cases = (*cases)[:2] }},
		{name: "duplicate coverage", code: "duplicate-case-coverage", mutate: func(_ *fakeSpecification, cases *[]Case) {
			duplicate := (*cases)[0]
			duplicate.ID = "duplicate"
			*cases = append(*cases, duplicate)
		}},
		{name: "unknown parameter", code: "invalid-case-parameters", mutate: func(_ *fakeSpecification, cases *[]Case) { (*cases)[0].Parameters["secret"] = []string{"value"} }},
		{name: "missing parameter", code: "invalid-case-parameters", mutate: func(_ *fakeSpecification, cases *[]Case) { delete((*cases)[0].Parameters, "corp_code") }},
		{name: "untrusted server", code: "untrusted-server", mutate: func(spec *fakeSpecification, _ *[]Case) { spec.catalog.Servers = []string{"https://proxy.invalid"} }},
		{name: "dot segment", code: "invalid-operation-identity", mutate: func(spec *fakeSpecification, cases *[]Case) {
			spec.catalog.Operations[0].Path = "/../outside"
			(*cases)[0].Path = "/../outside"
		}},
		{name: "backslash", code: "invalid-operation-identity", mutate: func(spec *fakeSpecification, cases *[]Case) {
			spec.catalog.Operations[0].Path = `/company\\outside`
			(*cases)[0].Path = `/company\\outside`
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := representativeSpecification()
			cases := representativeCases()
			test.mutate(spec, &cases)
			_, err := Preflight(spec, cases, representativeAssertions())
			var conformanceError *Error
			if !errors.As(err, &conformanceError) || conformanceError.Failure.Code != test.code || spec.requestValidations > len(cases) {
				t.Fatalf("Preflight() error = %#v, request validations = %d", err, spec.requestValidations)
			}
		})
	}
}

func TestPreflightSerializesOpenAPIFormArrays(t *testing.T) {
	spec := representativeSpecification()
	spec.catalog.Operations = []openapispec.Operation{spec.catalog.Operations[0]}
	spec.catalog.Operations[0].Parameters[0] = openapispec.Parameter{
		Name: "corp_code", Location: "query", Required: true, Style: "form", Explode: false, SchemaTypes: []string{"array"},
	}
	cases := representativeCases()[:1]
	cases[0].Parameters["corp_code"] = []string{"00334624", "00126380"}
	if _, err := Preflight(spec, cases, representativeAssertions()); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(spec.lastQuery["corp_code"], []string{"00334624,00126380"}) {
		t.Fatalf("serialized query = %#v", spec.lastQuery)
	}
}

func TestDiscoveryIsBudgetedReusableAndDistinctFromPrimaryFailure(t *testing.T) {
	spec := representativeSpecification()
	listParameters := []openapispec.Parameter{
		{Name: "bgn_de", Location: "query", SchemaTypes: []string{"string"}},
		{Name: "end_de", Location: "query", SchemaTypes: []string{"string"}},
		{Name: "pblntf_detail_ty", Location: "query", SchemaTypes: []string{"string"}},
		{Name: "page_no", Location: "query", SchemaTypes: []string{"string"}},
		{Name: "page_count", Location: "query", SchemaTypes: []string{"string"}},
	}
	spec.catalog.Operations = []openapispec.Operation{
		spec.catalog.Operations[0],
		{Method: http.MethodGet, Path: "/list.json", OperationID: "get_list_json", LogicalOperationID: "DS001-2019001", Servers: []string{TrustedServer}, SecurityRequirements: []string{"crtfcKey"}, Parameters: listParameters, PrimaryRepresentation: "application/json"},
	}
	cases := []Case{
		{ID: "company-json", Method: http.MethodGet, Path: "/company.json", Representation: "application/json", Parameters: parameters("corp_code", samsungCorpCode), Assertion: "company-discovered", Discovery: "event"},
		{ID: "list-json", Method: http.MethodGet, Path: "/list.json", Representation: "application/json", Parameters: parameters("bgn_de", "20240101", "end_de", "20240331", "pblntf_detail_ty", "B001", "page_no", "1", "page_count", "100"), Assertion: "list-content"},
	}
	assertions := map[AssertionID]Assertion{
		"company-discovered": {Representations: []string{"application/json"}, Check: func(response Response) (ComparisonEvidence, bool) {
			matched := exactFieldCount(response, "", "corp_code", "00999999")
			return ComparisonEvidence{Kind: "discovered-company", Count: matched}, matched > 0
		}},
		"list-content": {Representations: []string{"application/json"}, Check: discoveryEnvelopeAssertion},
	}
	discovery := Discovery{ID: "event", MaxRequests: 1,
		Requests: []DiscoveryRequest{{ID: "event-page", Parameters: parameters("bgn_de", "20240101", "end_de", "20240331", "pblntf_detail_ty", "B001", "page_no", "1", "page_count", "100")}},
		Targets:  []DiscoveryTarget{{CaseID: "company-json", DetailTypes: []string{"B001"}, Aliases: []string{"감자 결정"}}},
	}
	plan, err := Preflight(spec, cases, assertions, discovery)
	if err != nil {
		t.Fatal(err)
	}
	attempt := 0
	report, err := execute(context.Background(), plan, dependencies{
		credential: func() (string, error) { return testCredential, nil }, now: time.Now,
		wait: func(time.Duration) error { return nil },
		do: func(request *http.Request) (*http.Response, error) {
			attempt++
			if attempt == 1 {
				return response("application/json", []byte(`{"status":"000","page_no":"1","total_page":"1","list":[{"corp_code":"00999999","report_nm":"주요사항보고서(감자 결정)","rcept_dt":"20240201"}]}`)), nil
			}
			if request.URL.Path == "/api/company.json" {
				if request.URL.Query().Get("corp_code") != "00999999" {
					t.Fatalf("resolved query = %s", request.URL.RawQuery)
				}
				return response("application/json", []byte(`{"status":"000","list":[{"corp_code":"00999999","rcept_no":"20240201000001"}]}`)), nil
			}
			return response("application/json", []byte(`{"status":"000","page_no":"1","total_page":"1","list":[{"corp_code":"00999999"}]}`)), nil
		},
	})
	if err != nil || report.Outcome != "passed" || report.RequestBudget != (RequestBudget{Ceiling: 3, Used: 3, DiscoveryCeiling: 1, DiscoveryUsed: 1}) {
		t.Fatalf("report = %#v, error = %v", report, err)
	}

	discovery.Targets[0].Aliases = []string{"없는 공시"}
	plan, err = Preflight(spec, cases, assertions, discovery)
	if err != nil {
		t.Fatal(err)
	}
	report, err = execute(context.Background(), plan, dependencies{
		credential: func() (string, error) { return testCredential, nil }, now: time.Now,
		wait: func(time.Duration) error { return nil },
		do: func(*http.Request) (*http.Response, error) {
			return response("application/json", []byte(`{"status":"000","page_no":"1","total_page":"1","list":[{"corp_code":"00999999","report_nm":"주요사항보고서(감자 결정)","rcept_dt":"20240201"}]}`)), nil
		},
	})
	if err == nil || report.Failure == nil || report.Failure.Code != "discovery-incomplete" || report.Failure.DiscoveryID != "event" || len(report.Cases) != 0 {
		t.Fatalf("report = %#v, error = %v", report, err)
	}
	encoded, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if _, decodeErr := DecodeReport(bytes.NewReader(encoded)); decodeErr != nil {
		t.Fatalf("discovery failure did not round-trip: %v", decodeErr)
	}
}

func TestExecutionMakesOneAttemptAndSanitizesFailure(t *testing.T) {
	plan, err := Preflight(representativeSpecification(), representativeCases(), representativeAssertions())
	if err != nil {
		t.Fatal(err)
	}
	attempts := 0
	responseClosed := false
	report, err := execute(context.Background(), plan, dependencies{
		credential: func() (string, error) { return testCredential, nil },
		now:        time.Now,
		wait:       func(time.Duration) error { return nil },
		do: func(*http.Request) (*http.Response, error) {
			attempts++
			return &http.Response{Body: &recordingReadCloser{Reader: bytes.NewReader(nil), closed: &responseClosed}}, errors.New("arbitrary transport error containing " + testCredential)
		},
	})
	var conformanceError *Error
	if !errors.As(err, &conformanceError) || attempts != 1 || !responseClosed || report.RequestBudget.Used != 1 || report.Failure == nil || report.Failure.Code != "transport-failure" {
		t.Fatalf("attempts=%d responseClosed=%t report=%#v error=%v", attempts, responseClosed, report, err)
	}
	encoded, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if bytes.Contains(encoded, []byte(testCredential)) || bytes.Contains(encoded, []byte("arbitrary transport")) {
		t.Fatalf("failure report leaked unsafe diagnostics: %s", encoded)
	}
	if _, decodeErr := DecodeReport(bytes.NewReader(encoded)); decodeErr != nil {
		t.Fatalf("producer failure did not round-trip through strict decoder: %v", decodeErr)
	}
}

func TestExecutionUsesFixedFailureWhenAllowlistedReportValidationFails(t *testing.T) {
	spec := representativeSpecification()
	cases := representativeCases()
	cases = []Case{cases[0]}
	cases[0].ID = testCredential
	spec.catalog.Operations = []openapispec.Operation{spec.catalog.Operations[0]}
	plan, err := Preflight(spec, cases, representativeAssertions())
	if err != nil {
		t.Fatal(err)
	}
	report, err := execute(context.Background(), plan, dependencies{
		credential: func() (string, error) { return testCredential, nil },
		now:        time.Now,
		wait:       func(time.Duration) error { return nil },
		do: func(*http.Request) (*http.Response, error) {
			return response("application/json", []byte(`{"status":"000","corp_code":"00126380"}`)), nil
		},
	})
	var conformanceError *Error
	if !errors.As(err, &conformanceError) || report.Failure == nil || report.Failure.Code != "report-sanitization" || report.RequestBudget != (RequestBudget{}) || len(report.Cases) != 0 {
		t.Fatalf("report=%#v error=%v", report, err)
	}
	encoded, marshalErr := json.Marshal(report)
	if marshalErr != nil {
		t.Fatal(marshalErr)
	}
	if bytes.Contains(encoded, []byte(testCredential)) {
		t.Fatalf("fixed failure leaked credential: %s", encoded)
	}
}

func TestXMLAndZIPParsingRequireReviewedContent(t *testing.T) {
	parsed, err := parseXML([]byte(`<result><status>010</status><nested><status>000</status></nested></result>`))
	if err != nil || parsed.APIStatus != "010" {
		t.Fatalf("parsed=%#v error=%v", parsed, err)
	}
	if _, err := parseZIP(zipBodyWithContent(t, `<document>`)); err == nil {
		t.Fatal("malformed XML archive passed")
	}
	if _, err := parseZIPWithLimit(zipBody(t), 1); err == nil {
		t.Fatal("archive above expansion limit passed")
	}
	deepXML := strings.Repeat("<node>", maximumXMLDepth+1) + strings.Repeat("</node>", maximumXMLDepth+1)
	if _, _, err := parseXMLValues([]byte(deepXML)); err == nil {
		t.Fatal("deeply nested XML passed")
	}
}

func TestExecutionRejectsOversizedBodyAndZIPXMLAlternate(t *testing.T) {
	tests := []struct {
		name        string
		contentType string
		body        []byte
		code        string
	}{
		{name: "oversized", contentType: "application/json", body: bytes.Repeat([]byte("x"), MaximumBodyBytes+1), code: "bounded-body-failure"},
		{name: "unsupported media", contentType: "text/html; charset=utf-8", body: []byte(`<html><body>upstream error</body></html>`), code: "invalid-media-type"},
		{name: "ZIP XML API error", contentType: "application/xml", body: []byte(`<result><status>010</status><message>bad key</message></result>`), code: "unsuccessful-envelope"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			spec := representativeSpecification()
			cases := representativeCases()
			cases = []Case{cases[2]}
			spec.catalog.Operations = []openapispec.Operation{spec.catalog.Operations[2]}
			plan, err := Preflight(spec, cases, representativeAssertions())
			if err != nil {
				t.Fatal(err)
			}
			report, err := execute(context.Background(), plan, dependencies{
				credential: func() (string, error) { return testCredential, nil },
				now:        time.Now,
				wait:       func(time.Duration) error { return nil },
				do:         func(*http.Request) (*http.Response, error) { return response(test.contentType, test.body), nil },
			})
			var conformanceError *Error
			if !errors.As(err, &conformanceError) || report.Failure == nil || report.Failure.Code != test.code {
				t.Fatalf("report=%#v error=%v", report, err)
			}
			encoded, marshalErr := json.Marshal(report)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if _, decodeErr := DecodeReport(bytes.NewReader(encoded)); decodeErr != nil {
				t.Fatalf("producer failure did not round-trip through strict decoder: %v", decodeErr)
			}
		})
	}
}

type fakeSpecification struct {
	catalog             openapispec.OperationCatalog
	requestValidations  int
	responseValidations int
	lastQuery           url.Values
}

func representativeSpecification() *fakeSpecification {
	parameter := openapispec.Parameter{Name: "corp_code", Location: "query", Required: true, SchemaTypes: []string{"string"}}
	operations := []openapispec.Operation{
		{Method: http.MethodGet, Path: "/company.json", OperationID: "get_company_json", LogicalOperationID: "DS001-2019002", Servers: []string{TrustedServer}, SecurityRequirements: []string{"crtfcKey"}, Parameters: []openapispec.Parameter{parameter}, PrimaryRepresentation: "application/json"},
		{Method: http.MethodGet, Path: "/company.xml", OperationID: "get_company_xml", LogicalOperationID: "DS001-2019002", Servers: []string{TrustedServer}, SecurityRequirements: []string{"crtfcKey"}, Parameters: []openapispec.Parameter{parameter}, PrimaryRepresentation: "application/xml"},
		{Method: http.MethodGet, Path: "/document.xml", OperationID: "get_document_xml", LogicalOperationID: "DS001-2019003", Servers: []string{TrustedServer}, SecurityRequirements: []string{"crtfcKey"}, Parameters: []openapispec.Parameter{{Name: "rcept_no", Location: "query", Required: true, SchemaTypes: []string{"string"}}}, PrimaryRepresentation: "application/zip", AlternateResponseMedia: []string{"application/xml"}},
	}
	return &fakeSpecification{catalog: openapispec.OperationCatalog{
		Servers:         []string{TrustedServer},
		SecuritySchemes: []openapispec.SecurityScheme{{Name: "crtfcKey", Type: "apiKey", Location: "query", ParameterName: "crtfc_key"}},
		Operations:      operations,
	}}
}

func (s *fakeSpecification) Operations() (openapispec.OperationCatalog, error) {
	return s.catalog, nil
}

func (s *fakeSpecification) ValidateRequest(_ string, _ string, query url.Values) error {
	s.requestValidations++
	s.lastQuery = cloneValues(query)
	if len(query) == 0 {
		return errors.New("query is empty")
	}
	return nil
}

func (s *fakeSpecification) ValidateResponse(_, _ string, contentType string, _ int, body []byte) error {
	s.responseValidations++
	if contentType == "" || len(body) == 0 {
		return errors.New("response is empty")
	}
	return nil
}

func representativeCases() []Case {
	return []Case{
		{ID: "company-json", Method: http.MethodGet, Path: "/company.json", Representation: "application/json", Parameters: map[string][]string{"corp_code": {"00126380"}}, Assertion: "company-json-identity"},
		{ID: "company-xml", Method: http.MethodGet, Path: "/company.xml", Representation: "application/xml", Parameters: map[string][]string{"corp_code": {"00126380"}}, Assertion: "company-xml-identity"},
		{ID: "document-zip", Method: http.MethodGet, Path: "/document.xml", Representation: "application/zip", Parameters: map[string][]string{"rcept_no": {"20240101000001"}}, Assertion: "document-zip-content"},
	}
}

func representativeAssertions() map[AssertionID]Assertion {
	return map[AssertionID]Assertion{
		"company-json-identity": {
			Representations: []string{"application/json"},
			Check: func(response Response) (ComparisonEvidence, bool) {
				value, ok := response.JSON["corp_code"].(string)
				return ComparisonEvidence{Kind: "company-identity", Count: boolCount(ok && value == "00126380")}, ok && value == "00126380"
			},
		},
		"company-xml-identity": {
			Representations: []string{"application/xml"},
			Check: func(response Response) (ComparisonEvidence, bool) {
				value := firstElementValue(response.XMLValues, "result/corp_code")
				return ComparisonEvidence{Kind: "company-identity", Count: boolCount(value == "00126380")}, value == "00126380"
			},
		},
		"document-zip-content": {
			Representations: []string{"application/zip"},
			Check: func(response Response) (ComparisonEvidence, bool) {
				matched := len(response.ArchiveDocuments) == 1 && response.ArchiveDocuments[0].Root == "document" && firstElementValue(response.ArchiveDocuments[0].XMLValues, "document/id") == "one"
				return ComparisonEvidence{Kind: "archive-document-identity", Count: boolCount(matched)}, matched
			},
		},
	}
}

func response(contentType string, body []byte) *http.Response {
	return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(bytes.NewReader(body))}
}

func zipBody(t *testing.T) []byte {
	return zipBodyWithContent(t, `<document><id>one</id></document>`)
}

func zipBodyWithContent(t *testing.T, content string) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	entry, err := writer.Create("document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := io.WriteString(entry, content); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

type recordingReadCloser struct {
	io.Reader
	closed *bool
}

func (reader *recordingReadCloser) Close() error {
	*reader.closed = true
	return nil
}

func boolCount(value bool) int {
	if value {
		return 1
	}
	return 0
}
