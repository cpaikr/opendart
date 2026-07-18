package auditorprobe

import (
	"archive/zip"
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const testKey = "kkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkkk"

func TestRunUsesFixedSequentialMatrixPaginationAndBudget(t *testing.T) {
	var requests []*http.Request
	validator := &recordingValidator{}
	firmByReceipt := map[string]string{
		"20160330000001": "다산회계법인",
		"20210330000002": "삼정회계법인",
		"20260330000003": "한울회계법인",
		"20250318000123": "안진회계법인",
	}
	client := &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		requests = append(requests, request.Clone(request.Context()))
		if request.Header.Get("User-Agent") != probeUserAgent {
			t.Fatalf("User-Agent = %q", request.Header.Get("User-Agent"))
		}
		path := request.URL.Path
		var contentType, body string
		switch path {
		case "/api/accnutAdtorNmNdAdtOpinion.json", "/api/adtServcCnclsSttus.json", "/api/accnutAdtorNonAdtServcCnclsSttus.json":
			contentType = "application/json;charset=UTF-8"
			corpCode := request.URL.Query().Get("corp_code")
			year := request.URL.Query().Get("bsns_year")
			if corpCode == nuga.CorpCode {
				body = `{"status":"013","message":"no data"}`
			} else if corpCode == lotte.CorpCode {
				body = `{"status":"000","list":[{"rcept_no":"20250318000123","adtor":"-","adt_opinion":"적정"}]}`
			} else {
				body = `{"status":"000","list":[{"rcept_no":"` + year + `0318000123","adtor":"감사법인","adt_opinion":"적정"}]}`
			}
		case "/api/list.json":
			contentType = "application/json;charset=UTF-8"
			query := request.URL.Query()
			if query.Get("corp_code") == nuga.CorpCode && query.Get("pblntf_detail_ty") == "F001" {
				if query.Get("page_no") == "1" {
					body = listBody(1, 2, 3, []FilingEvidence{
						{CorpCode: nuga.CorpCode, CorpName: nuga.Name, ReportName: "감사보고서 (2015.12)", ReceiptNumber: "20160330000001", FilerName: "다산회계법인", ReceiptDate: "20160330"},
						{CorpCode: nuga.CorpCode, CorpName: nuga.Name, ReportName: "감사보고서 (2020.12)", ReceiptNumber: "20210330000002", FilerName: "삼정회계법인", ReceiptDate: "20210330"},
					})
				} else {
					body = listBody(2, 2, 3, []FilingEvidence{{CorpCode: nuga.CorpCode, CorpName: nuga.Name, ReportName: "감사보고서 (2025.12)", ReceiptNumber: "20260330000003", FilerName: "한울회계법인", ReceiptDate: "20260330"}})
				}
			} else {
				body = `{"status":"013","message":"no data"}`
			}
		case "/api/document.xml":
			contentType = "application/x-msdownload;charset=UTF-8"
			receipt := request.URL.Query().Get("rcept_no")
			body = string(zipFixture(t, map[string][]byte{"document.xml": []byte("<p>독립된 감사인의 감사보고서 " + firmByReceipt[receipt] + "</p>")}))
		default:
			t.Fatalf("unexpected path %q", path)
		}
		return &http.Response{
			StatusCode: http.StatusOK,
			Header:     http.Header{"Content-Type": []string{contentType}},
			Body:       io.NopCloser(strings.NewReader(body)),
			Request:    request,
		}, nil
	})}
	var waits []time.Duration
	report, err := run(context.Background(), dependencies{
		client: client, validate: validator,
		now:  func() time.Time { return time.Date(2026, 7, 18, 18, 23, 45, 123456789, time.UTC) },
		wait: func(_ context.Context, duration time.Duration) error { waits = append(waits, duration); return nil },
		key:  testKey, origin: apiOrigin,
	})
	if err != nil {
		t.Fatalf("run() error = %v", err)
	}
	if len(requests) != 25 || len(validator.calls) != 21 || len(waits) != 25 {
		t.Fatalf("requests = %d, validations = %d, waits = %d", len(requests), len(validator.calls), len(waits))
	}
	for index, probeCase := range structuredCases {
		request := requests[index]
		if request.URL.Path != "/api/"+probeCase.Endpoint+".json" || request.URL.Query().Get("corp_code") != probeCase.Company.CorpCode || request.URL.Query().Get("bsns_year") != probeCase.BusinessYear {
			t.Fatalf("structured request[%d] = %s", index, request.URL.String())
		}
	}
	firstSearch := requests[len(structuredCases)]
	secondSearchPage := requests[len(structuredCases)+1]
	thirdSearch := requests[len(structuredCases)+2]
	if firstSearch.URL.Query().Get("page_no") != "1" || secondSearchPage.URL.Query().Get("page_no") != "2" || thirdSearch.URL.Query().Get("page_no") != "1" || thirdSearch.URL.Query().Get("pblntf_detail_ty") != "F002" {
		t.Fatalf("search request order was not page-complete and sequential")
	}
	for _, request := range requests[len(structuredCases) : len(structuredCases)+10] {
		query := request.URL.Query()
		if query.Get("bgn_de") != searchBeginDate || query.Get("end_de") != searchEndDate || query.Get("last_reprt_at") != "N" || query.Get("pblntf_ty") != "F" || query.Get("page_count") != "100" {
			t.Fatalf("search coordinate drifted: %s", request.URL.String())
		}
	}
	wantReceipts := []string{"20160330000001", "20210330000002", "20260330000003", "20250318000123"}
	for index, request := range requests[len(requests)-4:] {
		if request.URL.Path != "/api/document.xml" || request.URL.Query().Get("rcept_no") != wantReceipts[index] {
			t.Fatalf("document request[%d] = %s", index, request.URL.String())
		}
	}
	if report.SchemaVersion != 1 || report.ObservedAt != "2026-07-18T18:23:45.123Z" || report.ObservedDate != "2026-07-19" || report.CredentialSource != apiKeyEnvironment || report.CredentialPersisted || report.RequestBudget != (RequestBudget{Maximum: 60, Used: 25}) {
		t.Fatalf("report header = %#v", report)
	}
	if len(report.Structured) != 11 || len(report.Searches) != 9 || len(report.Documents) != 4 {
		t.Fatalf("report section lengths = %d, %d, %d", len(report.Structured), len(report.Searches), len(report.Documents))
	}
	for _, document := range report.Documents {
		if document.ExpectedMatches != 1 || document.SectionMarkers != 1 || document.EntryCount != 1 || stringValue(document.Response.MediaType) != "application/x-msdownload" {
			t.Fatalf("document evidence = %#v", document)
		}
	}
	contains, err := reportContainsCredential(report, testKey)
	if err != nil || contains {
		t.Fatalf("credential check = %v, %v", contains, err)
	}
}

func TestStructuredStatusAndPlaceholdersAreEvidence(t *testing.T) {
	tests := []struct {
		name             string
		body             string
		wantStatus       string
		wantRows         int
		wantPlaceholders int
	}{
		{name: "no data", body: `{"status":"013"}`, wantStatus: "013"},
		{name: "placeholder row", body: `{"status":"000","list":[{"rcept_no":"20250318000123","adtor":"-","adt_opinion":""}]}`, wantStatus: "000", wantRows: 1, wantPlaceholders: 2},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			observation, err := observeStructured(context.Background(), dependencies{
				client: responseClient("application/json", []byte(test.body)), validate: &recordingValidator{},
				wait: func(context.Context, time.Duration) error { return nil }, key: testKey, origin: apiOrigin,
			}, &requestBudget{maximum: 1}, structuredCases[7])
			if err != nil {
				t.Fatal(err)
			}
			if stringValue(observation.Response.APIStatus) != test.wantStatus || observation.RowCount != test.wantRows || observation.PlaceholderCount != test.wantPlaceholders {
				t.Fatalf("observation = %#v", observation)
			}
		})
	}
}

func TestCredentialInspectionHandlesJSONEscaping(t *testing.T) {
	key := strings.Repeat("k", 38) + `\"`
	report := Report{Structured: []StructuredObservation{{
		Request: RequestCoordinate{CompanyName: key},
	}}}
	contains, err := reportContainsCredential(report, key)
	if err != nil || !contains {
		t.Fatalf("credential check = %v, %v", contains, err)
	}
}

func TestRunRejectsForbiddenRequestMaterialBeforeEmission(t *testing.T) {
	report := Report{Structured: []StructuredObservation{{
		Request: RequestCoordinate{CompanyName: "https://unexpected.example"},
	}}}
	contains, err := reportContainsForbiddenRequestMaterial(report)
	if err != nil || !contains {
		t.Fatalf("request-material check = %v, %v", contains, err)
	}
}

func TestArchiveInspectionLimitsAndDecoding(t *testing.T) {
	t.Run("utf8 and cp949", func(t *testing.T) {
		cp949, _, err := transform.Bytes(korean.EUCKR.NewEncoder(), []byte("독립된 감사인의 감사보고서 삼정회계법인"))
		if err != nil {
			t.Fatal(err)
		}
		archive := zipFixture(t, map[string][]byte{
			"utf8.xml":   []byte("안진회계법인"),
			"legacy.xml": cp949,
		})
		evidence, err := inspectArchive(archive, "삼정회계법인")
		if err != nil {
			t.Fatal(err)
		}
		decodings := []string{evidence.entries[0].Decoding, evidence.entries[1].Decoding}
		slices.Sort(decodings)
		if !slices.Equal(decodings, []string{"cp949-compatible", "utf-8"}) || evidence.expectedMatches != 1 || evidence.sectionMarkers != 1 {
			t.Fatalf("archive evidence = %#v", evidence)
		}
	})

	t.Run("entry count", func(t *testing.T) {
		entries := make(map[string][]byte, maxArchiveEntries+1)
		for index := 0; index <= maxArchiveEntries; index++ {
			entries[strconv.Itoa(index)+".xml"] = []byte("x")
		}
		_, err := inspectArchive(zipFixture(t, entries), "firm")
		if err == nil || !strings.Contains(err.Error(), "entry-count") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("member size", func(t *testing.T) {
		archive := zipFixture(t, map[string][]byte{"large.xml": bytes.Repeat([]byte("x"), maxArchiveMember+1)})
		_, err := inspectArchive(archive, "firm")
		if err == nil || !strings.Contains(err.Error(), "expanded-size") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestDocumentSelectionIsDeterministic(t *testing.T) {
	filings := []FilingEvidence{
		{ReportName: "감사보고서 (2020.12)", ReceiptDate: "20210329", ReceiptNumber: "20210329000009"},
		{ReportName: "[기재정정] 감사보고서 (2020.12)", ReceiptDate: "20210401", ReceiptNumber: "20210401000001"},
		{ReportName: "연결감사보고서 (2020.12)", ReceiptDate: "20210401", ReceiptNumber: "20210401000002"},
		{ReportName: "감사보고서 (2019.12)", ReceiptDate: "20200330", ReceiptNumber: "20200330000001"},
	}
	selected, ok := selectPeriodFiling(filings, "2020")
	if !ok || selected.ReceiptNumber != "20210401000002" {
		t.Fatalf("selected = %#v, %v", selected, ok)
	}
	slices.Reverse(filings)
	selectedAgain, ok := selectPeriodFiling(filings, "2020")
	if !ok || selectedAgain != selected {
		t.Fatalf("selection depended on input order: %#v", selectedAgain)
	}
}

func TestDocumentObservationRequiresFirmAndAuditorSectionEvidence(t *testing.T) {
	tests := []struct {
		name        string
		content     string
		wantMessage string
	}{
		{name: "missing firm", content: "독립된 감사인의 감사보고서 다른회계법인", wantMessage: "expected accounting firm"},
		{name: "missing section", content: "안진회계법인", wantMessage: "section marker"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			archive := zipFixture(t, map[string][]byte{"document.xml": []byte(test.content)})
			_, err := observeDocument(context.Background(), dependencies{
				client: responseClient("application/zip", archive), validate: &recordingValidator{},
				key: testKey, origin: apiOrigin,
			}, &requestBudget{maximum: 1}, DocumentSelection{
				CompanyName: lotte.Name, ReportPeriod: "2024", ReceiptNumber: "20250318000123", ExpectedFirm: "안진회계법인",
			})
			if err == nil || !strings.Contains(err.Error(), test.wantMessage) {
				t.Fatalf("error = %v", err)
			}
		})
	}
}

func TestDocumentObservationRequiresCoLocatedFirmAndSectionEvidence(t *testing.T) {
	archive := zipFixture(t, map[string][]byte{
		"firm.xml":    []byte("안진회계법인"),
		"section.xml": []byte("독립된 감사인의 감사보고서"),
	})
	_, err := observeDocument(context.Background(), dependencies{
		client: responseClient("application/x-msdownload", archive), validate: &recordingValidator{},
		key: testKey, origin: apiOrigin,
	}, &requestBudget{maximum: 1}, DocumentSelection{
		CompanyName: lotte.Name, ReportPeriod: "2024", ReceiptNumber: "20250318000123", ExpectedFirm: "안진회계법인",
	})
	if err == nil || !strings.Contains(err.Error(), "same member") {
		t.Fatalf("error = %v", err)
	}
}

func TestDocumentObservationRejectsUnexpectedMediaType(t *testing.T) {
	archive := zipFixture(t, map[string][]byte{
		"document.xml": []byte("독립된 감사인의 감사보고서 안진회계법인"),
	})
	_, err := observeDocument(context.Background(), dependencies{
		client: responseClient("text/plain", archive), validate: &recordingValidator{},
		key: testKey, origin: apiOrigin,
	}, &requestBudget{maximum: 1}, DocumentSelection{
		CompanyName: lotte.Name, ReportPeriod: "2024", ReceiptNumber: "20250318000123", ExpectedFirm: "안진회계법인",
	})
	if err == nil || !strings.Contains(err.Error(), "media type") {
		t.Fatalf("error = %v", err)
	}
}

func TestDocumentObservationRejectsTaintedReceiptBeforeRequest(t *testing.T) {
	calls := 0
	client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
		calls++
		return nil, errors.New("request should not be sent")
	})}
	_, err := observeDocument(context.Background(), dependencies{
		client: client, validate: &recordingValidator{}, key: testKey, origin: apiOrigin,
	}, &requestBudget{maximum: 1}, DocumentSelection{
		CompanyName: lotte.Name, ReportPeriod: "2024", ReceiptNumber: testKey, ExpectedFirm: "안진회계법인",
	})
	var probeError *Error
	if calls != 0 || !errors.As(err, &probeError) || probeError.Request != nil || strings.Contains(err.Error(), testKey) {
		t.Fatalf("calls = %d, error = %#v", calls, err)
	}
}

func TestPaginationCeilingFailsClosed(t *testing.T) {
	client := responseClient("application/json", []byte(listBody(1, maxSearchPages+1, 0, nil)))
	_, err := observeSearch(context.Background(), dependencies{
		client: client, validate: &recordingValidator{}, wait: func(context.Context, time.Duration) error { return nil }, key: testKey, origin: apiOrigin,
	}, &requestBudget{maximum: maximumRequestBudget}, searchCases[0])
	if err == nil || !strings.Contains(err.Error(), "page ceiling") {
		t.Fatalf("error = %v", err)
	}
}

func TestFailuresAndReportsDoNotExposeCredentialsOrUpstreamDetails(t *testing.T) {
	t.Run("transport", func(t *testing.T) {
		calls := 0
		client := &http.Client{Transport: roundTripFunc(func(*http.Request) (*http.Response, error) {
			calls++
			return nil, errors.New("authenticated URL contained " + testKey)
		})}
		_, err := observeStructured(context.Background(), dependencies{client: client, validate: &recordingValidator{}, key: testKey, origin: apiOrigin}, &requestBudget{maximum: 1}, structuredCases[0])
		if calls != 1 || err == nil || strings.Contains(err.Error(), testKey) || !strings.Contains(err.Error(), "before a response") {
			t.Fatalf("calls = %d, error = %v", calls, err)
		}
	})

	t.Run("validator", func(t *testing.T) {
		const secret = "raw validation secret"
		_, err := observeStructured(context.Background(), dependencies{
			client:   responseClient("application/json", []byte(`{"status":"013","secret":"`+secret+`"}`)),
			validate: &recordingValidator{err: errors.New(secret + testKey)}, key: testKey, origin: apiOrigin,
		}, &requestBudget{maximum: 1}, structuredCases[0])
		if err == nil || strings.Contains(err.Error(), secret) || strings.Contains(err.Error(), testKey) || !strings.Contains(err.Error(), "committed OpenAPI") {
			t.Fatalf("error = %v", err)
		}
	})
}

func TestFixturesSatisfyCommittedOpenAPI(t *testing.T) {
	document, err := openapispec.Load(filepath.Join("..", "..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	fixtures := []struct {
		path, media string
		body        []byte
	}{
		{"/accnutAdtorNmNdAdtOpinion.json", "application/json;charset=UTF-8", []byte(`{"status":"013"}`)},
		{"/adtServcCnclsSttus.json", "application/json;charset=UTF-8", []byte(`{"status":"013"}`)},
		{"/accnutAdtorNonAdtServcCnclsSttus.json", "application/json;charset=UTF-8", []byte(`{"status":"013"}`)},
		{"/list.json", "application/json;charset=UTF-8", []byte(listBody(1, 1, 0, nil))},
		{"/document.xml", "application/zip", zipFixture(t, map[string][]byte{"document.xml": []byte("text")})},
	}
	for _, fixture := range fixtures {
		if err := document.ValidateResponse(http.MethodGet, fixture.path, fixture.media, http.StatusOK, fixture.body); err != nil {
			t.Fatalf("validate %s fixture: %v", fixture.path, err)
		}
	}
}

func TestHTTPAndCredentialSafetyBoundaries(t *testing.T) {
	t.Setenv(apiKeyEnvironment, "short")
	if _, err := apiKey(); err == nil || !strings.Contains(err.Error(), "40-character") {
		t.Fatalf("apiKey error = %v", err)
	}
	t.Setenv(apiKeyEnvironment, testKey)
	if value, err := apiKey(); err != nil || value != testKey {
		t.Fatalf("apiKey = %q, %v", value, err)
	}

	client := newHTTPClient()
	if client.Timeout != requestTimeout {
		t.Fatalf("timeout = %s", client.Timeout)
	}
	if err := client.CheckRedirect(&http.Request{}, nil); err == nil {
		t.Fatal("redirect was accepted")
	}
	transport, ok := client.Transport.(*http.Transport)
	if !ok || !transport.DisableKeepAlives || transport.ForceAttemptHTTP2 || transport.TLSNextProto == nil {
		t.Fatalf("transport does not enforce one fresh HTTP/1 attempt: %#v", client.Transport)
	}
}

func listBody(page, totalPages, totalCount int, filings []FilingEvidence) string {
	rows := make([]string, 0, len(filings))
	for _, filing := range filings {
		encoded, err := jsonMarshal(filing)
		if err != nil {
			panic(err)
		}
		rows = append(rows, string(encoded))
	}
	return `{"status":"000","page_no":` + strconv.Itoa(page) + `,"page_count":100,"total_count":` + strconv.Itoa(totalCount) + `,"total_page":` + strconv.Itoa(totalPages) + `,"list":[` + strings.Join(rows, ",") + `]}`
}

func jsonMarshal(filing FilingEvidence) ([]byte, error) {
	// Keep the fixture keys aligned with the OpenDART wire representation.
	return []byte(`{"corp_code":"` + filing.CorpCode + `","corp_name":"` + filing.CorpName + `","report_nm":"` + filing.ReportName + `","rcept_no":"` + filing.ReceiptNumber + `","flr_nm":"` + filing.FilerName + `","rcept_dt":"` + filing.ReceiptDate + `","rm":"` + filing.Remarks + `"}`), nil
}

func zipFixture(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	slices.Sort(names)
	for _, name := range names {
		member, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := member.Write(entries[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func responseClient(contentType string, body []byte) *http.Client {
	return &http.Client{Transport: roundTripFunc(func(request *http.Request) (*http.Response, error) {
		return &http.Response{StatusCode: http.StatusOK, Header: http.Header{"Content-Type": []string{contentType}}, Body: io.NopCloser(bytes.NewReader(body)), Request: request}, nil
	})}
}

type roundTripFunc func(*http.Request) (*http.Response, error)

func (function roundTripFunc) RoundTrip(request *http.Request) (*http.Response, error) {
	return function(request)
}

type validationCall struct {
	path, contentType string
	status            int
	bodyBytes         int
}

type recordingValidator struct {
	calls []validationCall
	err   error
}

func (validator *recordingValidator) ValidateResponse(_ string, path, contentType string, status int, body []byte) error {
	validator.calls = append(validator.calls, validationCall{path, contentType, status, len(body)})
	return validator.err
}
