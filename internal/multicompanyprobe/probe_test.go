package multicompanyprobe

import (
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

const testKey = "kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk"

func TestEncodedQueryPreservesProbeSerializations(t *testing.T) {
	testCase := cases[1]
	tests := []struct {
		name          string
		format        string
		serialization serialization
		corpCodes     []string
		want          string
	}{
		{
			name:          "canonical comma separated",
			format:        "json",
			serialization: commaSeparated,
			want:          apiOrigin + "/fnlttCmpnyIndx.json?crtfc_key=" + testKey + "&corp_code=00164742,00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000",
		},
		{
			name:          "repeated key control",
			format:        "xml",
			serialization: repeatedQueryKey,
			want:          apiOrigin + "/fnlttCmpnyIndx.xml?crtfc_key=" + testKey + "&corp_code=00164742&corp_code=00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000",
		},
		{
			name:          "single baseline",
			format:        "json",
			serialization: singleValueBaseline,
			corpCodes:     []string{"00164742"},
			want:          apiOrigin + "/fnlttCmpnyIndx.json?crtfc_key=" + testKey + "&corp_code=00164742&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			current := testCase
			if test.corpCodes != nil {
				current.CorpCodes = test.corpCodes
			}
			if got := encodedQuery(current, test.format, test.serialization, testKey); got != test.want {
				t.Fatalf("encodedQuery() = %q, want %q", got, test.want)
			}
		})
	}
}

func TestQueryEscapeMatchesEncodeURIComponent(t *testing.T) {
	if got, want := queryEscape("a b!'()*~"), "a%20b!'()*~"; got != want {
		t.Fatalf("queryEscape() = %q, want %q", got, want)
	}
}

func TestJSONObservationExtractsOnlyListIdentities(t *testing.T) {
	observation, err := jsonObservation([]byte(`{
  "status": "000",
  "metadata": {"stock_code": "999999"},
  "list": [
    {"stock_code": "005930", "account_nm": "assets"},
    12,
    {"stock_code": "000660"},
    {"stock_code": "005930"}
  ]
}`), "stock_code", nil)
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(observation.apiStatus) != "000" || !slices.Equal(observation.returnedIdentityValues, []string{"000660", "005930"}) {
		t.Fatalf("observation = %#v", observation)
	}

	observation, err = jsonObservation([]byte(`{"status":3,"list":{"stock_code":"999999"}}`), "stock_code", nil)
	if err != nil {
		t.Fatal(err)
	}
	if observation.apiStatus != nil || len(observation.returnedIdentityValues) != 0 {
		t.Fatalf("noncanonical fields were accepted: %#v", observation)
	}
}

func TestXMLObservationExtractsDescendantIdentitiesAndRejectsMalformedDocuments(t *testing.T) {
	observation, err := xmlObservation([]byte(`<result><status>000</status><list><nested><corp_code> 00164742 </corp_code></nested></list><list><corp_code>00159023</corp_code></list></result>`), "corp_code", nil)
	if err != nil {
		t.Fatal(err)
	}
	if stringValue(observation.apiStatus) != "000" || !slices.Equal(observation.returnedIdentityValues, []string{"00159023", "00164742"}) {
		t.Fatalf("observation = %#v", observation)
	}
	for _, malformed := range []string{
		`<result><status>000</status>`,
		`<result></result><second></second>`,
		``,
	} {
		if _, err := xmlObservation([]byte(malformed), "corp_code", nil); err == nil || !strings.Contains(err.Error(), "malformed XML") {
			t.Fatalf("xmlObservation(%q) error = %v", malformed, err)
		}
	}
}

func TestCanonicalAndBaselineAssertionsRejectIdentityDrift(t *testing.T) {
	jsonMedia := "application/json"
	status := "000"
	canonical := Observation{
		Request: RequestObservation{LogicalOperationID: "DS003-2022002", Endpoint: "fnlttCmpnyIndx", Format: "json"},
		Response: ResponseObservation{
			HTTPStatus:                    http.StatusOK,
			MediaType:                     &jsonMedia,
			APIStatus:                     &status,
			MissingExpectedIdentityValues: []string{},
			UnexpectedIdentityValues:      []string{"99999999"},
		},
	}
	if err := assertCanonicalObservation(canonical); err == nil || !strings.Contains(err.Error(), "did not return both") {
		t.Fatalf("assertCanonicalObservation() error = %v", err)
	}

	testCase := cases[0]
	response := ResponseObservation{HTTPStatus: http.StatusOK, MediaType: &jsonMedia, APIStatus: &status, ReturnedIdentityValues: []string{"005930"}}
	if _, err := distinctBaselineIdentityValues(testCase, []Observation{{Response: response}, {Response: response}}); err == nil || !strings.Contains(err.Error(), "distinct identities") {
		t.Fatalf("distinctBaselineIdentityValues() error = %v", err)
	}
}

func TestRunPreservesRequestOrderPacingValidationAndReport(t *testing.T) {
	var requestURLs []string
	var acceptHeaders []string
	transport := roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requestURLs = append(requestURLs, request.URL.String())
		acceptHeaders = append(acceptHeaders, request.Header.Get("Accept"))
		if request.Header.Get("User-Agent") != probeUserAgent {
			t.Fatalf("User-Agent = %q", request.Header.Get("User-Agent"))
		}
		format := "json"
		if strings.HasSuffix(request.URL.Path, ".xml") {
			format = "xml"
		}
		identities := request.URL.Query()["corp_code"]
		if len(identities) == 1 && strings.Contains(identities[0], ",") {
			identities = strings.Split(identities[0], ",")
		}
		if strings.Contains(request.URL.Path, "fnlttMultiAcnt") {
			if len(identities) == 1 {
				identities = map[string][]string{"00334624": {"005930"}, "00126380": {"000660"}}[identities[0]]
			} else {
				identities = []string{"005930", "000660"}
			}
		}
		body := successfulBody(format, map[string]string{"fnlttMultiAcnt": "stock_code", "fnlttCmpnyIndx": "corp_code"}[endpointFromPath(request.URL.Path)], identities)
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/" + format + "; charset=UTF-8"}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})
	validator := &recordingValidator{}
	var waits []time.Duration
	deps := dependencies{
		client:   &http.Client{Transport: transport},
		validate: validator,
		now:      func() time.Time { return time.Date(2026, 7, 18, 18, 23, 45, 123456789, time.UTC) },
		wait: func(_ context.Context, duration time.Duration) error {
			waits = append(waits, duration)
			return nil
		},
		key: testKey,
	}
	report, err := run(context.Background(), deps)
	if err != nil {
		t.Fatalf("run() error = %v; URLs = %#v; validations = %#v", err, requestURLs, validator.calls)
	}
	if len(requestURLs) != 10 || len(waits) != 10 || len(validator.calls) != 10 {
		t.Fatalf("requests = %d, waits = %d, validations = %d", len(requestURLs), len(waits), len(validator.calls))
	}
	for _, duration := range waits {
		if duration != 100*time.Millisecond {
			t.Fatalf("wait = %s", duration)
		}
	}
	wantURLs := []string{
		apiOrigin + "/fnlttMultiAcnt.json?crtfc_key=" + testKey + "&corp_code=00334624&bsns_year=2018&reprt_code=11011",
		apiOrigin + "/fnlttMultiAcnt.json?crtfc_key=" + testKey + "&corp_code=00126380&bsns_year=2018&reprt_code=11011",
		apiOrigin + "/fnlttMultiAcnt.json?crtfc_key=" + testKey + "&corp_code=00334624,00126380&bsns_year=2018&reprt_code=11011",
		apiOrigin + "/fnlttMultiAcnt.json?crtfc_key=" + testKey + "&corp_code=00334624&corp_code=00126380&bsns_year=2018&reprt_code=11011",
		apiOrigin + "/fnlttMultiAcnt.xml?crtfc_key=" + testKey + "&corp_code=00334624,00126380&bsns_year=2018&reprt_code=11011",
		apiOrigin + "/fnlttMultiAcnt.xml?crtfc_key=" + testKey + "&corp_code=00334624&corp_code=00126380&bsns_year=2018&reprt_code=11011",
		apiOrigin + "/fnlttCmpnyIndx.json?crtfc_key=" + testKey + "&corp_code=00164742,00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000",
		apiOrigin + "/fnlttCmpnyIndx.json?crtfc_key=" + testKey + "&corp_code=00164742&corp_code=00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000",
		apiOrigin + "/fnlttCmpnyIndx.xml?crtfc_key=" + testKey + "&corp_code=00164742,00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000",
		apiOrigin + "/fnlttCmpnyIndx.xml?crtfc_key=" + testKey + "&corp_code=00164742&corp_code=00159023&bsns_year=2023&reprt_code=11014&idx_cl_code=M210000",
	}
	if !slices.Equal(requestURLs, wantURLs) {
		t.Fatalf("request order = %#v", requestURLs)
	}
	for index, accept := range acceptHeaders {
		want := "application/json"
		if strings.Contains(requestURLs[index], ".xml?") {
			want = "application/xml"
		}
		if accept != want {
			t.Fatalf("Accept[%d] = %q, want %q", index, accept, want)
		}
	}
	if report.SchemaVersion != 1 || report.ObservedAt != "2026-07-18T18:23:45.123Z" || report.ObservedDate != "2026-07-19" || report.RequestCount != 10 ||
		len(report.Baselines) != 2 || len(report.Canonical) != 4 || len(report.RepeatedKeyControls) != 4 || report.Conclusion.RepeatedQueryKey != "accepted-in-all-controls" {
		t.Fatalf("report = %#v", report)
	}
	contains, err := reportContainsCredential(report, testKey)
	if err != nil || contains {
		t.Fatalf("report credential check = %v, %v", contains, err)
	}
}

func TestProbeFixturesSatisfyCanonicalOpenAPIResponses(t *testing.T) {
	document, err := openapispec.Load(filepath.Join("..", "..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	for _, test := range []struct {
		endpoint      string
		format        string
		identityField string
	}{
		{endpoint: "fnlttMultiAcnt", format: "json", identityField: "stock_code"},
		{endpoint: "fnlttMultiAcnt", format: "xml", identityField: "stock_code"},
		{endpoint: "fnlttCmpnyIndx", format: "json", identityField: "corp_code"},
		{endpoint: "fnlttCmpnyIndx", format: "xml", identityField: "corp_code"},
	} {
		body := []byte(successfulBody(test.format, test.identityField, []string{"00126380"}))
		if err := document.ValidateResponse(http.MethodGet, "/"+test.endpoint+"."+test.format, "application/"+test.format+"; charset=UTF-8", http.StatusOK, body); err != nil {
			t.Fatalf("validate %s.%s fixture: %v", test.endpoint, test.format, err)
		}
	}
}

func TestRunDoesNotRetryOrExposeTransportErrors(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("transport included " + testKey + " and an authenticated URL")
	})}
	_, err := run(context.Background(), dependencies{
		client: client, validate: &recordingValidator{}, now: time.Now,
		wait: waitFor, key: testKey,
	})
	if calls != 1 || err == nil || strings.Contains(err.Error(), testKey) || !strings.Contains(err.Error(), "failed before a response") {
		t.Fatalf("calls = %d, error = %v", calls, err)
	}
}

func TestRunDoesNotExposeResponseOrValidatorDetails(t *testing.T) {
	const sentinel = "unrestricted response sentinel"
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		calls++
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{"application/json"}},
			Body:       io.NopCloser(strings.NewReader(`{"status":"000","list":[],"secret":"` + sentinel + `"}`)),
			Request:    request,
		}, nil
	})}
	validator := &recordingValidator{err: errors.New("validation detail included " + sentinel)}
	_, err := run(context.Background(), dependencies{
		client: client, validate: validator, now: time.Now,
		wait: waitFor, key: testKey,
	})
	if calls != 1 || err == nil || strings.Contains(err.Error(), sentinel) || !strings.Contains(err.Error(), "committed OpenAPI representation") {
		t.Fatalf("calls = %d, error = %v", calls, err)
	}
}

func TestResponseSafetyBoundaries(t *testing.T) {
	t.Run("body size", func(t *testing.T) {
		body := io.NopCloser(io.LimitReader(zeroReader{}, maxResponseBody+1))
		if _, err := readBoundedBody(body); err == nil || !strings.Contains(err.Error(), "size limit") {
			t.Fatalf("readBoundedBody() error = %v", err)
		}
	})

	t.Run("body read error", func(t *testing.T) {
		body := &errorReadCloser{}
		if _, err := readBoundedBody(body); err == nil || !strings.Contains(err.Error(), "could not be read") {
			t.Fatalf("readBoundedBody() error = %v", err)
		}
	})

	t.Run("redirect policy", func(t *testing.T) {
		client := newHTTPClient()
		if client.Timeout != 30*time.Second {
			t.Fatalf("client timeout = %s", client.Timeout)
		}
		if err := client.CheckRedirect(&http.Request{}, nil); err == nil || !strings.Contains(err.Error(), "not allowed") {
			t.Fatal("redirect was accepted")
		}
		transport, ok := client.Transport.(*http.Transport)
		if !ok || !transport.DisableKeepAlives || transport.ForceAttemptHTTP2 || transport.TLSNextProto == nil {
			t.Fatalf("transport does not enforce a fresh HTTP/1 connection: %#v", client.Transport)
		}
	})

	t.Run("credential detection is semantic", func(t *testing.T) {
		key := strings.Repeat("<", 40)
		report := Report{Canonical: []Observation{{Response: ResponseObservation{ReturnedIdentityValues: []string{"prefix" + key + "suffix"}}}}}
		contains, err := reportContainsCredential(report, key)
		if err != nil || !contains {
			t.Fatalf("reportContainsCredential() = %v, %v", contains, err)
		}
	})
}

func TestAPIKeyRemainsEnvironmentOnly(t *testing.T) {
	t.Setenv(apiKeyEnvironment, "short")
	if _, err := apiKey(); err == nil || !strings.Contains(err.Error(), "40-character") {
		t.Fatalf("apiKey() error = %v", err)
	}
	t.Setenv(apiKeyEnvironment, testKey)
	if key, err := apiKey(); err != nil || key != testKey {
		t.Fatalf("apiKey() = %q, %v", key, err)
	}
}

func successfulBody(format, identityField string, identities []string) string {
	if format == "json" {
		rows := make([]string, 0, len(identities))
		for _, identity := range identities {
			rows = append(rows, `{"`+identityField+`":"`+identity+`"}`)
		}
		return `{"status":"000","list":[` + strings.Join(rows, ",") + `]}`
	}
	rows := make([]string, 0, len(identities))
	for _, identity := range identities {
		rows = append(rows, "<list><"+identityField+">"+identity+"</"+identityField+"></list>")
	}
	return "<result><status>000</status>" + strings.Join(rows, "") + "</result>"
}

func endpointFromPath(path string) string {
	name := strings.TrimPrefix(path, "/api/")
	return strings.TrimSuffix(strings.TrimSuffix(name, ".json"), ".xml")
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type validationCall struct {
	method      string
	path        string
	contentType string
	status      int
	body        string
}

type recordingValidator struct {
	calls []validationCall
	err   error
}

func (validator *recordingValidator) ValidateResponse(method, path, contentType string, status int, body []byte) error {
	validator.calls = append(validator.calls, validationCall{method: method, path: path, contentType: contentType, status: status, body: string(body)})
	return validator.err
}

type zeroReader struct{}

func (zeroReader) Read(buffer []byte) (int, error) {
	for index := range buffer {
		buffer[index] = 0
	}
	return len(buffer), nil
}

type errorReadCloser struct{}

func (*errorReadCloser) Read([]byte) (int, error) { return 0, errors.New("secret response content") }
func (*errorReadCloser) Close() error             { return nil }
