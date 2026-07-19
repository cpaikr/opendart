package verification

import (
	"encoding/json"
	"errors"
	"fmt"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/cpaikr/opendart/internal/guide"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/releaseguard"
)

func TestVerifyRunsPhasesInOrderAndReturnsBoundedReport(t *testing.T) {
	var calls []string
	deps := dependencies{
		validateCatalog: func(options guide.CatalogOptions) (guide.CatalogReport, error) {
			calls = append(calls, phaseCatalog+":"+filepath.Base(options.Root))
			if options.StructuralOnly {
				t.Fatal("Verify requested structural-only catalog validation")
			}
			return guide.CatalogReport{
				Root:             options.Root,
				OpenAPI:          "3.2.0",
				LogicalEndpoints: 85,
				PhysicalPaths:    167,
				RequestArguments: 337,
				ResponseFields:   2383,
				MessageCodes:     13,
				GroupCounts:      map[string]int{"DS001": 4},
			}, nil
		},
		lint: func(artifact string) ([]openapispec.LintDiagnostic, error) {
			calls = append(calls, "lint:"+filepath.Base(artifact))
			return nil, nil
		},
		checkFresh: func(source, bundle string) error {
			calls = append(calls, phaseBundleFreshness+":"+filepath.Base(source)+":"+filepath.Base(bundle))
			return nil
		},
		checkEvidence: func(path string) error {
			calls = append(calls, phaseAuditorEvidence+":"+filepath.Base(path))
			return nil
		},
		checkRelease: func(root string) error {
			calls = append(calls, phaseReleaseGuard+":"+filepath.Base(root))
			return nil
		},
	}

	report, err := verifyWith(filepath.Join("testdata", "repository"), deps)
	if err != nil {
		t.Fatal(err)
	}
	wantCalls := []string{
		"catalog:openapi.yaml",
		"lint:openapi.yaml",
		"bundle-freshness:openapi.yaml:openapi.bundle.yaml",
		"lint:openapi.bundle.yaml",
		"auditor-evidence:auditor-2026-07-18.json",
		"release-guard:repository",
	}
	if !reflect.DeepEqual(calls, wantCalls) {
		t.Fatalf("calls = %#v, want %#v", calls, wantCalls)
	}
	if !reflect.DeepEqual(report.PassedPhases, passedPhases) {
		t.Fatalf("passed phases = %#v", report.PassedPhases)
	}
	if report.Catalog != (CatalogSummary{
		OpenAPI: "3.2.0", LogicalEndpoints: 85, PhysicalPaths: 167,
		RequestArguments: 337, ResponseFields: 2383, MessageCodes: 13,
	}) {
		t.Fatalf("catalog summary = %#v", report.Catalog)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	for _, forbidden := range []string{"testdata", "GroupCounts", "groupCounts", "sourceUrl", "https://"} {
		if strings.Contains(string(encoded), forbidden) {
			t.Fatalf("report contains %q: %s", forbidden, encoded)
		}
	}
}

func TestVerifyStopsAtFailedPhaseWithStructuredContext(t *testing.T) {
	cause := errors.New("unbounded body https://example.invalid/?token=secret")
	tests := []struct {
		name          string
		fail          string
		wantPhase     string
		wantArtifact  string
		wantRule      string
		wantCallCount int
	}{
		{name: "catalog", fail: phaseCatalog, wantPhase: "catalog/references", wantArtifact: "fragment.yaml", wantRule: "reference-escape", wantCallCount: 1},
		{name: "source lint diagnostic", fail: phaseSourceLint, wantPhase: phaseSourceLint, wantArtifact: "source.yaml", wantRule: "operation-summary", wantCallCount: 2},
		{name: "bundle lint error", fail: phaseBundleLint, wantPhase: phaseBundleLint, wantArtifact: "openapi.bundle.yaml", wantRule: "openapi-load-or-validation", wantCallCount: 4},
		{name: "stale bundle", fail: phaseBundleFreshness, wantPhase: phaseBundleFreshness, wantArtifact: "openapi.bundle.yaml", wantRule: "bundle-stale", wantCallCount: 3},
		{name: "auditor evidence", fail: phaseAuditorEvidence, wantPhase: phaseAuditorEvidence, wantArtifact: "auditor-2026-07-18.json", wantRule: "sanitized-evidence-manifest", wantCallCount: 5},
		{name: "release guard", fail: phaseReleaseGuard, wantPhase: phaseReleaseGuard, wantArtifact: "verify.yml", wantRule: "permissions are read-only", wantCallCount: 6},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			deps := dependencies{
				validateCatalog: func(guide.CatalogOptions) (guide.CatalogReport, error) {
					calls++
					if test.fail == phaseCatalog {
						return guide.CatalogReport{}, &guide.CatalogError{Diagnostic: guide.CatalogDiagnostic{
							Phase: "references", Rule: "reference-escape", Artifact: "fragment.yaml", Message: "reference rejected",
						}}
					}
					return guide.CatalogReport{OpenAPI: "3.2.0"}, nil
				},
				lint: func(artifact string) ([]openapispec.LintDiagnostic, error) {
					calls++
					if test.fail == phaseSourceLint && filepath.Base(artifact) == "openapi.yaml" {
						return []openapispec.LintDiagnostic{{Rule: "operation-summary", Artifact: "source.yaml", Operation: "getCompany", Location: "#/paths/~1company/get"}}, nil
					}
					if test.fail == phaseBundleLint && filepath.Base(artifact) == "openapi.bundle.yaml" {
						return nil, cause
					}
					return nil, nil
				},
				checkFresh: func(string, string) error {
					calls++
					if test.fail == phaseBundleFreshness {
						return fmt.Errorf("%w: unsafe detail", openapispec.ErrBundleStale)
					}
					return nil
				},
				checkEvidence: func(string) error {
					calls++
					if test.fail == phaseAuditorEvidence {
						return cause
					}
					return nil
				},
				checkRelease: func(string) error {
					calls++
					if test.fail == phaseReleaseGuard {
						return &releaseguard.Error{Artifact: verifyWorkflowArtifactForTest, Invariant: "permissions are read-only", Cause: cause}
					}
					return nil
				},
			}

			_, err := verifyWith(t.TempDir(), deps)
			var verifyError *Error
			if !errors.As(err, &verifyError) {
				t.Fatalf("error = %v, want *Error", err)
			}
			if verifyError.Phase != test.wantPhase || filepath.Base(verifyError.Artifact) != test.wantArtifact || verifyError.Rule != test.wantRule {
				t.Fatalf("error = %#v", verifyError)
			}
			if calls != test.wantCallCount {
				t.Fatalf("calls = %d, want %d", calls, test.wantCallCount)
			}
			if test.fail == phaseSourceLint && (verifyError.Operation != "getCompany" || verifyError.Location != "#/paths/~1company/get") {
				t.Fatalf("lint context = %#v", verifyError)
			}
			if strings.Contains(verifyError.Error(), "https://") || strings.Contains(verifyError.Error(), "secret") || strings.Contains(verifyError.Error(), "body") {
				t.Fatalf("error leaked cause: %v", verifyError)
			}
			if test.fail == phaseBundleLint && !errors.Is(err, cause) {
				t.Fatalf("error does not preserve cause: %v", err)
			}
		})
	}
}

func TestVerificationErrorBoundsContext(t *testing.T) {
	err := contextualFailure("https://unsafe.invalid/phase", "artifact\nbody", strings.Repeat("r", 1025), "operation\nbody", "https://unsafe.invalid/location", errors.New("secret"))
	if err.Phase != "unknown" || err.Artifact != "unknown-artifact" || err.Rule != "unknown-rule" {
		t.Fatalf("error context = %#v", err)
	}
	if err.Operation != "" || err.Location != "" {
		t.Fatalf("optional error context = %#v", err)
	}
	if err.Error() != "verification failed: phase=unknown artifact=unknown-artifact rule=unknown-rule" {
		t.Fatalf("error = %q", err.Error())
	}
}

func TestVerifyReportsMissingBundleBeforeTryingToLintIt(t *testing.T) {
	bundleLintCalled := false
	deps := dependencies{
		validateCatalog: func(guide.CatalogOptions) (guide.CatalogReport, error) {
			return guide.CatalogReport{OpenAPI: "3.2.0"}, nil
		},
		lint: func(artifact string) ([]openapispec.LintDiagnostic, error) {
			if filepath.Base(artifact) == "openapi.bundle.yaml" {
				bundleLintCalled = true
				return nil, errors.New("missing bundle reached lint")
			}
			return nil, nil
		},
		checkFresh:    func(string, string) error { return openapispec.ErrBundleMissing },
		checkEvidence: func(string) error { return nil },
		checkRelease:  func(string) error { return nil },
	}

	_, err := verifyWith(t.TempDir(), deps)
	var verifyError *Error
	if !errors.As(err, &verifyError) || verifyError.Rule != "bundle-missing" {
		t.Fatalf("error = %#v", err)
	}
	if bundleLintCalled {
		t.Fatal("missing bundle was linted before freshness reported it")
	}
}

func TestVerifyAcceptedRepository(t *testing.T) {
	report, err := Verify(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report.PassedPhases, passedPhases) || report.Catalog.OpenAPI != "3.2.0" || report.Catalog.LogicalEndpoints == 0 {
		t.Fatalf("report = %#v", report)
	}
}

const verifyWorkflowArtifactForTest = ".github/workflows/verify.yml"

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("runtime.Caller failed")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}
