package openapi_test

import (
	"path/filepath"
	"runtime"
	"slices"
	"sort"
	"testing"

	"github.com/cpaikr/opendart/internal/guide"
	"github.com/cpaikr/opendart/internal/openapi"
)

func TestSDKSurfaceCoversCanonicalPhysicalAndLogicalOperations(t *testing.T) {
	root := canonicalRoot(t)
	document, err := openapi.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	surface, err := document.InspectSDKSurface()
	if err != nil {
		t.Fatal(err)
	}
	catalog, err := guide.ValidateCatalog(guide.CatalogOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	if len(surface.Operations) != catalog.PhysicalPaths {
		t.Fatalf("SDK surface operations = %d, canonical physical paths = %d", len(surface.Operations), catalog.PhysicalPaths)
	}

	physical := make(map[string]bool, len(surface.Operations))
	logical := make(map[string]bool)
	for _, operation := range surface.Operations {
		if operation.OperationID == "" || operation.LogicalOperationID == "" || operation.Path == "" || operation.Method == "" || operation.Description == "" {
			t.Fatalf("incomplete operation identity: %#v", operation)
		}
		if operation.GuideURL == "" || operation.SourceCheckedAt == "" {
			t.Fatalf("missing source provenance for %s", operation.OperationID)
		}
		if physical[operation.OperationID] {
			t.Fatalf("duplicate operationId %q", operation.OperationID)
		}
		physical[operation.OperationID] = true
		logical[operation.LogicalOperationID] = true
		if len(operation.Security) == 0 || len(operation.Responses) == 0 {
			t.Fatalf("missing security or response routing for %s", operation.OperationID)
		}
		for _, parameter := range operation.Parameters {
			if parameter.Name == "" || parameter.Location == "" || parameter.Description == "" || !parameter.HasSchema {
				t.Fatalf("incomplete parameter evidence for %s: %#v", operation.OperationID, parameter)
			}
		}
	}
	if len(logical) != catalog.LogicalEndpoints {
		t.Fatalf("SDK surface logical operations = %d, canonical logical endpoints = %d", len(logical), catalog.LogicalEndpoints)
	}
	physicalIDs := boolSetKeys(physical)
	logicalIDs := boolSetKeys(logical)
	if !slices.Equal(physicalIDs, catalog.PhysicalOperationIDs()) {
		t.Fatalf("SDK surface physical identities differ from the canonical catalog")
	}
	if !slices.Equal(logicalIDs, catalog.LogicalOperationIDs()) {
		t.Fatalf("SDK surface logical identities differ from the canonical catalog")
	}

	multiCompany := findSDKOperation(t, surface, "get_fnlttMultiAcnt_json")
	corpCode := findSDKParameter(t, multiCompany, "corp_code")
	if !corpCode.Required || corpCode.Style != "form" || corpCode.Explode ||
		!slices.Equal(corpCode.Types, []string{"array"}) || !slices.Equal(corpCode.ItemTypes, []string{"string"}) {
		t.Fatalf("multi-company serialization evidence = %#v", corpCode)
	}
	if corpCode.MinItems == nil || *corpCode.MinItems != 1 || corpCode.MaxItems == nil || *corpCode.MaxItems != 100 {
		t.Fatalf("multi-company cardinality evidence = %#v", corpCode)
	}
	*corpCode.MinItems = 99
	fresh, err := document.InspectSDKSurface()
	if err != nil {
		t.Fatal(err)
	}
	freshCorpCode := findSDKParameter(t, findSDKOperation(t, fresh, "get_fnlttMultiAcnt_json"), "corp_code")
	if freshCorpCode.MinItems == nil || *freshCorpCode.MinItems != 1 {
		t.Fatal("mutating repository-owned SDK evidence changed the hidden OpenAPI model")
	}
	if len(multiCompany.Security) != 1 || len(multiCompany.Security[0].Schemes) != 1 {
		t.Fatalf("authentication requirements = %#v", multiCompany.Security)
	}
	scheme := multiCompany.Security[0].Schemes[0]
	if scheme.Identifier != "crtfcKey" || scheme.Type != "apiKey" || scheme.Location != "query" || scheme.Name != "crtfc_key" {
		t.Fatalf("authentication scheme evidence = %#v", scheme)
	}

	zipOperation := findSDKOperation(t, surface, "get_corpCode_xml")
	if len(zipOperation.Responses) != 1 || zipOperation.Responses[0].Selector != "default" || zipOperation.Responses[0].HTTPStatusEvidence != "not-documented" {
		t.Fatalf("ZIP response selector evidence = %#v", zipOperation.Responses)
	}
	media := zipOperation.Responses[0].MediaTypes
	if len(media) != 2 ||
		media[0].Name != "application/xml" || media[0].ContentTypeStatus != "empirically-observed-error-response" ||
		media[1].Name != "application/zip" || media[1].ContentTypeStatus != "inferred-from-documented-output-format" {
		t.Fatalf("ZIP/XML routing evidence = %#v", media)
	}
}

func boolSetKeys(values map[string]bool) []string {
	keys := make([]string, 0, len(values))
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func findSDKOperation(t *testing.T, surface openapi.SDKSurface, operationID string) openapi.SDKSurfaceOperation {
	t.Helper()
	for _, operation := range surface.Operations {
		if operation.OperationID == operationID {
			return operation
		}
	}
	t.Fatalf("SDK surface does not contain operation %q", operationID)
	return openapi.SDKSurfaceOperation{}
}

func findSDKParameter(t *testing.T, operation openapi.SDKSurfaceOperation, name string) openapi.SDKSurfaceParameter {
	t.Helper()
	for _, parameter := range operation.Parameters {
		if parameter.Name == name {
			return parameter
		}
	}
	t.Fatalf("SDK operation %q does not contain parameter %q", operation.OperationID, name)
	return openapi.SDKSurfaceParameter{}
}

func canonicalRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	return filepath.Join(filepath.Dir(current), "..", "..", "openapi", "openapi.yaml")
}
