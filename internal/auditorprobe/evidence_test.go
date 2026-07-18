package auditorprobe

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestCommittedAuditorEvidenceIsCompleteAndSanitized(t *testing.T) {
	path := filepath.Join("..", "..", "docs", "api", "evidence", "auditor-2026-07-18.json")
	if err := ValidateEvidenceFile(path); err != nil {
		t.Fatal(err)
	}
}

func TestValidateEvidenceFileRejectsUnknownFields(t *testing.T) {
	path := filepath.Join(t.TempDir(), "evidence.json")
	if err := os.WriteFile(path, []byte(`{"schemaVersion":1,"rawBody":"forbidden"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := ValidateEvidenceFile(path); err == nil {
		t.Fatal("unknown evidence field was accepted")
	}
}

func TestValidateEvidenceRejectsStatesOutsideProbeBounds(t *testing.T) {
	report := committedEvidence(t)
	report.Structured[0].Response.HTTPStatus = 201
	if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "structured response") {
		t.Fatalf("non-200 structured response error = %v", err)
	}

	report = committedEvidence(t)
	report.Searches[0].Pages[0].Response.BodyBytes = maxJSONBody + 1
	if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "bounded response evidence") {
		t.Fatalf("oversized search response error = %v", err)
	}

	report = committedEvidence(t)
	report.Documents[0].ExpandedBytes = maxArchiveExpanded + 1
	if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "verified auditor section") {
		t.Fatalf("oversized archive expansion error = %v", err)
	}
}

func committedEvidence(t *testing.T) Report {
	t.Helper()
	path := filepath.Join("..", "..", "docs", "api", "evidence", "auditor-2026-07-18.json")
	encoded, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	var report Report
	if err := json.Unmarshal(encoded, &report); err != nil {
		t.Fatal(err)
	}
	return report
}
