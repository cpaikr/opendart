package guide

import (
	"context"
	"errors"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

type driftDependencies struct {
	acquire    func(context.Context) (DriftAcquisition, error)
	generate   func([]Endpoint, GenerateOptions) (GenerationResult, error)
	validate   func(string, bool) error
	compare    func(string, string) (openapispec.Comparison, error)
	operations func(string) ([]openapispec.Operation, error)
	now        func() time.Time
}

// Drift generates a temporary current-guide candidate and compares it to the
// committed baseline without publishing or modifying repository artifacts.
func Drift(ctx context.Context, repositoryRoot string) (DriftReport, error) {
	return driftWithDependencies(ctx, repositoryRoot, driftDependencies{
		acquire:  AcquireDrift,
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
		now: time.Now,
	})
}

func driftWithDependencies(ctx context.Context, repositoryRoot string, deps driftDependencies) (DriftReport, error) {
	report := newDriftReport(deps.now)
	if strings.TrimSpace(repositoryRoot) == "" || deps.acquire == nil || deps.generate == nil || deps.validate == nil || deps.compare == nil || deps.operations == nil {
		return failDriftReport(report, DriftFailure{Code: "validation-failed", Stage: "validation"}, errors.New("drift configuration is incomplete"))
	}
	baseline := filepath.Join(repositoryRoot, "openapi", "openapi.yaml")
	if info, err := os.Stat(baseline); err != nil || !info.Mode().IsRegular() {
		return failDriftReport(report, DriftFailure{Code: "validation-failed", Stage: "validation"}, errors.New("committed baseline is unavailable"))
	}

	acquisition, err := deps.acquire(ctx)
	report.RequestBudget = acquisition.RequestBudget
	if err != nil {
		return failDriftReport(report, DriftFailure{Code: "acquisition-failed", Stage: "acquisition"}, err)
	}
	if err := ctx.Err(); err != nil {
		return failDriftReport(report, DriftFailure{Code: "acquisition-failed", Stage: "acquisition"}, err)
	}
	staging, err := os.MkdirTemp("", "opendart-guide-drift-")
	if err != nil {
		return failDriftReport(report, DriftFailure{Code: "generation-failed", Stage: "generation"}, err)
	}
	defer func() { _ = os.RemoveAll(staging) }()
	checkedAt := report.ObservedAt[:10]
	if _, err := deps.generate(acquisition.Endpoints, GenerateOptions{OutputDir: staging, CheckedAt: checkedAt}); err != nil {
		return failDriftReport(report, DriftFailure{Code: "generation-failed", Stage: "generation"}, err)
	}
	if err := ctx.Err(); err != nil {
		return failDriftReport(report, DriftFailure{Code: "generation-failed", Stage: "generation"}, err)
	}
	if err := deps.validate(staging, false); err != nil {
		return failDriftReport(report, DriftFailure{Code: "validation-failed", Stage: "validation"}, err)
	}
	if err := ctx.Err(); err != nil {
		return failDriftReport(report, DriftFailure{Code: "validation-failed", Stage: "validation"}, err)
	}
	comparison, err := deps.compare(baseline, filepath.Join(staging, "openapi.yaml"))
	if err != nil {
		return failDriftReport(report, DriftFailure{Code: "comparison-failed", Stage: "comparison"}, err)
	}
	if err := ctx.Err(); err != nil {
		return failDriftReport(report, DriftFailure{Code: "comparison-failed", Stage: "comparison"}, err)
	}
	baselineOperations, err := deps.operations(baseline)
	if err != nil {
		return failDriftReport(report, DriftFailure{Code: "comparison-failed", Stage: "comparison"}, err)
	}
	candidateOperations, err := deps.operations(filepath.Join(staging, "openapi.yaml"))
	if err != nil {
		return failDriftReport(report, DriftFailure{Code: "comparison-failed", Stage: "comparison"}, err)
	}
	report.Comparison = buildDriftComparison(comparison, baselineOperations, candidateOperations)
	if comparison.TotalChanges == 0 {
		report.Outcome = "unchanged"
	} else {
		report.Outcome = "changed"
	}
	if err := validateDriftReport(report); err != nil {
		return failDriftReport(newDriftReport(deps.now), DriftFailure{Code: "report-failed", Stage: "report"}, err)
	}
	return report, nil
}

func newDriftReport(now func() time.Time) DriftReport {
	observed := time.Now().UTC()
	if now != nil {
		observed = now().UTC()
	}
	return DriftReport{
		SchemaVersion: DriftReportSchemaVersion,
		Kind:          DriftReportKind,
		ObservedAt:    observed.Format("2006-01-02T15:04:05.000Z"),
	}
}

func failDriftReport(report DriftReport, failure DriftFailure, cause error) (DriftReport, error) {
	report.Outcome = "error"
	report.Comparison = nil
	report.Failure = &failure
	if err := validateDriftReport(report); err != nil {
		report = DriftReport{
			SchemaVersion: DriftReportSchemaVersion,
			Kind:          DriftReportKind,
			Outcome:       "error",
			ObservedAt:    report.ObservedAt,
			Failure:       &DriftFailure{Code: "report-failed", Stage: "report"},
		}
		failure = *report.Failure
	}
	return report, &DriftError{Failure: failure, cause: cause}
}

func buildDriftComparison(comparison openapispec.Comparison, baseline, candidate []openapispec.Operation) *DriftComparison {
	baselineByPath := operationsByPath(baseline)
	candidateByPath := operationsByPath(candidate)
	coordinated := make([]DriftFinding, 0, MaximumDriftFindings)
	general := make([]DriftFinding, 0, MaximumDriftFindings)
	for _, detail := range comparison.Details {
		finding := DriftFinding{Change: driftChange(detail), Location: detail.Location}
		if path, ok := driftLocationPath(detail.Location); ok {
			operation, exists := candidateByPath[path]
			if finding.Change == "removed" || !exists {
				operation, exists = baselineByPath[path]
			}
			if exists {
				finding.LogicalOperationID = operation.LogicalOperationID
				finding.OperationID = operation.OperationID
				finding.Method = operation.Method
				finding.Path = operation.Path
			}
		}
		if !validDriftFinding(finding) {
			continue
		}
		if finding.OperationID != "" {
			if len(coordinated) < MaximumDriftFindings {
				coordinated = append(coordinated, finding)
			}
		} else if len(general) < MaximumDriftFindings {
			general = append(general, finding)
		}
	}
	findings := append([]DriftFinding(nil), coordinated...)
	findings = append(findings, general[:min(len(general), MaximumDriftFindings-len(findings))]...)
	return &DriftComparison{
		TotalChanges: comparison.TotalChanges, BreakingChanges: comparison.BreakingChanges,
		Findings: findings, Truncated: len(findings) < comparison.TotalChanges,
	}
}

func operationsByPath(operations []openapispec.Operation) map[string]openapispec.Operation {
	result := make(map[string]openapispec.Operation, len(operations))
	for _, operation := range operations {
		if operation.Method == http.MethodGet {
			result[operation.Path] = operation
		}
	}
	return result
}

func driftChange(detail openapispec.ChangeDetail) string {
	switch {
	case detail.Original == "<missing>":
		return "added"
	case detail.New == "<missing>":
		return "removed"
	default:
		return "modified"
	}
}

func driftLocationPath(location string) (string, bool) {
	const prefix = "#/paths/"
	if !strings.HasPrefix(location, prefix) {
		return "", false
	}
	segment := strings.TrimPrefix(location, prefix)
	if index := strings.IndexByte(segment, '/'); index >= 0 {
		segment = segment[:index]
	}
	segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
	if !driftPath.MatchString(segment) {
		return "", false
	}
	return segment, true
}
