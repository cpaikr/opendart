// Package auditorprobe owns the bounded live probe that supports the external
// auditor retrieval guide. It records only allowlisted evidence and never
// persists credentials or response bodies.
package auditorprobe

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"html"
	"io"
	"mime"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
	"unicode/utf8"

	"github.com/cpaikr/opendart/internal/liveprobe"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

const (
	apiOrigin           = "https://opendart.fss.or.kr/api"
	apiKeyEnvironment   = "OPENDART_API_KEY"
	reportSchemaVersion = 1
	probeUserAgent      = "opendart-auditor-evidence-probe/1.0"
	requestTimeout      = 30 * time.Second
	requestPacing       = 100 * time.Millisecond
	maxJSONBody         = 8 << 20
	maxArchiveBody      = 32 << 20
	maxArchiveEntries   = 64
	maxArchiveMember    = 8 << 20
	maxArchiveExpanded  = 32 << 20
	maxSearchPages      = 5
	searchPageCount     = 100
	// The fixed ceiling covers the complete matrix at the search page limit.
	maximumRequestBudget   = 60
	searchBeginDate        = "20140101"
	searchEndDate          = "20260718"
	annualReportCode       = "11011"
	auditorOpinionEndpoint = "accnutAdtorNmNdAdtOpinion"
)

type company struct {
	Name     string
	CorpCode string
}

var (
	nuga        = company{Name: "누가의료기", CorpCode: "00571818"}
	bioepis     = company{Name: "삼성바이오에피스", CorpCode: "00907679"}
	display     = company{Name: "삼성디스플레이", CorpCode: "00912006"}
	electronics = company{Name: "삼성전자", CorpCode: "00126380"}
	kyobo       = company{Name: "교보생명보험", CorpCode: "00112882"}
	lotte       = company{Name: "롯데건설", CorpCode: "00120438"}
	kakao       = company{Name: "카카오모빌리티", CorpCode: "01250666"}
)

type structuredCase struct {
	Endpoint     string
	LogicalID    string
	Company      company
	BusinessYear string
	ReportCode   string
}

var structuredCases = []structuredCase{
	{auditorOpinionEndpoint, "DS002-2020009", nuga, "2015", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", nuga, "2020", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", nuga, "2024", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", bioepis, "2024", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", display, "2024", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", electronics, "2024", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", kyobo, "2024", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", lotte, "2024", annualReportCode},
	{auditorOpinionEndpoint, "DS002-2020009", kakao, "2024", annualReportCode},
	{"adtServcCnclsSttus", "DS002-2020010", nuga, "2024", annualReportCode},
	{"accnutAdtorNonAdtServcCnclsSttus", "DS002-2020011", nuga, "2024", annualReportCode},
}

type searchCase struct {
	Company    company
	DetailType string
}

var searchCases = []searchCase{
	{nuga, "F001"}, {nuga, "F002"},
	{bioepis, "F001"}, {bioepis, "F002"},
	{display, "F001"}, {display, "F002"},
	{electronics, "F001"}, {kyobo, "F001"}, {kakao, "F001"},
}

// Report is the stdout-ready, sanitized evidence manifest emitted by Run.
type Report struct {
	SchemaVersion       int                     `json:"schemaVersion"`
	ObservedAt          string                  `json:"observedAt"`
	ObservedDate        string                  `json:"observedDate"`
	CredentialSource    string                  `json:"credentialSource"`
	CredentialPersisted bool                    `json:"credentialPersisted"`
	RequestBudget       RequestBudget           `json:"requestBudget"`
	Structured          []StructuredObservation `json:"structured"`
	Searches            []SearchObservation     `json:"searches"`
	Documents           []DocumentObservation   `json:"documents"`
}

type RequestBudget struct {
	Maximum int `json:"maximum"`
	Used    int `json:"used"`
}

type RequestCoordinate struct {
	LogicalOperationID string `json:"logicalOperationId"`
	Endpoint           string `json:"endpoint"`
	CompanyName        string `json:"companyName,omitempty"`
	CorpCode           string `json:"corpCode,omitempty"`
	BusinessYear       string `json:"businessYear,omitempty"`
	ReportCode         string `json:"reportCode,omitempty"`
	BeginDate          string `json:"beginDate,omitempty"`
	EndDate            string `json:"endDate,omitempty"`
	LastReport         string `json:"lastReport,omitempty"`
	PublicationType    string `json:"publicationType,omitempty"`
	DetailType         string `json:"detailType,omitempty"`
	PageNumber         int    `json:"pageNumber,omitempty"`
	PageCount          int    `json:"pageCount,omitempty"`
	ReceiptNumber      string `json:"receiptNumber,omitempty"`
}

type ResponseEvidence struct {
	HTTPStatus int     `json:"httpStatus"`
	MediaType  *string `json:"mediaType"`
	APIStatus  *string `json:"apiStatus"`
	BodyBytes  int     `json:"bodyBytes"`
	BodySHA256 string  `json:"bodySha256"`
}

type StructuredObservation struct {
	Request          RequestCoordinate `json:"request"`
	Response         ResponseEvidence  `json:"response"`
	RowCount         int               `json:"rowCount"`
	ReceiptNumbers   []string          `json:"receiptNumbers"`
	DistinctAuditors []string          `json:"distinctAuditors"`
	DistinctOpinions []string          `json:"distinctOpinions"`
	PlaceholderCount int               `json:"placeholderCount"`
}

type SearchObservation struct {
	CompanyName string       `json:"companyName"`
	CorpCode    string       `json:"corpCode"`
	DetailType  string       `json:"detailType"`
	Pages       []SearchPage `json:"pages"`
}

type SearchPage struct {
	Request    RequestCoordinate `json:"request"`
	Response   ResponseEvidence  `json:"response"`
	PageNumber int               `json:"pageNumber"`
	PageCount  int               `json:"pageCount"`
	TotalCount int               `json:"totalCount"`
	TotalPages int               `json:"totalPages"`
	Filings    []FilingEvidence  `json:"filings"`
}

type FilingEvidence struct {
	CorpCode      string `json:"corpCode"`
	CorpName      string `json:"corpName"`
	ReportName    string `json:"reportName"`
	ReceiptNumber string `json:"receiptNumber"`
	FilerName     string `json:"filerName"`
	ReceiptDate   string `json:"receiptDate"`
	Remarks       string `json:"remarks"`
}

type DocumentObservation struct {
	Selection       DocumentSelection `json:"selection"`
	Request         RequestCoordinate `json:"request"`
	Response        ResponseEvidence  `json:"response"`
	ArchiveBytes    int               `json:"archiveBytes"`
	ArchiveSHA256   string            `json:"archiveSha256"`
	EntryCount      int               `json:"entryCount"`
	ExpandedBytes   int64             `json:"expandedBytes"`
	Entries         []DocumentEntry   `json:"entries"`
	ExpectedMatches int               `json:"expectedFirmMatches"`
	SectionMarkers  int               `json:"auditorSectionMarkerMatches"`
}

type DocumentSelection struct {
	CompanyName    string `json:"companyName"`
	ReportPeriod   string `json:"reportPeriod"`
	ReceiptNumber  string `json:"receiptNumber"`
	ExpectedFirm   string `json:"expectedFirm"`
	SelectionBasis string `json:"selectionBasis"`
}

type DocumentEntry struct {
	Name                        string `json:"name"`
	NameSHA256                  string `json:"nameSha256"`
	CompressedBytes             uint64 `json:"compressedBytes"`
	ExpandedBytes               uint64 `json:"expandedBytes"`
	ContentSHA256               string `json:"contentSha256"`
	Decoding                    string `json:"decoding"`
	ExpectedFirmMatches         int    `json:"expectedFirmMatches"`
	AuditorSectionMarkerMatches int    `json:"auditorSectionMarkerMatches"`
}

// Error is a sanitized failure containing only an allowlisted request
// coordinate. It never wraps transport, validation, or response-body errors.
type Error struct {
	Message string             `json:"message"`
	Request *RequestCoordinate `json:"request,omitempty"`
}

func (e *Error) Error() string { return e.Message }

type responseValidator interface {
	ValidateResponse(method, path, contentType string, status int, body []byte) error
}

type dependencies struct {
	client   *http.Client
	validate responseValidator
	now      func() time.Time
	wait     func(context.Context, time.Duration) error
	key      string
	origin   string
}

// Run performs the fixed evidence matrix with OPENDART_API_KEY and the
// repository's canonical OpenAPI document.
func Run(ctx context.Context, repositoryRoot string) (Report, error) {
	key, err := apiKey()
	if err != nil {
		return Report{}, err
	}
	document, err := openapispec.Load(filepath.Join(repositoryRoot, "openapi", "openapi.yaml"))
	if err != nil {
		return Report{}, newError("OpenAPI response validator could not be initialized", nil)
	}
	defer document.Close()
	return run(ctx, dependencies{
		client: newHTTPClient(), validate: document, now: time.Now,
		wait: waitFor, key: key, origin: apiOrigin,
	})
}

func run(ctx context.Context, deps dependencies) (Report, error) {
	if deps.origin == "" {
		deps.origin = apiOrigin
	}
	budget := &requestBudget{maximum: maximumRequestBudget}
	structured := make([]StructuredObservation, 0, len(structuredCases))
	for _, probeCase := range structuredCases {
		observation, err := observeStructured(ctx, deps, budget, probeCase)
		if err != nil {
			return Report{}, err
		}
		structured = append(structured, observation)
		if err := pace(ctx, deps); err != nil {
			return Report{}, err
		}
	}

	searches := make([]SearchObservation, 0, len(searchCases))
	for _, probeCase := range searchCases {
		observation, err := observeSearch(ctx, deps, budget, probeCase)
		if err != nil {
			return Report{}, err
		}
		searches = append(searches, observation)
	}

	selections, err := documentSelections(structured, searches)
	if err != nil {
		return Report{}, err
	}
	documents := make([]DocumentObservation, 0, len(selections))
	for _, selection := range selections {
		observation, err := observeDocument(ctx, deps, budget, selection)
		if err != nil {
			return Report{}, err
		}
		documents = append(documents, observation)
		if err := pace(ctx, deps); err != nil {
			return Report{}, err
		}
	}

	observedAt := deps.now().UTC().Truncate(time.Millisecond)
	observedDate, err := checkedAtInSeoul(observedAt)
	if err != nil {
		return Report{}, newError("Probe observation date could not be determined", nil)
	}
	report := Report{
		SchemaVersion:       reportSchemaVersion,
		ObservedAt:          observedAt.Format("2006-01-02T15:04:05.000Z"),
		ObservedDate:        observedDate,
		CredentialSource:    apiKeyEnvironment,
		CredentialPersisted: false,
		RequestBudget:       RequestBudget{Maximum: budget.maximum, Used: budget.used},
		Structured:          structured, Searches: searches, Documents: documents,
	}
	contains, err := reportContainsCredential(report, deps.key)
	if err != nil {
		return Report{}, newError("Sanitized probe report could not be inspected", nil)
	}
	if contains {
		return Report{}, newError("Sanitized probe report unexpectedly contains the API key", nil)
	}
	contains, err = reportContainsForbiddenRequestMaterial(report)
	if err != nil {
		return Report{}, newError("Sanitized probe report could not be inspected", nil)
	}
	if contains {
		return Report{}, newError("Sanitized probe report unexpectedly contains request material", nil)
	}
	return report, nil
}

func newHTTPClient() *http.Client {
	return liveprobe.NewSequentialHTTPClient(requestTimeout)
}

type requestBudget struct {
	maximum int
	used    int
}

func (budget *requestBudget) consume(coordinate RequestCoordinate) error {
	if budget.used >= budget.maximum {
		return newError("Auditor evidence probe exhausted its request budget", &coordinate)
	}
	budget.used++
	return nil
}

func observeStructured(ctx context.Context, deps dependencies, budget *requestBudget, probeCase structuredCase) (StructuredObservation, error) {
	coordinate := RequestCoordinate{
		LogicalOperationID: probeCase.LogicalID, Endpoint: probeCase.Endpoint + ".json",
		CompanyName: probeCase.Company.Name, CorpCode: probeCase.Company.CorpCode,
		BusinessYear: probeCase.BusinessYear, ReportCode: probeCase.ReportCode,
	}
	query := url.Values{
		"crtfc_key": {deps.key}, "corp_code": {probeCase.Company.CorpCode},
		"bsns_year": {probeCase.BusinessYear}, "reprt_code": {probeCase.ReportCode},
	}
	body, evidence, err := execute(ctx, deps, budget, coordinate, "/"+probeCase.Endpoint+".json", query, maxJSONBody, nil)
	if err != nil {
		return StructuredObservation{}, err
	}
	if evidence.HTTPStatus != http.StatusOK {
		return StructuredObservation{}, newError("Structured auditor request returned an unexpected HTTP status", &coordinate)
	}
	envelope, err := parseJSONEnvelope(body)
	if err != nil {
		return StructuredObservation{}, newError("OpenDART returned invalid structured JSON", &coordinate)
	}
	evidence.APIStatus = optionalString(envelope.Status)
	if envelope.Status != "000" && envelope.Status != "013" {
		return StructuredObservation{}, newError("Structured auditor request returned an unexpected API status", &coordinate)
	}
	receipts := map[string]bool{}
	auditors := map[string]bool{}
	opinions := map[string]bool{}
	placeholders := 0
	for _, row := range envelope.List {
		if receipt, ok := rowString(row, "rcept_no"); ok && receipt != "" {
			if !validReceiptNumber(receipt) {
				return StructuredObservation{}, newError("Structured auditor response contained an invalid receipt number", &coordinate)
			}
			receipts[receipt] = true
		}
		if value, ok := rowString(row, "adtor"); ok {
			if isPlaceholder(value) {
				placeholders++
			} else {
				auditors[value] = true
			}
		}
		if value, ok := rowString(row, "adt_opinion"); ok {
			if isPlaceholder(value) {
				placeholders++
			} else {
				opinions[value] = true
			}
		}
	}
	return StructuredObservation{
		Request: coordinate, Response: evidence, RowCount: len(envelope.List),
		ReceiptNumbers: sortedSet(receipts), DistinctAuditors: sortedSet(auditors),
		DistinctOpinions: sortedSet(opinions), PlaceholderCount: placeholders,
	}, nil
}

func observeSearch(ctx context.Context, deps dependencies, budget *requestBudget, probeCase searchCase) (SearchObservation, error) {
	observation := SearchObservation{CompanyName: probeCase.Company.Name, CorpCode: probeCase.Company.CorpCode, DetailType: probeCase.DetailType}
	expectedTotal, expectedPages, observedRows := -1, -1, 0
	for pageNumber := 1; pageNumber <= maxSearchPages; pageNumber++ {
		coordinate := RequestCoordinate{
			LogicalOperationID: "DS001-2019001", Endpoint: "list.json",
			CompanyName: probeCase.Company.Name, CorpCode: probeCase.Company.CorpCode,
			BeginDate: searchBeginDate, EndDate: searchEndDate, LastReport: "N",
			PublicationType: "F", DetailType: probeCase.DetailType,
			PageNumber: pageNumber, PageCount: searchPageCount,
		}
		query := url.Values{
			"crtfc_key": {deps.key}, "corp_code": {probeCase.Company.CorpCode},
			"bgn_de": {searchBeginDate}, "end_de": {searchEndDate}, "last_reprt_at": {"N"},
			"pblntf_ty": {"F"}, "pblntf_detail_ty": {probeCase.DetailType},
			"page_no": {strconv.Itoa(pageNumber)}, "page_count": {strconv.Itoa(searchPageCount)},
		}
		body, evidence, err := execute(ctx, deps, budget, coordinate, "/list.json", query, maxJSONBody, nil)
		if err != nil {
			return SearchObservation{}, err
		}
		if evidence.HTTPStatus != http.StatusOK {
			return SearchObservation{}, newError("Disclosure search returned an unexpected HTTP status", &coordinate)
		}
		envelope, err := parseJSONEnvelope(body)
		if err != nil {
			return SearchObservation{}, newError("OpenDART returned invalid disclosure-search JSON", &coordinate)
		}
		evidence.APIStatus = optionalString(envelope.Status)
		page := SearchPage{Request: coordinate, Response: evidence}
		if envelope.Status == "013" {
			if len(envelope.List) != 0 || pageNumber != 1 {
				return SearchObservation{}, newError("Disclosure search returned inconsistent empty-result pagination", &coordinate)
			}
			page.PageNumber = pageNumber
			page.PageCount = searchPageCount
			page.Filings = []FilingEvidence{}
			observation.Pages = append(observation.Pages, page)
			if err := pace(ctx, deps); err != nil {
				return SearchObservation{}, err
			}
			return observation, nil
		}
		if envelope.Status != "000" {
			return SearchObservation{}, newError("Disclosure search returned an unexpected API status", &coordinate)
		}
		pageNo, pageCount, totalCount, totalPages, ok := pagination(envelope)
		if !ok || pageNo != pageNumber || pageCount != searchPageCount || totalPages < pageNo || totalCount < 0 || len(envelope.List) > pageCount {
			return SearchObservation{}, newError("Disclosure search returned inconsistent pagination metadata", &coordinate)
		}
		if expectedTotal < 0 {
			expectedTotal, expectedPages = totalCount, totalPages
		} else if totalCount != expectedTotal || totalPages != expectedPages {
			return SearchObservation{}, newError("Disclosure search pagination metadata changed between pages", &coordinate)
		}
		if totalPages > maxSearchPages {
			return SearchObservation{}, newError("Disclosure search exceeded the page ceiling", &coordinate)
		}
		page.PageNumber, page.PageCount, page.TotalCount, page.TotalPages = pageNo, pageCount, totalCount, totalPages
		page.Filings = filingEvidence(envelope.List)
		for _, filing := range page.Filings {
			if filing.CorpCode != probeCase.Company.CorpCode || !validReceiptNumber(filing.ReceiptNumber) {
				return SearchObservation{}, newError("Disclosure search returned an invalid filing identity", &coordinate)
			}
		}
		observedRows += len(page.Filings)
		observation.Pages = append(observation.Pages, page)
		if err := pace(ctx, deps); err != nil {
			return SearchObservation{}, err
		}
		if pageNumber == totalPages {
			if observedRows != expectedTotal {
				return SearchObservation{}, newError("Disclosure search pagination did not account for every reported row", &coordinate)
			}
			return observation, nil
		}
	}
	return SearchObservation{}, newError("Disclosure search pagination was incomplete", nil)
}

func observeDocument(ctx context.Context, deps dependencies, budget *requestBudget, selection DocumentSelection) (DocumentObservation, error) {
	if !validReceiptNumber(selection.ReceiptNumber) {
		return DocumentObservation{}, newError("Original-document selection contained an invalid receipt number", nil)
	}
	coordinate := RequestCoordinate{
		LogicalOperationID: "DS001-2019003", Endpoint: "document.xml",
		CompanyName: selection.CompanyName, ReceiptNumber: selection.ReceiptNumber,
	}
	query := url.Values{"crtfc_key": {deps.key}, "rcept_no": {selection.ReceiptNumber}}
	body, evidence, err := execute(ctx, deps, budget, coordinate, "/document.xml", query, maxArchiveBody, documentValidationContentType)
	if err != nil {
		return DocumentObservation{}, err
	}
	if evidence.HTTPStatus != http.StatusOK {
		return DocumentObservation{}, newError("Original-document request returned an unexpected HTTP status", &coordinate)
	}
	if !hasZIPSignature(body) {
		status := parseXMLErrorStatus(body)
		evidence.APIStatus = status
		return DocumentObservation{}, newError("Original-document request did not return a ZIP archive", &coordinate)
	}
	if !supportedDocumentMediaType(evidence.MediaType) {
		return DocumentObservation{}, newError("Original-document response used an unexpected media type", &coordinate)
	}
	archiveEvidence, err := inspectArchive(body, selection.ExpectedFirm)
	if err != nil {
		return DocumentObservation{}, newError(err.Error(), &coordinate)
	}
	if archiveEvidence.expectedMatches == 0 {
		return DocumentObservation{}, newError("Original document did not contain the expected accounting firm", &coordinate)
	}
	if archiveEvidence.sectionMarkers == 0 {
		return DocumentObservation{}, newError("Original document did not contain an independent-auditor section marker", &coordinate)
	}
	if !hasCoLocatedAuditorEvidence(archiveEvidence.entries) {
		return DocumentObservation{}, newError("Original document did not contain the expected firm and auditor marker in the same member", &coordinate)
	}
	return DocumentObservation{
		Selection: selection, Request: coordinate, Response: evidence,
		ArchiveBytes: len(body), ArchiveSHA256: sumHex(body), EntryCount: len(archiveEvidence.entries),
		ExpandedBytes: archiveEvidence.expandedBytes, Entries: archiveEvidence.entries,
		ExpectedMatches: archiveEvidence.expectedMatches, SectionMarkers: archiveEvidence.sectionMarkers,
	}, nil
}

func hasCoLocatedAuditorEvidence(entries []DocumentEntry) bool {
	for _, entry := range entries {
		if entry.ExpectedFirmMatches > 0 && entry.AuditorSectionMarkerMatches > 0 {
			return true
		}
	}
	return false
}

type responseValidation func(string, []byte) (string, bool)

func execute(ctx context.Context, deps dependencies, budget *requestBudget, coordinate RequestCoordinate, path string, query url.Values, limit int64, validation responseValidation) ([]byte, ResponseEvidence, error) {
	if err := budget.consume(coordinate); err != nil {
		return nil, ResponseEvidence{}, err
	}
	requestContext, cancel := context.WithTimeout(ctx, requestTimeout)
	defer cancel()
	request, err := http.NewRequestWithContext(requestContext, http.MethodGet, strings.TrimRight(deps.origin, "/")+path+"?"+query.Encode(), nil)
	if err != nil {
		return nil, ResponseEvidence{}, newError("OpenDART request could not be constructed", &coordinate)
	}
	request.Header.Set("Accept", "application/zip, application/json, application/xml")
	request.Header.Set("User-Agent", probeUserAgent)
	response, err := deps.client.Do(request)
	if err != nil {
		return nil, ResponseEvidence{}, newError("OpenDART request failed before a response was received", &coordinate)
	}
	body, err := readBoundedBody(response.Body, limit)
	if err != nil {
		return nil, ResponseEvidence{}, newError(err.Error(), &coordinate)
	}
	contentType := response.Header.Get("Content-Type")
	validatedContentType, validate := contentType, true
	if validation != nil {
		validatedContentType, validate = validation(contentType, body)
	}
	if validate {
		if err := deps.validate.ValidateResponse(http.MethodGet, path, validatedContentType, response.StatusCode, body); err != nil {
			return nil, ResponseEvidence{}, newError("OpenDART response did not satisfy the committed OpenAPI representation", &coordinate)
		}
	}
	return body, ResponseEvidence{
		HTTPStatus: response.StatusCode, MediaType: mediaType(contentType),
		BodyBytes: len(body), BodySHA256: sumHex(body),
	}, nil
}

func documentValidationContentType(observed string, body []byte) (string, bool) {
	if hasZIPSignature(body) {
		// The source uses a non-contract media type and absolute-looking member
		// names. The caller validates ZIP structure and content under stricter
		// probe-specific bounds without extracting those paths.
		return "", false
	}
	return observed, true
}

func hasZIPSignature(body []byte) bool {
	return len(body) >= 4 && body[0] == 'P' && body[1] == 'K' &&
		(body[2] == 0x03 && body[3] == 0x04 || body[2] == 0x05 && body[3] == 0x06)
}

type jsonEnvelope struct {
	Status     string
	List       []map[string]any
	PageNumber any
	PageCount  any
	TotalCount any
	TotalPages any
}

func parseJSONEnvelope(body []byte) (jsonEnvelope, error) {
	decoder := json.NewDecoder(bytes.NewReader(body))
	decoder.UseNumber()
	var raw struct {
		Status     string           `json:"status"`
		List       []map[string]any `json:"list"`
		PageNumber any              `json:"page_no"`
		PageCount  any              `json:"page_count"`
		TotalCount any              `json:"total_count"`
		TotalPages any              `json:"total_page"`
	}
	if err := decoder.Decode(&raw); err != nil {
		return jsonEnvelope{}, err
	}
	var trailing any
	if err := decoder.Decode(&trailing); !errors.Is(err, io.EOF) {
		return jsonEnvelope{}, errors.New("trailing JSON value")
	}
	return jsonEnvelope{raw.Status, raw.List, raw.PageNumber, raw.PageCount, raw.TotalCount, raw.TotalPages}, nil
}

func pagination(envelope jsonEnvelope) (int, int, int, int, bool) {
	page, ok1 := integer(envelope.PageNumber)
	count, ok2 := integer(envelope.PageCount)
	total, ok3 := integer(envelope.TotalCount)
	pages, ok4 := integer(envelope.TotalPages)
	return page, count, total, pages, ok1 && ok2 && ok3 && ok4
}

func integer(value any) (int, bool) {
	switch typed := value.(type) {
	case json.Number:
		parsed, err := strconv.Atoi(string(typed))
		return parsed, err == nil
	case string:
		parsed, err := strconv.Atoi(typed)
		return parsed, err == nil
	default:
		return 0, false
	}
}

func filingEvidence(rows []map[string]any) []FilingEvidence {
	filings := make([]FilingEvidence, 0, len(rows))
	for _, row := range rows {
		filings = append(filings, FilingEvidence{
			CorpCode: valueString(row, "corp_code"), CorpName: valueString(row, "corp_name"),
			ReportName: valueString(row, "report_nm"), ReceiptNumber: valueString(row, "rcept_no"),
			FilerName: valueString(row, "flr_nm"), ReceiptDate: valueString(row, "rcept_dt"),
			Remarks: valueString(row, "rm"),
		})
	}
	return filings
}

func documentSelections(structured []StructuredObservation, searches []SearchObservation) ([]DocumentSelection, error) {
	allNuga := make([]FilingEvidence, 0)
	for _, search := range searches {
		if search.CorpCode != nuga.CorpCode {
			continue
		}
		for _, page := range search.Pages {
			allNuga = append(allNuga, page.Filings...)
		}
	}
	selections := make([]DocumentSelection, 0, 4)
	for _, year := range []string{"2015", "2020", "2025"} {
		filing, ok := selectPeriodFiling(allNuga, year)
		if !ok || !validReceiptNumber(filing.ReceiptNumber) || filing.FilerName == "" {
			return nil, newError("Required Nuga document sample could not be selected", nil)
		}
		selections = append(selections, DocumentSelection{
			CompanyName: nuga.Name, ReportPeriod: year, ReceiptNumber: filing.ReceiptNumber,
			ExpectedFirm: filing.FilerName, SelectionBasis: "latest-receipt-matching-report-period-from-F001-or-F002",
		})
	}
	lotteReceipts := map[string]bool{}
	for _, observation := range structured {
		if observation.Request.CorpCode == lotte.CorpCode && observation.Request.Endpoint == auditorOpinionEndpoint+".json" {
			for _, receipt := range observation.ReceiptNumbers {
				if !validReceiptNumber(receipt) {
					return nil, newError("Required Lotte result contained an invalid receipt number", nil)
				}
				lotteReceipts[receipt] = true
			}
		}
	}
	receipts := sortedSet(lotteReceipts)
	if len(receipts) == 0 {
		return nil, newError("Required Lotte periodic-report document could not be selected", nil)
	}
	selections = append(selections, DocumentSelection{
		CompanyName: lotte.Name, ReportPeriod: "2024", ReceiptNumber: receipts[len(receipts)-1],
		ExpectedFirm: "안진회계법인", SelectionBasis: "greatest-receipt-from-2024-annual-structured-result",
	})
	return selections, nil
}

func selectPeriodFiling(filings []FilingEvidence, year string) (FilingEvidence, bool) {
	candidates := make([]FilingEvidence, 0)
	for _, filing := range filings {
		if strings.Contains(filing.ReportName, year) {
			candidates = append(candidates, filing)
		}
	}
	sort.Slice(candidates, func(left, right int) bool {
		if candidates[left].ReceiptDate != candidates[right].ReceiptDate {
			return candidates[left].ReceiptDate > candidates[right].ReceiptDate
		}
		if candidates[left].ReceiptNumber != candidates[right].ReceiptNumber {
			return candidates[left].ReceiptNumber > candidates[right].ReceiptNumber
		}
		return candidates[left].ReportName < candidates[right].ReportName
	})
	if len(candidates) == 0 {
		return FilingEvidence{}, false
	}
	return candidates[0], true
}

type archiveEvidence struct {
	entries         []DocumentEntry
	expandedBytes   int64
	expectedMatches int
	sectionMarkers  int
}

func inspectArchive(body []byte, expectedFirm string) (archiveEvidence, error) {
	if compactDocumentText(expectedFirm) == "" {
		return archiveEvidence{}, errors.New("Original-document selection did not identify an expected accounting firm")
	}
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return archiveEvidence{}, errors.New("Original-document response was not a valid ZIP archive")
	}
	if len(reader.File) == 0 || len(reader.File) > maxArchiveEntries {
		return archiveEvidence{}, errors.New("Original-document ZIP exceeded the entry-count limit")
	}
	result := archiveEvidence{entries: make([]DocumentEntry, 0, len(reader.File))}
	for _, member := range reader.File {
		if member.FileInfo().IsDir() {
			continue
		}
		if member.UncompressedSize64 > maxArchiveMember || result.expandedBytes+int64(member.UncompressedSize64) > maxArchiveExpanded {
			return archiveEvidence{}, errors.New("Original-document ZIP exceeded an expanded-size limit")
		}
		opened, err := member.Open()
		if err != nil {
			return archiveEvidence{}, errors.New("Original-document ZIP member could not be opened")
		}
		content, err := readBoundedBody(opened, maxArchiveMember)
		if err != nil {
			return archiveEvidence{}, errors.New("Original-document ZIP member exceeded a safety limit")
		}
		if result.expandedBytes+int64(len(content)) > maxArchiveExpanded {
			return archiveEvidence{}, errors.New("Original-document ZIP exceeded an expanded-size limit")
		}
		result.expandedBytes += int64(len(content))
		text, decoding := decodeDocumentText(content)
		searchable := compactDocumentText(text)
		expectedMatches := strings.Count(searchable, compactDocumentText(expectedFirm))
		markerMatches := auditorMarkerMatches(searchable)
		name := decodedMemberName(member.Name)
		result.entries = append(result.entries, DocumentEntry{
			Name: name, NameSHA256: sumHex([]byte(member.Name)), CompressedBytes: member.CompressedSize64,
			ExpandedBytes: uint64(len(content)), ContentSHA256: sumHex(content), Decoding: decoding,
			ExpectedFirmMatches: expectedMatches, AuditorSectionMarkerMatches: markerMatches,
		})
		result.expectedMatches += expectedMatches
		result.sectionMarkers += markerMatches
	}
	if len(result.entries) == 0 {
		return archiveEvidence{}, errors.New("Original-document ZIP contained no file entries")
	}
	return result, nil
}

func decodeDocumentText(content []byte) (string, string) {
	if utf8.Valid(content) {
		return string(content), "utf-8"
	}
	decoded, _, err := transform.Bytes(korean.EUCKR.NewDecoder(), content)
	if err == nil && utf8.Valid(decoded) && !bytes.Contains(decoded, []byte("\uFFFD")) {
		return string(decoded), "cp949-compatible"
	}
	return "", "binary-or-unsupported"
}

func decodedMemberName(name string) string {
	decoded, _ := decodeDocumentText([]byte(name))
	if decoded == "" {
		decoded = "undecodable-member-name"
	}
	decoded = strings.Map(func(r rune) rune {
		if r < 0x20 || r == 0x7f {
			return -1
		}
		return r
	}, decoded)
	const maximumNameRunes = 256
	runes := []rune(decoded)
	if len(runes) > maximumNameRunes {
		decoded = string(runes[:maximumNameRunes])
	}
	return decoded
}

func auditorMarkerMatches(text string) int {
	markers := []string{"독립된외부감사인의감사보고서", "독립된감사인의감사보고서", "외부감사인의감사보고서"}
	total := 0
	for _, marker := range markers {
		matches := strings.Count(text, marker)
		total += matches
		if matches > 0 {
			text = strings.ReplaceAll(text, marker, "")
		}
	}
	return total
}

func compactDocumentText(text string) string {
	text = html.UnescapeString(text)
	var result strings.Builder
	result.Grow(len(text))
	inTag := false
	for _, current := range text {
		switch current {
		case '<':
			inTag = true
		case '>':
			inTag = false
		default:
			if !inTag && current != '\u00a0' && current != '\u200b' && current != '\ufeff' && !strings.ContainsRune(" \t\r\n", current) {
				result.WriteRune(current)
			}
		}
	}
	return result.String()
}

func parseXMLErrorStatus(body []byte) *string {
	var envelope struct {
		Status string `xml:"status"`
	}
	if xml.Unmarshal(body, &envelope) != nil {
		return nil
	}
	return optionalString(envelope.Status)
}

func apiKey() (string, error) {
	key := os.Getenv(apiKeyEnvironment)
	if len(key) != 40 {
		return "", newError(apiKeyEnvironment+" must contain the 40-character OpenDART API key", nil)
	}
	return key, nil
}

func readBoundedBody(body io.ReadCloser, maximum int64) ([]byte, error) {
	limited := io.LimitReader(body, maximum+1)
	value, readErr := io.ReadAll(limited)
	closeErr := body.Close()
	if readErr != nil {
		return nil, errors.New("OpenDART response body could not be read")
	}
	if closeErr != nil {
		return nil, errors.New("OpenDART response body could not be closed")
	}
	if int64(len(value)) > maximum {
		return nil, errors.New("OpenDART response body exceeded the size limit")
	}
	return value, nil
}

func mediaType(header string) *string {
	if header == "" {
		return nil
	}
	value, _, err := mime.ParseMediaType(header)
	if err != nil || strings.TrimSpace(value) == "" {
		return nil
	}
	value = strings.ToLower(strings.TrimSpace(value))
	return &value
}

func supportedDocumentMediaType(value *string) bool {
	if value == nil {
		return false
	}
	return *value == "application/zip" || *value == "application/x-msdownload"
}

func rowString(row map[string]any, name string) (string, bool) {
	value, exists := row[name]
	text, ok := value.(string)
	return strings.TrimSpace(text), exists && ok
}

func valueString(row map[string]any, name string) string {
	value, _ := rowString(row, name)
	return value
}

func validReceiptNumber(value string) bool {
	if len(value) != 14 {
		return false
	}
	for _, character := range value {
		if character < '0' || character > '9' {
			return false
		}
	}
	return true
}

func isPlaceholder(value string) bool {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "", "-", "--", "n/a", "na", "해당사항없음", "해당사항 없음", "없음":
		return true
	default:
		return false
	}
}

func sumHex(value []byte) string {
	sum := sha256.Sum256(value)
	return hex.EncodeToString(sum[:])
}

func sortedSet(values map[string]bool) []string {
	result := make([]string, 0, len(values))
	for value := range values {
		result = append(result, value)
	}
	sort.Strings(result)
	return result
}

func optionalString(value string) *string {
	if value == "" {
		return nil
	}
	copy := value
	return &copy
}

func stringValue(value *string) string {
	if value == nil {
		return ""
	}
	return *value
}

func newError(message string, coordinate *RequestCoordinate) *Error {
	return &Error{Message: message, Request: coordinate}
}

func checkedAtInSeoul(now time.Time) (string, error) {
	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return "", fmt.Errorf("load timezone: %w", err)
	}
	return now.In(location).Format(time.DateOnly), nil
}

func pace(ctx context.Context, deps dependencies) error {
	if err := deps.wait(ctx, requestPacing); err != nil {
		return newError("Probe pacing was interrupted", nil)
	}
	return nil
}

func waitFor(ctx context.Context, duration time.Duration) error {
	timer := time.NewTimer(duration)
	defer timer.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return nil
	}
}

func reportContainsCredential(report Report, key string) (bool, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return false, err
	}
	encodedKey, err := json.Marshal(key)
	if err != nil || len(encodedKey) < 2 {
		return false, err
	}
	return bytes.Contains(encoded, encodedKey[1:len(encodedKey)-1]), nil
}

func reportContainsForbiddenRequestMaterial(report Report) (bool, error) {
	encoded, err := json.Marshal(report)
	if err != nil {
		return false, err
	}
	return bytes.Contains(encoded, []byte("://")) ||
		bytes.Contains(encoded, []byte("crtfc_key")) ||
		bytes.Contains(encoded, []byte("OPENDART_API_KEY=")), nil
}
