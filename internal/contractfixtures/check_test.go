package contractfixtures

import (
	"crypto/sha256"
	"encoding/hex"
	"os"
	"path/filepath"
	"runtime"
	"testing"
)

func TestVerifyBodyRequiresExactDeclaredSize(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, "bodies"), 0o700); err != nil {
		t.Fatal(err)
	}
	body := []byte(`{"status":"000"}`)
	path := filepath.Join(root, "bodies", "success.json")
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	digest := sha256.Sum256(body)
	item := responseCase{
		ID:     "exact-size",
		File:   "bodies/success.json",
		SHA256: hex.EncodeToString(digest[:]),
		Bytes:  int64(len(body)),
	}
	if err := verifyBody(root, item); err != nil {
		t.Fatalf("exact declared size rejected: %v", err)
	}
	item.Bytes++
	if err := verifyBody(root, item); err == nil {
		t.Fatal("incorrect declared size accepted")
	}
	item.Bytes = 0
	if err := verifyBody(root, item); err == nil {
		t.Fatal("undeclared size accepted")
	}
}

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

func TestCompleteObservationProvenanceRequiresTimestamp(t *testing.T) {
	value := provenance{
		Kind: "empirical-summary", ObservedAt: "2026-07-18", SourceCommit: "abc123",
		RequestCondition: "read-only", HTTPStatus: 200, ContentTypeHeader: "application/json",
		APIStatus: "000", SampledPhysicalOperations: []string{"get_company_json"},
	}
	if !completeObservationProvenance(value) {
		t.Fatal("complete empirical provenance was rejected")
	}
	value.ObservedAt = ""
	if completeObservationProvenance(value) {
		t.Fatal("empirical provenance without observedAt was accepted")
	}
}

func TestVerifyBodyInventoryRejectsOrphanAndOutsideReference(t *testing.T) {
	root := t.TempDir()
	bodies := filepath.Join(root, "bodies")
	if err := os.Mkdir(bodies, 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bodies, "one.json"), []byte("{}"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := verifyBodyInventory(root, map[string]struct{}{}); err == nil {
		t.Fatal("orphan fixture body was accepted")
	}
	if err := verifyBodyInventory(root, map[string]struct{}{"bodies/one.json": {}, "manifest.json": {}}); err == nil {
		t.Fatal("manifest reference outside the body inventory was accepted")
	}
	if err := verifyBodyInventory(root, map[string]struct{}{"bodies/one.json": {}}); err != nil {
		t.Fatalf("exact body inventory was rejected: %v", err)
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
