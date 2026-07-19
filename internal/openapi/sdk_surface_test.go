package openapi_test

import (
	"path/filepath"
	"runtime"
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
		if operation.OperationID == "" || operation.LogicalOperationID == "" || operation.Path == "" || operation.Method == "" {
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
		if len(operation.SecuritySchemes) == 0 || len(operation.Responses) == 0 {
			t.Fatalf("missing security or response routing for %s", operation.OperationID)
		}
		for _, parameter := range operation.Parameters {
			if parameter.Name == "" || parameter.Location == "" || !parameter.HasSchema {
				t.Fatalf("incomplete parameter evidence for %s: %#v", operation.OperationID, parameter)
			}
		}
	}
	if len(logical) != catalog.LogicalEndpoints {
		t.Fatalf("SDK surface logical operations = %d, canonical logical endpoints = %d", len(logical), catalog.LogicalEndpoints)
	}

	multiCompany := findSDKOperation(t, surface, "get_fnlttMultiAcnt_json")
	corpCode := findSDKParameter(t, multiCompany, "corp_code")
	if !corpCode.Required || corpCode.Style != "form" || corpCode.Explode || len(corpCode.Types) != 1 || corpCode.Types[0] != "array" {
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
