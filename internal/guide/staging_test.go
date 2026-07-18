package guide

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateStagingAcceptsCompleteAndStructuralCatalogs(t *testing.T) {
	completeRoot := copyCatalogTree(t)
	if err := validateStaging(filepath.Dir(completeRoot), true); err != nil {
		t.Fatal(err)
	}

	partialRoot := copyCatalogTree(t)
	removeCatalogEndpoint(t, partialRoot)
	if err := validateStaging(filepath.Dir(partialRoot), false); err != nil {
		t.Fatal(err)
	}
	if err := validateStaging(filepath.Dir(partialRoot), true); err == nil || !strings.Contains(err.Error(), "physical-path-completeness") {
		t.Fatalf("complete validation error = %v", err)
	}
}

func TestValidateStagingStopsAtCatalogFailure(t *testing.T) {
	root := copyCatalogTree(t)
	writeCatalogTestFile(t, filepath.Join(filepath.Dir(root), OutputMarker), "changed\n")
	replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml"), "  summary: 기업개황 (JSON)\n", "")

	err := validateStaging(filepath.Dir(root), true)
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Diagnostic.Rule != "output-marker" {
		t.Fatalf("validation error = %v", err)
	}
	var stagingErr *stagingValidationError
	if !errors.As(err, &stagingErr) || stagingErr.phase != "catalog/ownership" ||
		stagingErr.artifact != filepath.Join(filepath.Dir(root), OutputMarker) {
		t.Fatalf("staging diagnostic = %#v", stagingErr)
	}
	if !strings.Contains(err.Error(), "staged catalog") || strings.Contains(err.Error(), "operation-summary") {
		t.Fatalf("catalog failure is not bounded or ordered: %v", err)
	}
}

func TestValidateStagingReportsBoundedOpenAPILintContext(t *testing.T) {
	root := copyCatalogTree(t)
	path := filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml")
	replaceCatalogTestFile(t, path, "  summary: 기업개황 (JSON)\n", "")

	err := validateStaging(filepath.Dir(root), true)
	if err == nil || !strings.Contains(err.Error(), "staged OpenAPI") || !strings.Contains(err.Error(), "operation-summary") || !strings.Contains(err.Error(), "GET /company.json") {
		t.Fatalf("lint validation error = %v", err)
	}
	if len([]rune(err.Error())) > 4*maxStagingDiagnosticRunes {
		t.Fatalf("lint validation error is unbounded: %d runes", len([]rune(err.Error())))
	}
}

func TestBoundedStagingErrorPreservesCause(t *testing.T) {
	cause := errors.New(strings.Repeat("sentinel", maxStagingDiagnosticRunes))
	err := boundedStagingError("OpenAPI", strings.Repeat("artifact", maxStagingDiagnosticRunes), cause)
	if !errors.Is(err, cause) {
		t.Fatalf("error does not preserve cause: %v", err)
	}
	if len([]rune(err.Error())) > 3*maxStagingDiagnosticRunes {
		t.Fatalf("error = %d runes, want bounded output", len([]rune(err.Error())))
	}
}

func TestCompareStagedRejectsMarkerSymlinkBeforeReadingTarget(t *testing.T) {
	stagedRoot := copyCatalogTree(t)
	marker := filepath.Join(filepath.Dir(stagedRoot), OutputMarker)
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Join(filepath.Dir(stagedRoot), "missing-target"), marker); err != nil {
		t.Fatal(err)
	}

	// A missing baseline proves marker validation happens before semantic comparison.
	err := compareStaged(filepath.Dir(stagedRoot), filepath.Join(t.TempDir(), "missing-baseline"))
	if err == nil || !strings.Contains(err.Error(), "staged ownership marker is invalid") || strings.Contains(err.Error(), "no such file") {
		t.Fatalf("comparison error = %v", err)
	}
}
