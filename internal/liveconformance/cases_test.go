package liveconformance

import (
	"archive/zip"
	"bytes"
	"errors"
	"net/url"
	"path/filepath"
	"reflect"
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"golang.org/x/text/encoding/korean"
	"golang.org/x/text/transform"
)

func TestCanonicalRepositoryPreflight(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	report, err := PreflightRepository(repositoryRoot)
	if err != nil {
		if typed, ok := err.(*Error); ok {
			t.Fatalf("PreflightRepository() failure = %+v", typed.Failure)
		}
		t.Fatalf("PreflightRepository() error = %v", err)
	}
	if !report.Valid || report.PrimaryCases != 167 || report.RequestCeiling != 199 {
		t.Fatalf("PreflightRepository() = %+v", report)
	}
	seen := make(map[string]bool)
	cases := PrimaryCases()
	if !reflect.DeepEqual(cases, PrimaryCases()) || len(PrimaryAssertions()) != 85 {
		t.Fatal("primary registry is not deterministic or logically complete")
	}
	discoveries := PrimaryDiscoveries()
	if len(discoveries) != 1 || len(discoveries[0].Requests) != 32 || len(discoveries[0].Targets) != 84 {
		t.Fatalf("discoveries = %#v", discoveries)
	}
	for _, testCase := range cases {
		identity := testCase.operationIdentity()
		if seen[identity] {
			t.Fatalf("duplicate primary operation %q", identity)
		}
		seen[identity] = true
		if _, exists := testCase.Parameters["crtfc_key"]; exists {
			t.Fatalf("case %q contains credential material", testCase.ID)
		}
	}
	document, err := openapispec.Load(filepath.Join(repositoryRoot, "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	discoveries[0].MaxRequests += 2
	_, err = Preflight(document, cases, PrimaryAssertions(), discoveries...)
	var conformanceError *Error
	if !errors.As(err, &conformanceError) || conformanceError.Failure.Code != "request-budget-exceeded" {
		t.Fatalf("over-budget preflight error = %v", err)
	}
}

func TestCanonicalPreflightRejectsDiscoveryPageGapsAndDuplicates(t *testing.T) {
	repositoryRoot := filepath.Clean(filepath.Join("..", ".."))
	document, err := openapispec.Load(filepath.Join(repositoryRoot, "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	for _, test := range []struct {
		name string
		page string
	}{
		{name: "duplicate", page: "1"},
		{name: "gap", page: "3"},
	} {
		t.Run(test.name, func(t *testing.T) {
			discoveries := PrimaryDiscoveries()
			discoveries[0].Requests[1].Parameters["page_no"] = []string{test.page}
			_, err := Preflight(document, PrimaryCases(), PrimaryAssertions(), discoveries...)
			var conformanceError *Error
			if !errors.As(err, &conformanceError) || conformanceError.Failure.Code != "invalid-discovery-request" {
				t.Fatalf("Preflight() error = %v", err)
			}
		})
	}
}

func TestDiscoveredEventAssertionsBindIdentityAndDS006Nesting(t *testing.T) {
	jsonResponse := Response{Representation: "application/json", JSON: map[string]any{"group": []any{map[string]any{"list": []any{map[string]any{"corp_code": "00999999", "rcept_no": "20250101000001"}}}}}}
	xmlResponse := Response{Representation: "application/xml", XMLValues: map[string][]string{
		"result/group/list/corp_code": {"00999999"},
		"result/group/list/rcept_no":  {"20250101000001"},
	}}
	assertion := eventIdentityAssertion("00999999", "group/list")
	for _, response := range []Response{jsonResponse, xmlResponse} {
		if evidence, ok := assertion(response); !ok || evidence.Count != 1 {
			t.Fatalf("DS006 identity assertion = %+v, %v", evidence, ok)
		}
		if _, ok := eventIdentityAssertion("00888888", "group/list")(response); ok {
			t.Fatal("DS006 identity assertion accepted the wrong corporation")
		}
	}
}

func TestDiscoveryPaginationRequiresClosedConsistentSequence(t *testing.T) {
	query := url.Values{"pblntf_detail_ty": {"B001"}, "bgn_de": {"20250101"}, "end_de": {"20250331"}, "page_no": {"2"}}
	partition := discoveryPartitionKey(query)
	response := Response{Representation: "application/json", JSON: map[string]any{"page_no": "2", "total_page": "1"}}
	if discoveryPagesClosed(response, query, map[string]int{partition: 2}, map[string]int{partition: 2}) {
		t.Fatal("pagination accepted a page beyond the response total")
	}
	response.JSON["total_page"] = "2"
	if !discoveryPagesClosed(response, query, map[string]int{partition: 2}, map[string]int{partition: 2}) {
		t.Fatal("pagination rejected a consistent declared page")
	}
	response.APIStatus = "013"
	if discoveryPagesClosed(response, query, map[string]int{partition: 2}, map[string]int{partition: 2}) {
		t.Fatal("pagination accepted an empty envelope after a nonempty first page")
	}
}

func TestDocumentArchiveAcceptsKoreanXMLAndCanonicalDownloadMedia(t *testing.T) {
	encoded, _, err := transform.Bytes(korean.EUCKR.NewEncoder(), []byte(`<DOCUMENT><AUDITOR>삼정회계법인</AUDITOR></DOCUMENT>`))
	if err != nil {
		t.Fatal(err)
	}
	var archive bytes.Buffer
	writer := zip.NewWriter(&archive)
	entry, err := writer.Create("document.xml")
	if err != nil {
		t.Fatal(err)
	}
	if _, err := entry.Write(encoded); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if got := canonicalResponseMediaType("application/zip", "application/x-msdownload", archive.Bytes()); got != "application/zip" {
		t.Fatalf("canonicalResponseMediaType() = %q", got)
	}
	response, err := parseZIP(archive.Bytes())
	if err != nil {
		t.Fatalf("parseZIP() error = %v", err)
	}
	evidence, ok := documentArchiveAssertion(response)
	if !ok || evidence.Count != 1 {
		t.Fatalf("documentArchiveAssertion() = %+v, %v", evidence, ok)
	}
	document, err := openapispec.Load(filepath.Join("..", "..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, archive.Bytes()); err != nil {
		t.Fatalf("ValidateResponse() error = %v", err)
	}
}
