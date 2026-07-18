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

	guidesync "github.com/cpaikr/opendart/internal/guide"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
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

func TestRunPrintsHelp(t *testing.T) {
	var stdout bytes.Buffer
	var stderr bytes.Buffer

	if code := Run([]string{"help"}, &stdout, &stderr); code != 0 {
		t.Fatalf("Run() code = %d, want 0", code)
	}
	for _, command := range []string{"sync", "catalog", "lint", "bundle", "compatibility"} {
		if !strings.Contains(stdout.String(), command) {
			t.Fatalf("stdout does not list %q: %q", command, stdout.String())
		}
	}
}

func TestRunCompatibility(t *testing.T) {
	root := filepath.Join("..", "openapi", "testdata", "compatibility", "openapi.yaml")
	t.Run("success", func(t *testing.T) {
		var stdout bytes.Buffer
		var stderr bytes.Buffer
		if code := Run([]string{"compatibility", "--root", root, "--baseline", root}, &stdout, &stderr); code != 0 {
			t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
		}
		var report struct {
			DocumentValid           bool `json:"documentValid"`
			BundleSemanticChanges   int  `json:"bundleSemanticChanges"`
			BaselineSemanticChanges int  `json:"baselineSemanticChanges"`
		}
		if err := json.Unmarshal(stdout.Bytes(), &report); err != nil {
			t.Fatalf("decode report: %v", err)
		}
		if !report.DocumentValid || report.BundleSemanticChanges != 0 || report.BaselineSemanticChanges != 0 {
			t.Fatalf("report = %+v", report)
		}
	})

	for _, test := range []struct {
		name string
		args []string
	}{
		{name: "invalid root", args: []string{"compatibility", "--root", "missing-openapi.yaml"}},
		{name: "invalid baseline", args: []string{"compatibility", "--root", root, "--baseline", "missing-bundle.yaml"}},
	} {
		t.Run(test.name, func(t *testing.T) {
			var stdout bytes.Buffer
			var stderr bytes.Buffer
			if code := Run(test.args, &stdout, &stderr); code != 1 {
				t.Fatalf("Run() code = %d, stderr = %q", code, stderr.String())
			}
			if !strings.Contains(stderr.String(), "compatibility:") {
				t.Fatalf("stderr = %q", stderr.String())
			}
		})
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

func TestNewCommandsRejectPositionalArguments(t *testing.T) {
	for _, command := range []string{"catalog", "lint", "bundle"} {
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
