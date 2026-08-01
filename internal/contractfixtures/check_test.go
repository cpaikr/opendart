package contractfixtures

import (
	"path/filepath"
	"runtime"
	"testing"
)

func TestCheckRepositoryCorpus(t *testing.T) {
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	root := filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
	if err := Check(root); err != nil {
		t.Fatal(err)
	}
}

func TestResponseOutcomeMatchesHTTPStatus(t *testing.T) {
	tests := []struct {
		outcome string
		status  int
		want    bool
	}{
		{outcome: "typed-success", status: 200, want: true},
		{outcome: "http-status-failure", status: 500, want: true},
		{outcome: "http-status-failure", status: 200, want: false},
		{outcome: "invented", status: 200, want: false},
	}
	for _, test := range tests {
		if got := responseOutcomeMatchesStatus(test.outcome, test.status); got != test.want {
			t.Fatalf("responseOutcomeMatchesStatus(%q, %d) = %v, want %v", test.outcome, test.status, got, test.want)
		}
	}
}

func TestCheckCoverageRejectsVacuousAndReducedCorpora(t *testing.T) {
	if err := checkCoverage(corpus{}); err == nil {
		t.Fatal("empty corpus should not satisfy required coverage")
	}
	complete := corpus{
		Observations: []observation{{ID: "binary-observation"}},
		RequestCases: []requestCase{
			{Representation: "json"},
			{Representation: "zip"},
		},
		ResponseCases: []responseCase{
			{PhysicalOperation: "get_company_json", Outcome: "typed-success"},
			{PhysicalOperation: "get_company_xml", Outcome: "typed-success"},
			{PhysicalOperation: "get_company_json", Outcome: "source-status"},
			{PhysicalOperation: "get_company_json", Outcome: "http-status-failure"},
			{PhysicalOperation: "get_company_xml", Outcome: "decode-failure"},
			{PhysicalOperation: "get_company_json", Outcome: "envelope-failure"},
			{PhysicalOperation: "get_corpCode_xml", Outcome: "source-status", Provenance: provenance{Observation: "binary-observation"}},
		},
	}
	if err := checkCoverage(complete); err != nil {
		t.Fatalf("complete protocol-family coverage rejected: %v", err)
	}
	complete.ResponseCases = complete.ResponseCases[:6]
	if err := checkCoverage(complete); err == nil {
		t.Fatal("corpus without the binary alternate case should be rejected")
	}
}
