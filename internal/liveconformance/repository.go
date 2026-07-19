package liveconformance

import (
	"bytes"
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"time"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

// PreflightReport is safe evidence that the repository inventory passed every
// credential-free coverage, request, budget, and report-identity gate.
type PreflightReport struct {
	Valid          bool `json:"valid"`
	PrimaryCases   int  `json:"primaryCases"`
	RequestCeiling int  `json:"requestCeiling"`
}

// PreflightRepository validates the committed matrix without reading the
// credential or constructing an HTTP client.
func PreflightRepository(repositoryRoot string) (PreflightReport, error) {
	document, plan, err := repositoryPlan(repositoryRoot)
	if document != nil {
		defer document.Close()
	}
	if err != nil {
		return PreflightReport{}, err
	}
	if err := validateReportContract(plan); err != nil {
		return PreflightReport{}, preflightError("report-sanitization", "")
	}
	return PreflightReport{Valid: true, PrimaryCases: len(plan.cases), RequestCeiling: plan.requestBudget}, nil
}

func validateReportContract(plan *Plan) error {
	report := newReport(time.Unix(0, 0), plan.requestBudget, plan.discoveryBudget)
	report.Failure = &Failure{Code: "credential-unavailable", Stage: "credential"}
	if err := validateReport(report, ""); err != nil {
		return err
	}
	encoded, err := json.Marshal(report)
	if err != nil {
		return err
	}
	_, err = DecodeReport(bytes.NewReader(encoded))
	return err
}

// RunRepository preflights and executes the committed matrix while retaining
// the OpenAPI document for the complete run.
func RunRepository(ctx context.Context, repositoryRoot string) (Report, error) {
	document, plan, err := repositoryPlan(repositoryRoot)
	if document != nil {
		defer document.Close()
	}
	if err != nil {
		return Report{}, err
	}
	return plan.Run(ctx)
}

func repositoryPlan(repositoryRoot string) (*openapispec.Document, *Plan, error) {
	if strings.TrimSpace(repositoryRoot) == "" {
		return nil, nil, preflightError("repository-root", "")
	}
	document, err := openapispec.Load(filepath.Join(repositoryRoot, "openapi", "openapi.yaml"))
	if err != nil {
		return nil, nil, preflightError("openapi-load", "")
	}
	plan, err := Preflight(document, PrimaryCases(), PrimaryAssertions(), PrimaryDiscoveries()...)
	if err != nil {
		document.Close()
		return nil, nil, err
	}
	for _, primary := range plan.cases {
		if primary.definition.Assertion != AssertionID(primary.operation.LogicalOperationID) {
			document.Close()
			return nil, nil, preflightError("invalid-primary-assertion", primary.operation.Identity())
		}
	}
	return document, plan, nil
}
