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

func TestValidateEvidenceRejectsNoncanonicalSanitizedClaims(t *testing.T) {
	t.Run("older document selection", func(t *testing.T) {
		report := committedEvidence(t)
		selected := &report.Documents[0]
		var replacement FilingEvidence
		for _, search := range report.Searches {
			if search.CorpCode != nuga.CorpCode {
				continue
			}
			for _, page := range search.Pages {
				for _, filing := range page.Filings {
					if filing.ReceiptNumber != selected.Selection.ReceiptNumber && strings.Contains(filing.ReportName, selected.Selection.ReportPeriod) {
						replacement = filing
					}
				}
			}
		}
		if replacement.ReceiptNumber == "" {
			t.Fatal("fixture has no older filing for the selected period")
		}
		selected.Selection.ReceiptNumber = replacement.ReceiptNumber
		selected.Selection.ExpectedFirm = replacement.FilerName
		selected.Request.ReceiptNumber = replacement.ReceiptNumber
		if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "not canonical") {
			t.Fatalf("older selection error = %v", err)
		}
	})

	t.Run("duplicate receipt across search rows", func(t *testing.T) {
		report := committedEvidence(t)
		filings := report.Searches[0].Pages[0].Filings
		if len(filings) < 2 {
			t.Fatal("fixture has fewer than two search filings")
		}
		filings[1].ReceiptNumber = filings[0].ReceiptNumber
		if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "invalid filing") {
			t.Fatalf("duplicate receipt error = %v", err)
		}
	})

	t.Run("negative entry matches", func(t *testing.T) {
		report := committedEvidence(t)
		document := &report.Documents[3]
		entry := &document.Entries[0]
		document.ExpectedMatches += -1 - entry.ExpectedFirmMatches
		entry.ExpectedFirmMatches = -1
		if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "invalid entry evidence") {
			t.Fatalf("negative entry count error = %v", err)
		}
	})

	t.Run("unbounded entry matches", func(t *testing.T) {
		report := committedEvidence(t)
		entry := &report.Documents[3].Entries[0]
		entry.ExpectedFirmMatches = int(^uint(0) >> 1)
		if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "invalid entry evidence") {
			t.Fatalf("unbounded entry count error = %v", err)
		}
	})

	t.Run("placeholder distinct value", func(t *testing.T) {
		report := committedEvidence(t)
		for index := range report.Structured {
			if len(report.Structured[index].DistinctAuditors) > 0 {
				report.Structured[index].DistinctAuditors[0] = "-"
				if err := validateEvidence(report); err == nil || !strings.Contains(err.Error(), "invalid distinct values") {
					t.Fatalf("placeholder distinct value error = %v", err)
				}
				return
			}
		}
		t.Fatal("fixture has no substantive structured auditor")
	})
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
