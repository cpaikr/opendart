package openapi

import (
	"net/http"
	"net/url"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestCanonicalOperationsAreDeterministicPhysicalProjection(t *testing.T) {
	document, err := Load(filepath.Join("..", "..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	catalog, err := document.Operations()
	if err != nil {
		t.Fatal(err)
	}
	if len(catalog.Operations) != 167 {
		t.Fatalf("operation count = %d", len(catalog.Operations))
	}
	if !slices.Equal(catalog.Servers, []string{"https://opendart.fss.or.kr/api"}) {
		t.Fatalf("servers = %#v", catalog.Servers)
	}
	if !slices.Equal(catalog.SecuritySchemes, []SecurityScheme{{
		Name: "crtfcKey", Type: "apiKey", Location: "query", ParameterName: "crtfc_key",
	}}) {
		t.Fatalf("security schemes = %#v", catalog.SecuritySchemes)
	}

	representations := make(map[string]int)
	identities := make(map[string]bool)
	for index, operation := range catalog.Operations {
		if index > 0 && catalog.Operations[index-1].Identity() >= operation.Identity() {
			t.Fatalf("operations are not strictly sorted at %q", operation.Identity())
		}
		if identities[operation.Identity()] {
			t.Fatalf("duplicate identity %q", operation.Identity())
		}
		identities[operation.Identity()] = true
		representations[operation.PrimaryRepresentation]++
		if operation.Method != http.MethodGet || operation.OperationID == "" || operation.LogicalOperationID == "" {
			t.Fatalf("incomplete operation = %#v", operation)
		}
		if !slices.Equal(operation.Servers, catalog.Servers) || !slices.Equal(operation.SecurityRequirements, []string{"crtfcKey"}) {
			t.Fatalf("operation trust metadata = %#v", operation)
		}
	}
	if representations["application/json"] != 82 || representations["application/xml"] != 82 || representations["application/zip"] != 3 {
		t.Fatalf("representations = %#v", representations)
	}

	zipOperation := operationByPath(t, catalog.Operations, "/document.xml")
	if zipOperation.PrimaryRepresentation != "application/zip" || !slices.Equal(zipOperation.AlternateResponseMedia, []string{"application/xml"}) {
		t.Fatalf("ZIP representation routing = %#v", zipOperation)
	}
}

func TestValidateRequestChecksCanonicalQueryParametersOffline(t *testing.T) {
	document, err := Load(filepath.Join("..", "..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	query := url.Values{
		"corp_code":  {"00334624,00126380"},
		"bsns_year":  {"2018"},
		"reprt_code": {"11011"},
	}
	if err := document.ValidateRequest(http.MethodGet, "/fnlttMultiAcnt.json", query); err != nil {
		t.Fatal(err)
	}
	query.Del("bsns_year")
	if err := document.ValidateRequest(http.MethodGet, "/fnlttMultiAcnt.json", query); err == nil || !strings.Contains(err.Error(), "required") {
		t.Fatalf("missing required parameter error = %v", err)
	}
}

func operationByPath(t *testing.T, operations []Operation, path string) Operation {
	t.Helper()
	for _, operation := range operations {
		if operation.Path == path {
			return operation
		}
	}
	t.Fatalf("operation %q is missing", path)
	return Operation{}
}
