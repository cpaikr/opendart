package cli

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
	"testing"
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

type failingWriter struct{}

func (failingWriter) Write([]byte) (int, error) {
	return 0, errors.New("write failed")
}
