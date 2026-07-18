package guide

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

func validateStaging(staging string, complete bool) error {
	root := filepath.Join(staging, "openapi.yaml")
	if _, err := ValidateCatalog(CatalogOptions{Root: root, StructuralOnly: !complete}); err != nil {
		return boundedStagingError("catalog", root, err)
	}
	diagnostics, err := openapispec.Lint(root)
	if err != nil {
		return boundedStagingError("OpenAPI", root, err)
	}
	if len(diagnostics) > 0 {
		first := diagnostics[0]
		return &stagingValidationError{
			phase: "OpenAPI", artifact: root, rule: first.Rule,
			operation: first.Operation, location: first.Location,
			message: fmt.Sprintf("strict lint found %d issue(s): %s", len(diagnostics), first.Message),
		}
	}
	return nil
}

const maxStagingDiagnosticRunes = 512

type stagingValidationError struct {
	phase     string
	artifact  string
	rule      string
	operation string
	location  string
	message   string
	cause     error
}

func (e *stagingValidationError) Error() string {
	parts := []string{"validate staged " + boundedStagingValue(e.phase) + " artifact " + boundedStagingValue(e.artifact)}
	if e.rule != "" {
		parts = append(parts, "rule="+boundedStagingValue(e.rule))
	}
	if e.operation != "" {
		parts = append(parts, "operation="+boundedStagingValue(e.operation))
	}
	if e.location != "" {
		parts = append(parts, "location="+boundedStagingValue(e.location))
	}
	return strings.Join(parts, ": ") + ": " + boundedStagingValue(e.message)
}

func (e *stagingValidationError) Unwrap() error { return e.cause }

func boundedStagingError(phase, artifact string, err error) error {
	validationErr := &stagingValidationError{phase: phase, artifact: artifact, message: err.Error(), cause: err}
	var catalogErr *CatalogError
	if errors.As(err, &catalogErr) {
		if catalogErr.Diagnostic.Phase != "" {
			validationErr.phase = phase + "/" + catalogErr.Diagnostic.Phase
		}
		if catalogErr.Diagnostic.Artifact != "" {
			validationErr.artifact = catalogErr.Diagnostic.Artifact
		}
		validationErr.rule = catalogErr.Diagnostic.Rule
		validationErr.operation = catalogErr.Diagnostic.Operation
		validationErr.location = catalogErr.Diagnostic.Location
		validationErr.message = catalogErr.Diagnostic.Message
	}
	return validationErr
}

func boundedStagingValue(value string) string {
	value = strings.TrimSpace(value)
	runes := []rune(value)
	if len(runes) <= maxStagingDiagnosticRunes {
		return value
	}
	return string(runes[:maxStagingDiagnosticRunes]) + "…"
}

func compareStaged(staging, baseline string) error {
	if err := validateOutputMarker(staging); err != nil {
		return fmt.Errorf("staged ownership marker is invalid: %w", err)
	}
	comparison, err := openapispec.Compare(filepath.Join(staging, "openapi.yaml"), filepath.Join(baseline, "openapi.yaml"))
	if err != nil {
		return fmt.Errorf("compare staged OpenAPI with accepted artifact: %w", err)
	}
	if comparison.TotalChanges != 0 {
		return fmt.Errorf("staged OpenAPI differs semantically from accepted artifact at %s", firstChangeLocation(comparison))
	}
	return nil
}

func firstChangeLocation(comparison openapispec.Comparison) string {
	if len(comparison.Details) == 0 || comparison.Details[0].Location == "" {
		return "unknown location"
	}
	return comparison.Details[0].Location
}
