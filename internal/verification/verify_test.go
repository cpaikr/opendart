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
	"github.com/cpaikr/opendart/internal/liveconformance"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/releaseguard"
	"github.com/cpaikr/opendart/internal/sdkgen"
	"github.com/cpaikr/opendart/internal/sdkgen/model"
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
		checkLive: func(root string) error {
			calls = append(calls, phaseLiveConformance+":"+filepath.Base(root))
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
		checkRustSDK: func(source string, output sdkgen.RustOutputs) error {
			calls = append(calls, phaseRustSDKFreshness+":"+filepath.Base(source)+":"+filepath.Base(output.SDK)+":"+filepath.Base(output.CLI))
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
		"rust-sdk-freshness:openapi.yaml:generated:generated",
		"live-conformance-preflight:repository",
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

func TestLiveConformanceFailurePreservesSanitizedContext(t *testing.T) {
	cause := &liveconformance.Error{Failure: liveconformance.Failure{Code: "invalid-primary-assertion", Stage: "preflight", Operation: "GET /company.json application/json"}}
	got := liveConformanceFailure(cause)
	if got.Phase != phaseLiveConformance || got.Artifact != "live conformance inventory" || got.Rule != "invalid-primary-assertion" || got.Operation != cause.Failure.Operation || !errors.Is(got, cause) {
		t.Fatalf("failure = %#v", got)
	}

	nested := failure("nested-phase", "nested-artifact", "nested-rule", errors.New("detail"))
	got = liveConformanceFailure(nested)
	if got.Phase != phaseLiveConformance || got.Artifact != "nested-artifact" || got.Rule != "nested-rule" {
		t.Fatalf("nested failure = %#v", got)
	}

	unknown := errors.New("unclassified detail")
	got = liveConformanceFailure(unknown)
	if got.Phase != phaseLiveConformance || got.Artifact != "live conformance repository" || got.Rule != "unknown-rule" || !errors.Is(got, unknown) {
		t.Fatalf("unclassified failure = %#v", got)
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
		wantOperation string
		wantLocation  string
		rustError     error
		wantCallCount int
	}{
		{name: "catalog", fail: phaseCatalog, wantPhase: "catalog/references", wantArtifact: "fragment.yaml", wantRule: "reference-escape", wantCallCount: 1},
		{name: "source lint diagnostic", fail: phaseSourceLint, wantPhase: phaseSourceLint, wantArtifact: "source.yaml", wantRule: "operation-summary", wantOperation: "getCompany", wantLocation: "#/paths/~1company/get", wantCallCount: 2},
		{name: "bundle lint error", fail: phaseBundleLint, wantPhase: phaseBundleLint, wantArtifact: "openapi.bundle.yaml", wantRule: "openapi-load-or-validation", wantCallCount: 4},
		{name: "stale bundle", fail: phaseBundleFreshness, wantPhase: phaseBundleFreshness, wantArtifact: "openapi.bundle.yaml", wantRule: "bundle-stale", wantCallCount: 3},
		{name: "Rust SDK freshness", fail: phaseRustSDKFreshness, wantPhase: phaseRustSDKFreshness, wantArtifact: "generated", wantRule: "generated-stale", wantCallCount: 5},
		{name: "Rust SDK model context", fail: phaseRustSDKFreshness, wantPhase: phaseRustSDKFreshness, wantArtifact: "generated", wantRule: "unsupported-response-schema", wantOperation: "get_company_json", wantLocation: "/company.json/get/responses/default", rustError: &openapispec.SDKSurfaceError{Rule: "unsupported-response-schema", Operation: "get_company_json", Location: "/company.json/get/responses/default", Detail: "const"}, wantCallCount: 5},
		{name: "Rust SDK normalized model context", fail: phaseRustSDKFreshness, wantPhase: phaseRustSDKFreshness, wantArtifact: "generated", wantRule: "rust-name-collision", wantOperation: "get_company_json", wantLocation: "/logicalOperations/0", rustError: &model.Error{Rule: "rust-name-collision", Operation: "get_company_json", Location: "/logicalOperations/0", Detail: "collision"}, wantCallCount: 5},
		{name: "live conformance", fail: phaseLiveConformance, wantPhase: phaseLiveConformance, wantArtifact: "live conformance repository", wantRule: "unknown-rule", wantCallCount: 6},
		{name: "auditor evidence", fail: phaseAuditorEvidence, wantPhase: phaseAuditorEvidence, wantArtifact: "auditor-2026-07-18.json", wantRule: "sanitized-evidence-manifest", wantCallCount: 7},
		{name: "release guard", fail: phaseReleaseGuard, wantPhase: phaseReleaseGuard, wantArtifact: "verify.yml", wantRule: "permissions are read-only", wantCallCount: 8},
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
				checkLive: func(string) error {
					calls++
					if test.fail == phaseLiveConformance {
						return cause
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
				checkRustSDK: func(string, sdkgen.RustOutputs) error {
					calls++
					if test.fail == phaseRustSDKFreshness {
						if test.rustError != nil {
							return test.rustError
						}
						return sdkgen.ErrGeneratedStale
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
			if verifyError.Operation != test.wantOperation || verifyError.Location != test.wantLocation {
				t.Fatalf("structured context = %#v", verifyError)
			}
			if calls != test.wantCallCount {
				t.Fatalf("calls = %d, want %d", calls, test.wantCallCount)
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
		checkLive:     func(string) error { return nil },
		checkEvidence: func(string) error { return nil },
		checkRelease:  func(string) error { return nil },
		checkRustSDK:  func(string, sdkgen.RustOutputs) error { return nil },
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
