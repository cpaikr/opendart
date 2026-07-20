package guide

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

func TestDriftReportsUnchangedAfterOfflineProcessing(t *testing.T) {
	root := driftRepository(t)
	deps := successfulDriftDependencies(t)

	report, err := driftWithDependencies(context.Background(), root, deps)
	if err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "unchanged" || report.Comparison == nil || report.Comparison.TotalChanges != 0 || report.RequestBudget != (RequestBudget{Ceiling: 6, Used: 6}) {
		t.Fatalf("report = %#v", report)
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if _, err := DecodeDriftReport(bytes.NewReader(encoded)); err != nil {
		t.Fatal(err)
	}
}

func TestDriftOfflineFixtureCoversUnchangedAdditionAndRemoval(t *testing.T) {
	fixture := generationFixture(t)
	multiDetailChange := generationFixture(t)
	multiArgumentRemoval := generationFixture(t)
	for endpointIndex := range multiDetailChange {
		endpoint := &multiDetailChange[endpointIndex]
		if endpoint.LogicalOperationID == "DS003-2019017" {
			for argumentIndex := range endpoint.GuideTestRequestArguments {
				if endpoint.GuideTestRequestArguments[argumentIndex].Key == "corp_code" {
					endpoint.GuideTestRequestArguments[argumentIndex].Value = "00126380,00334624"
				}
			}
		}
		for messageIndex := range endpoint.MessageCodes {
			if endpoint.MessageCodes[messageIndex].Code == "021" {
				endpoint.MessageCodes[messageIndex].Description = strings.Replace(endpoint.MessageCodes[messageIndex].Description, "100건", "101건", 1)
			}
		}
	}
	for endpointIndex := range multiArgumentRemoval {
		endpoint := &multiArgumentRemoval[endpointIndex]
		if endpoint.LogicalOperationID != "DS003-2019017" {
			continue
		}
		arguments := endpoint.RequestArguments[:0]
		for _, argument := range endpoint.RequestArguments {
			if argument.Key != "corp_code" {
				arguments = append(arguments, argument)
			}
		}
		endpoint.RequestArguments = arguments
	}
	tests := []struct {
		name      string
		baseline  []Endpoint
		candidate []Endpoint
		outcome   string
	}{
		{name: "unchanged", baseline: fixture, candidate: fixture, outcome: "unchanged"},
		{name: "addition", baseline: fixture[:1], candidate: fixture, outcome: "changed"},
		{name: "removal", baseline: fixture, candidate: fixture[:1], outcome: "changed"},
		{name: "multi-company detail", baseline: fixture, candidate: multiDetailChange, outcome: "changed"},
		{name: "multi-company argument removal", baseline: fixture, candidate: multiArgumentRemoval, outcome: "changed"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := t.TempDir()
			baselineDir := filepath.Join(root, "openapi")
			if err := os.Mkdir(baselineDir, 0o755); err != nil {
				t.Fatal(err)
			}
			if _, err := Generate(test.baseline, GenerateOptions{OutputDir: baselineDir, CheckedAt: "2026-07-17"}); err != nil {
				t.Fatal(err)
			}
			deps := driftDependencies{
				acquire: func(context.Context) (DriftAcquisition, error) {
					budget := len(Groups) + len(test.candidate)
					return DriftAcquisition{Endpoints: test.candidate, RequestBudget: RequestBudget{Ceiling: budget, Used: budget}}, nil
				},
				generate: Generate,
				validate: validateStaging,
				compare:  openapispec.CompareGuideDrift,
				operations: func(root string) ([]openapispec.Operation, error) {
					document, err := openapispec.Load(root)
					if err != nil {
						return nil, err
					}
					defer document.Close()
					catalog, err := document.Operations()
					return catalog.Operations, err
				},
				now: func() time.Time { return time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC) },
			}
			report, err := driftWithDependencies(context.Background(), root, deps)
			if err != nil || report.Outcome != test.outcome {
				t.Fatalf("report=%#v error=%v", report, err)
			}
			if test.outcome == "changed" && (report.Comparison == nil || len(report.Comparison.Findings) == 0 || report.Comparison.Findings[0].OperationID == "") {
				t.Fatalf("changed report lacks operation evidence: %#v", report)
			}
		})
	}
}

func TestDriftReportsSanitizedAdditionRemovalAndTruncation(t *testing.T) {
	root := driftRepository(t)
	deps := successfulDriftDependencies(t)
	details := make([]openapispec.ChangeDetail, MaximumDriftFindings+2)
	for index := range details {
		details[index] = openapispec.ChangeDetail{Location: "#/paths/~1added/get/parameters/" + string(rune('a'+index%26)), Original: "<missing>", New: `"private guide value"`}
	}
	details[1] = openapispec.ChangeDetail{Location: "#/paths/~1removed/get", Original: `{"raw":"private"}`, New: "<missing>"}
	details[len(details)-1] = openapispec.ChangeDetail{Location: "#/paths/~1added/get/`malicious`", Original: "one", New: "two"}
	deps.compare = func(string, string) (openapispec.Comparison, error) {
		return openapispec.Comparison{TotalChanges: len(details), BreakingChanges: 1, Details: details}, nil
	}
	deps.operations = func(root string) ([]openapispec.Operation, error) {
		if strings.Contains(root, "opendart-guide-drift-") {
			return []openapispec.Operation{{Method: "GET", Path: "/added", OperationID: "get_added", LogicalOperationID: "DS001-2099999"}}, nil
		}
		return []openapispec.Operation{{Method: "GET", Path: "/removed", OperationID: "get_removed", LogicalOperationID: "DS001-2019002"}}, nil
	}

	report, err := driftWithDependencies(context.Background(), root, deps)
	if err != nil {
		t.Fatal(err)
	}
	if report.Outcome != "changed" || report.Comparison == nil || !report.Comparison.Truncated || len(report.Comparison.Findings) != MaximumDriftFindings || report.Comparison.TotalChanges != len(details) {
		t.Fatalf("report = %#v", report)
	}
	if report.Comparison.Findings[0].OperationID == "" {
		t.Fatalf("operation-aware findings were not retained first: %#v", report.Comparison.Findings[0])
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	if bytes.Contains(encoded, []byte("private")) || bytes.Contains(encoded, []byte("raw")) || bytes.Contains(encoded, []byte("malicious")) {
		t.Fatalf("report leaked comparison values: %s", encoded)
	}
	if _, err := DecodeDriftReport(bytes.NewReader(encoded)); err != nil {
		t.Fatal(err)
	}
}

func TestDriftClassifiesProcessingFailuresWithoutDiagnostics(t *testing.T) {
	tests := []struct {
		name  string
		alter func(*driftDependencies)
		code  string
		stage string
	}{
		{name: "acquisition", code: "acquisition-failed", stage: "acquisition", alter: func(deps *driftDependencies) {
			deps.acquire = func(context.Context) (DriftAcquisition, error) {
				return DriftAcquisition{RequestBudget: RequestBudget{Ceiling: 3, Used: 1}}, errors.New("unsafe source body")
			}
		}},
		{name: "generation", code: "generation-failed", stage: "generation", alter: func(deps *driftDependencies) {
			deps.generate = func([]Endpoint, GenerateOptions) (GenerationResult, error) {
				return GenerationResult{}, errors.New("unsafe endpoint value")
			}
		}},
		{name: "validation", code: "validation-failed", stage: "validation", alter: func(deps *driftDependencies) {
			deps.validate = func(string, bool) error { return errors.New("unsafe candidate content") }
		}},
		{name: "comparison", code: "comparison-failed", stage: "comparison", alter: func(deps *driftDependencies) {
			deps.compare = func(string, string) (openapispec.Comparison, error) {
				return openapispec.Comparison{}, errors.New("unsafe comparison value")
			}
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			deps := successfulDriftDependencies(t)
			test.alter(&deps)
			report, err := driftWithDependencies(context.Background(), driftRepository(t), deps)
			var driftErr *DriftError
			if !errors.As(err, &driftErr) || report.Outcome != "error" || report.Failure == nil || report.Failure.Code != test.code || report.Failure.Stage != test.stage {
				t.Fatalf("report=%#v error=%v", report, err)
			}
			encoded, marshalErr := json.Marshal(report)
			if marshalErr != nil {
				t.Fatal(marshalErr)
			}
			if bytes.Contains(encoded, []byte("unsafe")) {
				t.Fatalf("report leaked failure diagnostics: %s", encoded)
			}
		})
	}
}

func TestDecodeDriftReportRejectsUntrustedShapes(t *testing.T) {
	report := newDriftReport(func() time.Time { return time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC) })
	report.Outcome = "unchanged"
	report.Comparison = &DriftComparison{}
	encoded, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	unknown := bytes.Replace(encoded, []byte(`"kind"`), []byte(`"unknown":true,"kind"`), 1)
	if _, err := DecodeDriftReport(bytes.NewReader(unknown)); err == nil {
		t.Fatal("unknown report field was accepted")
	}
	if _, err := DecodeDriftReport(strings.NewReader(strings.Repeat("x", MaximumDriftReportBytes+1))); err == nil {
		t.Fatal("oversized report was accepted")
	}
	report.Outcome = "changed"
	if content, err := json.Marshal(report); err != nil {
		t.Fatal(err)
	} else if _, err := DecodeDriftReport(bytes.NewReader(content)); err == nil {
		t.Fatal("conclusion-inconsistent report was accepted")
	}
	report.Comparison = &DriftComparison{TotalChanges: 1, Findings: []DriftFinding{{
		Change: "modified", LogicalOperationID: "DS003-2019017", OperationID: "get_company",
		Method: "GET", Path: "/company.json", Location: "#/paths/~1list.json/get/summary",
	}}}
	if content, err := json.Marshal(report); err != nil {
		t.Fatal(err)
	} else if _, err := DecodeDriftReport(bytes.NewReader(content)); err == nil {
		t.Fatal("finding with mismatched path and location was accepted")
	}
}

func driftRepository(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "openapi"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "openapi", "openapi.yaml"), []byte("openapi: 3.2.0\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root
}

func successfulDriftDependencies(t *testing.T) driftDependencies {
	t.Helper()
	return driftDependencies{
		acquire: func(context.Context) (DriftAcquisition, error) {
			return DriftAcquisition{RequestBudget: RequestBudget{Ceiling: 6, Used: 6}}, nil
		},
		generate: func(_ []Endpoint, options GenerateOptions) (GenerationResult, error) {
			if err := os.WriteFile(filepath.Join(options.OutputDir, "openapi.yaml"), []byte("openapi: 3.2.0\n"), 0o600); err != nil {
				return GenerationResult{}, err
			}
			return GenerationResult{}, nil
		},
		validate: func(string, bool) error { return nil },
		compare:  func(string, string) (openapispec.Comparison, error) { return openapispec.Comparison{}, nil },
		operations: func(string) ([]openapispec.Operation, error) {
			return nil, nil
		},
		now: func() time.Time { return time.Date(2026, 7, 20, 1, 2, 3, 0, time.UTC) },
	}
}
