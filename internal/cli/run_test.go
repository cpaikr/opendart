package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"

	"github.com/cpaikr/opendart/internal/auditorprobe"
	"github.com/cpaikr/opendart/internal/driftnotifier"
	guidesync "github.com/cpaikr/opendart/internal/guide"
	"github.com/cpaikr/opendart/internal/liveconformance"
	"github.com/cpaikr/opendart/internal/livenotifier"
	"github.com/cpaikr/opendart/internal/multicompanyprobe"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/sdkgen"
	"github.com/cpaikr/opendart/internal/verification"
)

func TestRunRejectsUnknownCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"unknown"}, &stdout, &stderr); code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unknown command "unknown"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunRejectsRetiredCompatibilityCommand(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"compatibility"}, &stdout, &stderr); code != 2 {
		t.Fatalf("Run() code = %d, want 2", code)
	}
	if !strings.Contains(stderr.String(), `unknown command "compatibility"`) {
		t.Fatalf("stderr = %q", stderr.String())
	}
}

func TestRunPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	for _, command := range []string{"sync", "catalog", "lint", "bundle", "generate-sdk", "verify", "guide-drift", "guide-drift-notify", "live-conformance", "live-conformance-notify", "probe-multi-company", "probe-auditor-evidence"} {
		if !strings.Contains(stdout.String(), command) {
			t.Fatalf("stdout does not list %q: %q", command, stdout.String())
		}
	}
}

func TestRunGuideDriftEmitsChangedAndErrorReports(t *testing.T) {
	changed := guidesync.DriftReport{
		SchemaVersion: guidesync.DriftReportSchemaVersion,
		Kind:          guidesync.DriftReportKind,
		Outcome:       "changed",
	}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runGuideDriftWith(context.Background(), []string{"--repository-root", "repository"}, &stdout, &stderr, func(_ context.Context, root string) (guidesync.DriftReport, error) {
		if root != "repository" {
			t.Fatalf("root = %q", root)
		}
		return changed, nil
	})
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"outcome": "changed"`) {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}

	stdout.Reset()
	errorReport := changed
	errorReport.Outcome = "error"
	errorReport.Failure = &guidesync.DriftFailure{Code: "acquisition-failed", Stage: "acquisition"}
	code = runGuideDriftWith(context.Background(), nil, &stdout, &stderr, func(context.Context, string) (guidesync.DriftReport, error) {
		return errorReport, errors.New("unsafe source body and URL")
	})
	if code != 1 || !strings.Contains(stdout.String(), guidesync.DriftReportKind) || stderr.String() != "guide-drift: processing failed\n" || strings.Contains(stderr.String(), "unsafe") {
		t.Fatalf("code=%d stdout=%q stderr=%q", code, stdout.String(), stderr.String())
	}
}

func TestRunProbeAuditorEvidence(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	report := auditorprobe.Report{SchemaVersion: 1, RequestBudget: auditorprobe.RequestBudget{Maximum: 60, Used: 24}}
	code := runProbeAuditorEvidenceWith(context.Background(), nil, &stdout, &stderr, "/repository", func(_ context.Context, root string) (auditorprobe.Report, error) {
		if root != "/repository" {
			t.Fatalf("root = %q", root)
		}
		return report, nil
	})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var actual auditorprobe.Report
	if err := json.Unmarshal(stdout.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.SchemaVersion != 1 || actual.RequestBudget.Used != 24 {
		t.Fatalf("report = %#v", actual)
	}
}

func TestRunProbeAuditorEvidenceEmitsSanitizedDiagnostic(t *testing.T) {
	var stderr bytes.Buffer
	request := &auditorprobe.RequestCoordinate{Endpoint: "document", ReceiptNumber: "20240101000001"}
	code := runProbeAuditorEvidenceWith(context.Background(), nil, &bytes.Buffer{}, &stderr, "/repository", func(context.Context, string) (auditorprobe.Report, error) {
		return auditorprobe.Report{}, &auditorprobe.Error{Message: "document inspection failed", Request: request}
	})
	if code != 1 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var diagnostic struct {
		Error   string                          `json:"error"`
		Message string                          `json:"message"`
		Request *auditorprobe.RequestCoordinate `json:"request"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diagnostic); err != nil {
		t.Fatal(err)
	}
	if diagnostic.Error != "ProbeError" || diagnostic.Message != "document inspection failed" || diagnostic.Request == nil || diagnostic.Request.Endpoint != "document" {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}

func TestRunProbeAuditorEvidenceRejectsArgumentsAndHidesUnexpectedErrors(t *testing.T) {
	var stderr bytes.Buffer
	runner := func(context.Context, string) (auditorprobe.Report, error) {
		t.Fatal("runner should not be called for positional arguments")
		return auditorprobe.Report{}, nil
	}
	if code := runProbeAuditorEvidenceWith(context.Background(), []string{"unexpected"}, &bytes.Buffer{}, &stderr, "/repository", runner); code != 2 || !strings.Contains(stderr.String(), "does not accept positional arguments") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}

	stderr.Reset()
	if code := runProbeAuditorEvidenceWith(context.Background(), nil, &bytes.Buffer{}, &stderr, "/repository", func(context.Context, string) (auditorprobe.Report, error) {
		return auditorprobe.Report{}, errors.New("unexpected secret and authenticated URL")
	}); code != 1 || strings.Contains(stderr.String(), "unexpected secret") || !strings.Contains(stderr.String(), "Unexpected auditor evidence probe failure") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunProbeMultiCompany(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	report := multicompanyprobe.Report{SchemaVersion: 1, RequestCount: 10}
	code := runProbeMultiCompanyWith(context.Background(), nil, &stdout, &stderr, "/repository", func(_ context.Context, root string) (multicompanyprobe.Report, error) {
		if root != "/repository" {
			t.Fatalf("root = %q", root)
		}
		return report, nil
	})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var actual multicompanyprobe.Report
	if err := json.Unmarshal(stdout.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if actual.SchemaVersion != 1 || actual.RequestCount != 10 {
		t.Fatalf("report = %#v", actual)
	}
}

func TestRunProbeMultiCompanyEmitsSanitizedDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runProbeMultiCompanyWith(context.Background(), nil, &stdout, &stderr, "/repository", func(context.Context, string) (multicompanyprobe.Report, error) {
		return multicompanyprobe.Report{}, &multicompanyprobe.Error{Message: "request failed", Context: map[string]any{"endpoint": "fnlttMultiAcnt"}}
	})
	if code != 1 || stdout.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
	var diagnostic struct {
		Error   string         `json:"error"`
		Message string         `json:"message"`
		Context map[string]any `json:"context"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diagnostic); err != nil {
		t.Fatal(err)
	}
	if diagnostic.Error != "ProbeError" || diagnostic.Message != "request failed" || diagnostic.Context["endpoint"] != "fnlttMultiAcnt" {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}

func TestRunProbeMultiCompanyRejectsArguments(t *testing.T) {
	var stderr bytes.Buffer
	code := runProbeMultiCompanyWith(context.Background(), []string{"unexpected"}, &bytes.Buffer{}, &stderr, "/repository", func(context.Context, string) (multicompanyprobe.Report, error) {
		t.Fatal("runner should not be called")
		return multicompanyprobe.Report{}, nil
	})
	if code != 2 || !strings.Contains(stderr.String(), "does not accept positional arguments") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunProbeMultiCompanyHidesUnexpectedErrors(t *testing.T) {
	var stderr bytes.Buffer
	code := runProbeMultiCompanyWith(context.Background(), nil, &bytes.Buffer{}, &stderr, "/repository", func(context.Context, string) (multicompanyprobe.Report, error) {
		return multicompanyprobe.Report{}, errors.New("unexpected secret and authenticated URL")
	})
	if code != 1 || strings.Contains(stderr.String(), "unexpected secret") || !strings.Contains(stderr.String(), "Unexpected serialization probe failure") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunProbeMultiCompanyReportsOutputFailure(t *testing.T) {
	var stderr bytes.Buffer
	code := runProbeMultiCompanyWith(context.Background(), nil, failingWriter{}, &stderr, "/repository", func(context.Context, string) (multicompanyprobe.Report, error) {
		return multicompanyprobe.Report{SchemaVersion: 1}, nil
	})
	if code != 1 || !strings.Contains(stderr.String(), "write probe-multi-company report") {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
}

func TestRunReportsOutputFailure(t *testing.T) {
	if code := Run([]string{"help"}, failingWriter{}, &bytes.Buffer{}); code != 1 {
		t.Fatalf("Run() code = %d, want 1", code)
	}
}

func TestRunSyncEmitsReport(t *testing.T) {
	repository := t.TempDir()
	var received guidesync.SyncOptions
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runSyncWith(context.Background(), []string{"--checked-at", "2026-07-18"}, &stdout, &stderr, repository, time.Now(), func(_ context.Context, options guidesync.SyncOptions) (guidesync.SyncReport, error) {
		received = options
		return guidesync.SyncReport{Output: options.Output, CheckedAt: options.CheckedAt, LogicalEndpoints: 85}, nil
	})
	if code != 0 {
		t.Fatalf("runSyncWith() code = %d, stderr = %q", code, stderr.String())
	}
	if received.CheckedAt != "2026-07-18" || received.Output != filepath.Join(repository, "openapi") {
		t.Fatalf("options = %#v", received)
	}
	if !strings.Contains(stdout.String(), `"logicalEndpoints": 85`) {
		t.Fatalf("stdout = %q", stdout.String())
	}
}

func TestRunSyncEmitsNestedSourceContext(t *testing.T) {
	repository := t.TempDir()
	inner := &guidesync.SourceError{Message: "request failed", Context: map[string]any{"status": 503, "attempt": 3}}
	outer := &guidesync.SourceError{Message: "group failed", Context: map[string]any{"group": "DS002"}, Cause: inner}
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runSyncWith(context.Background(), []string{"--checked-at", "2026-07-18"}, &stdout, &stderr, repository, time.Now(), func(context.Context, guidesync.SyncOptions) (guidesync.SyncReport, error) {
		return guidesync.SyncReport{}, outer
	})
	if code != 1 {
		t.Fatalf("code = %d", code)
	}
	var diagnostic map[string]any
	if err := json.Unmarshal(stderr.Bytes(), &diagnostic); err != nil {
		t.Fatal(err)
	}
	if diagnostic["group"] != "DS002" || diagnostic["status"] != float64(503) || diagnostic["attempt"] != float64(3) {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}

func TestRunCatalogEmitsReportAndForwardsMode(t *testing.T) {
	var received guidesync.CatalogOptions
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runCatalogWith([]string{"--root", "spec.yaml", "--structural-only"}, &stdout, &stderr, func(options guidesync.CatalogOptions) (guidesync.CatalogReport, error) {
		received = options
		return guidesync.CatalogReport{Root: options.Root, OpenAPI: "3.2.0", LogicalEndpoints: 1}, nil
	})
	if code != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if received.Root != "spec.yaml" || !received.StructuralOnly {
		t.Fatalf("options = %#v", received)
	}
	var report guidesync.CatalogReport
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
		t.Fatal(err)
	}
	if report.OpenAPI != "3.2.0" || report.LogicalEndpoints != 1 {
		t.Fatalf("report = %#v", report)
	}
}

func TestRunCatalogEmitsStructuredDiagnostic(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	diagnostic := guidesync.CatalogDiagnostic{Rule: "reference", Phase: "references", Artifact: "spec.yaml", Message: "invalid reference"}
	code := runCatalogWith(nil, &stdout, &stderr, func(guidesync.CatalogOptions) (guidesync.CatalogReport, error) {
		return guidesync.CatalogReport{}, &guidesync.CatalogError{Diagnostic: diagnostic}
	})
	if code != 1 || stdout.Len() != 0 {
		t.Fatalf("code = %d, stdout = %q", code, stdout.String())
	}
	var actual guidesync.CatalogDiagnostic
	if err := json.Unmarshal(stderr.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if actual != diagnostic {
		t.Fatalf("diagnostic = %#v", actual)
	}
}

func TestRunLintEmitsSuccessAndDeterministicDiagnostics(t *testing.T) {
	t.Run("success", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		code := runLintWith([]string{"--root", "spec.yaml"}, &stdout, &stderr, func(root string) ([]openapispec.LintDiagnostic, error) {
			if root != "spec.yaml" {
				t.Fatalf("root = %q", root)
			}
			return nil, nil
		})
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		var report struct {
			Root  string `json:"root"`
			Valid bool   `json:"valid"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatal(err)
		}
		if report.Root != "spec.yaml" || !report.Valid {
			t.Fatalf("report = %#v", report)
		}
	})

	t.Run("policy diagnostics", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		diagnostics := []openapispec.LintDiagnostic{
			{Rule: "a", Artifact: "spec.yaml", Location: "/a", Message: "first"},
			{Rule: "b", Artifact: "spec.yaml", Location: "/b", Message: "second"},
		}
		code := runLintWith(nil, &stdout, &stderr, func(string) ([]openapispec.LintDiagnostic, error) {
			return diagnostics, nil
		})
		if code != 1 || stdout.Len() != 0 {
			t.Fatalf("code = %d, stdout = %q", code, stdout.String())
		}
		var actual []openapispec.LintDiagnostic
		if err := json.Unmarshal(stderr.Bytes(), &actual); err != nil {
			t.Fatal(err)
		}
		if !slices.Equal(actual, diagnostics) {
			t.Fatalf("diagnostics = %#v", actual)
		}
	})
}

func TestRunBundleRequiresOutputAndForwardsPaths(t *testing.T) {
	t.Run("missing output", func(t *testing.T) {
		var stderr bytes.Buffer
		called := false
		code := runBundleWith(nil, &bytes.Buffer{}, &stderr, func(string, string) error {
			called = true
			return nil
		})
		if code != 2 || called || !strings.Contains(stderr.String(), "--output is required") {
			t.Fatalf("code = %d, called = %v, stderr = %q", code, called, stderr.String())
		}
	})

	t.Run("success", func(t *testing.T) {
		var root, output string
		var stderr bytes.Buffer
		code := runBundleWith([]string{"--root", "spec.yaml", "--output", "bundle.yaml"}, &bytes.Buffer{}, &stderr, func(receivedRoot, receivedOutput string) error {
			root, output = receivedRoot, receivedOutput
			return nil
		})
		if code != 0 || stderr.Len() != 0 {
			t.Fatalf("code = %d, stderr = %q", code, stderr.String())
		}
		if root != "spec.yaml" || output != "bundle.yaml" {
			t.Fatalf("root = %q, output = %q", root, output)
		}
	})
}

func TestRunGenerateSDKRequiresRustAndForwardsPaths(t *testing.T) {
	var root, output string
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runGenerateSDKWith([]string{"--language", "rust", "--root", "spec.yaml", "--output", "generated"}, &stdout, &stderr, func(receivedRoot, receivedOutput string) (sdkgen.Report, error) {
		root, output = receivedRoot, receivedOutput
		return sdkgen.Report{Language: "rust", SchemaVersion: 1, Checksum: "checksum", Output: receivedOutput}, nil
	})
	if code != 0 || stderr.Len() != 0 || root != "spec.yaml" || output != "generated" {
		t.Fatalf("code = %d, root = %q, output = %q, stderr = %q", code, root, output, stderr.String())
	}
	var report sdkgen.Report
	if err := json.Unmarshal(stdout.Bytes(), &report); err != nil || report.Checksum != "checksum" {
		t.Fatalf("report = %#v, error = %v", report, err)
	}

	for _, args := range [][]string{{"--language", "python", "--output", "generated"}, {"--language", "rust"}} {
		stderr.Reset()
		if code := runGenerateSDKWith(args, &bytes.Buffer{}, &stderr, func(string, string) (sdkgen.Report, error) {
			t.Fatal("runner should not be called for invalid options")
			return sdkgen.Report{}, nil
		}); code != 2 {
			t.Fatalf("args = %#v, code = %d, stderr = %q", args, code, stderr.String())
		}
	}
}

func TestRunVerifyEmitsReportAndForwardsRepositoryRoot(t *testing.T) {
	var received string
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	report := verification.Report{PassedPhases: []string{"catalog"}, Catalog: verification.CatalogSummary{OpenAPI: "3.2.0"}}
	code := runVerifyWith([]string{"--repository-root", "repository"}, &stdout, &stderr, func(root string) (verification.Report, error) {
		received = root
		return report, nil
	})
	if code != 0 || stderr.Len() != 0 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	if received != "repository" {
		t.Fatalf("repository root = %q", received)
	}
	var actual verification.Report
	if err := json.Unmarshal(stdout.Bytes(), &actual); err != nil {
		t.Fatal(err)
	}
	if !slices.Equal(actual.PassedPhases, report.PassedPhases) || actual.Catalog.OpenAPI != "3.2.0" {
		t.Fatalf("report = %#v", actual)
	}

	stderr.Reset()
	code = runVerifyWith(nil, &stdout, &stderr, func(string) (verification.Report, error) {
		return verification.Report{}, &verification.Error{
			Phase: "source-lint", Artifact: "openapi.yaml", Rule: "operation-summary",
			Operation: "getCompany", Location: "#/paths/~1company/get",
		}
	})
	if code != 1 {
		t.Fatalf("code = %d, stderr = %q", code, stderr.String())
	}
	var diagnostic struct {
		Phase     string `json:"phase"`
		Artifact  string `json:"artifact"`
		Rule      string `json:"rule"`
		Operation string `json:"operation"`
		Location  string `json:"location"`
	}
	if err := json.Unmarshal(stderr.Bytes(), &diagnostic); err != nil {
		t.Fatalf("decode diagnostic: %v; stderr = %q", err, stderr.String())
	}
	if diagnostic.Phase != "source-lint" || diagnostic.Artifact != "openapi.yaml" ||
		diagnostic.Rule != "operation-summary" || diagnostic.Operation != "getCompany" ||
		diagnostic.Location != "#/paths/~1company/get" {
		t.Fatalf("diagnostic = %#v", diagnostic)
	}
}

func TestRunLiveConformancePreflightDoesNotCallLiveRunner(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	liveCalled := false
	code := runLiveConformanceWith(context.Background(), []string{"--repository-root", "repository", "--preflight-only"}, &stdout, &stderr,
		func(root string) (liveconformance.PreflightReport, error) {
			if root != "repository" {
				t.Fatalf("root = %q", root)
			}
			return liveconformance.PreflightReport{Valid: true, PrimaryCases: 167, RequestCeiling: 200}, nil
		},
		func(context.Context, string) (liveconformance.Report, error) {
			liveCalled = true
			return liveconformance.Report{}, nil
		})
	if code != 0 || liveCalled || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"valid": true`) {
		t.Fatalf("code = %d, liveCalled = %v, stdout = %q, stderr = %q", code, liveCalled, stdout.String(), stderr.String())
	}
}

func TestRunLiveConformancePreflightFailureIsFixed(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	liveCalled := false
	code := runLiveConformanceWith(context.Background(), []string{"--preflight-only"}, &stdout, &stderr,
		func(string) (liveconformance.PreflightReport, error) {
			return liveconformance.PreflightReport{}, errors.New("unsafe preflight detail")
		},
		func(context.Context, string) (liveconformance.Report, error) {
			liveCalled = true
			return liveconformance.Report{}, nil
		})
	if code != 1 || liveCalled || stdout.Len() != 0 || stderr.String() != "live-conformance: preflight failed\n" || strings.Contains(stderr.String(), "unsafe") {
		t.Fatalf("code = %d, liveCalled = %v, stdout = %q, stderr = %q", code, liveCalled, stdout.String(), stderr.String())
	}
}

func TestRunLiveConformanceEmitsSuccessfulReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	report := liveconformance.Report{SchemaVersion: liveconformance.ReportSchemaVersion, Kind: liveconformance.ReportKind, Outcome: "passed"}
	code := runLiveConformanceWith(context.Background(), []string{"--repository-root", "repository"}, &stdout, &stderr,
		func(string) (liveconformance.PreflightReport, error) { return liveconformance.PreflightReport{}, nil },
		func(_ context.Context, root string) (liveconformance.Report, error) {
			if root != "repository" {
				t.Fatalf("root = %q", root)
			}
			return report, nil
		})
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"outcome": "passed"`) || !strings.Contains(stdout.String(), liveconformance.ReportKind) {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestRunLiveConformanceEmitsSanitizedFailureReport(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	report := liveconformance.Report{SchemaVersion: liveconformance.ReportSchemaVersion, Kind: liveconformance.ReportKind, Outcome: "failed"}
	code := runLiveConformanceWith(context.Background(), nil, &stdout, &stderr,
		func(string) (liveconformance.PreflightReport, error) { return liveconformance.PreflightReport{}, nil },
		func(context.Context, string) (liveconformance.Report, error) {
			return report, errors.New("secret authenticated URL")
		})
	if code != 1 || !strings.Contains(stdout.String(), liveconformance.ReportKind) || stderr.String() != "live-conformance: execution failed\n" || strings.Contains(stderr.String(), "secret") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestRunLiveConformanceNotifyPassesOnlyExplicitMetadataAndEnvironmentToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runLiveConformanceNotifyWith(context.Background(), []string{
		"--report", "report.json",
		"--repository", "cpaikr/opendart",
		"--producer-conclusion", "failure",
		"--artifact-outcome", "success",
		"--run-id", "123",
		"--run-attempt", "2",
	}, &stdout, &stderr, func(name string) string {
		if name != "GITHUB_TOKEN" {
			t.Fatalf("environment name = %q", name)
		}
		return "job-token"
	}, func(_ context.Context, options livenotifier.Options) (livenotifier.Result, error) {
		if options.ReportPath != "report.json" || options.Repository != "cpaikr/opendart" || options.ProducerConclusion != "failure" || options.ArtifactOutcome != "success" || options.RunID != 123 || options.RunAttempt != 2 || options.Token != "job-token" {
			t.Fatalf("options = %#v", options)
		}
		return livenotifier.Result{Action: "updated", IssueNumber: 17}, nil
	})
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"action": "updated"`) || !strings.Contains(stdout.String(), `"issueNumber": 17`) {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestRunLiveConformanceNotifyFailureIsFixed(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runLiveConformanceNotifyWith(context.Background(), nil, &stdout, &stderr, func(string) string {
		return "secret-token"
	}, func(context.Context, livenotifier.Options) (livenotifier.Result, error) {
		return livenotifier.Result{}, errors.New("secret authenticated URL and raw issue body")
	})
	if code != 1 || stdout.Len() != 0 || stderr.String() != "live-conformance-notify: notification failed\n" || strings.Contains(stderr.String(), "secret") || strings.Contains(stderr.String(), "raw") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestRunGuideDriftNotifyPassesOnlyExplicitMetadataAndEnvironmentToken(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runGuideDriftNotifyWith(context.Background(), []string{
		"--report", "report.json",
		"--repository", "cpaikr/opendart",
		"--producer-conclusion", "failure",
		"--artifact-outcome", "success",
		"--run-id", "123",
		"--run-attempt", "2",
	}, &stdout, &stderr, func(name string) string {
		if name != "GITHUB_TOKEN" {
			t.Fatalf("environment name = %q", name)
		}
		return "job-token"
	}, func(_ context.Context, options driftnotifier.Options) (driftnotifier.Result, error) {
		if options.ReportPath != "report.json" || options.Repository != "cpaikr/opendart" || options.ProducerConclusion != "failure" || options.ArtifactOutcome != "success" || options.RunID != 123 || options.RunAttempt != 2 || options.Token != "job-token" {
			t.Fatalf("options = %#v", options)
		}
		return driftnotifier.Result{Action: "updated", IssueNumber: 17}, nil
	})
	if code != 0 || stderr.Len() != 0 || !strings.Contains(stdout.String(), `"action": "updated"`) || !strings.Contains(stdout.String(), `"issueNumber": 17`) {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestRunGuideDriftNotifyFailureIsFixed(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	code := runGuideDriftNotifyWith(context.Background(), nil, &stdout, &stderr, func(string) string {
		return "secret-token"
	}, func(context.Context, driftnotifier.Options) (driftnotifier.Result, error) {
		return driftnotifier.Result{}, errors.New("secret authenticated URL and raw issue body")
	})
	if code != 1 || stdout.Len() != 0 || stderr.String() != "guide-drift-notify: notification failed\n" || strings.Contains(stderr.String(), "secret") || strings.Contains(stderr.String(), "raw") {
		t.Fatalf("code = %d, stdout = %q, stderr = %q", code, stdout.String(), stderr.String())
	}
}

func TestNewCommandsRejectPositionalArguments(t *testing.T) {
	for _, command := range []string{"catalog", "lint", "bundle", "generate-sdk", "verify", "guide-drift", "guide-drift-notify", "live-conformance", "live-conformance-notify"} {
		t.Run(command, func(t *testing.T) {
			var stderr bytes.Buffer
			if code := Run([]string{command, "unexpected"}, &bytes.Buffer{}, &stderr); code != 2 {
				t.Fatalf("code = %d, stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), "does not accept positional arguments") {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
	}
}

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
