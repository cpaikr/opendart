package cli

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
	"time"

	guidesync "github.com/cpaikr/opendart/internal/guide"
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
	if !strings.Contains(stdout.String(), "compatibility") {
		t.Fatalf("stdout = %q", stdout.String())
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

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
