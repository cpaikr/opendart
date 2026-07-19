package verification

import (
	"errors"
	"path/filepath"
	"strings"

	"github.com/cpaikr/opendart/internal/auditorprobe"
	"github.com/cpaikr/opendart/internal/guide"
	"github.com/cpaikr/opendart/internal/liveconformance"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/releaseguard"
	"github.com/cpaikr/opendart/internal/sdkgen"
	"github.com/cpaikr/opendart/internal/sdkgen/model"
)

const (
	phaseCatalog          = "catalog"
	phaseSourceLint       = "source-lint"
	phaseBundleLint       = "bundle-lint"
	phaseBundleFreshness  = "bundle-freshness"
	phaseRustSDKFreshness = "rust-sdk-freshness"
	phaseLiveConformance  = "live-conformance-preflight"
	phaseAuditorEvidence  = "auditor-evidence"
	phaseReleaseGuard     = "release-guard"
)

var passedPhases = []string{
	phaseCatalog,
	phaseSourceLint,
	phaseBundleFreshness,
	phaseBundleLint,
	phaseRustSDKFreshness,
	phaseLiveConformance,
	phaseAuditorEvidence,
	phaseReleaseGuard,
}

// Report is the bounded, deterministic result of credential-free repository
// verification. Artifact paths and detailed source metadata stay out of it.
type Report struct {
	PassedPhases []string       `json:"passedPhases"`
	Catalog      CatalogSummary `json:"catalog"`
}

// CatalogSummary retains the accepted inventory totals without copying paths,
// source URLs, or generated document bodies into verification output.
type CatalogSummary struct {
	OpenAPI          string `json:"openapi"`
	LogicalEndpoints int    `json:"logicalEndpoints"`
	PhysicalPaths    int    `json:"physicalPaths"`
	RequestArguments int    `json:"requestArguments"`
	ResponseFields   int    `json:"responseFields"`
	MessageCodes     int    `json:"messageCodes"`
}

// Error identifies the failed verification phase, artifact, and rule. The
// underlying error remains available through errors.Is/As but is deliberately
// omitted from Error() so source bodies, URLs, and credentials cannot leak.
type Error struct {
	Phase     string
	Artifact  string
	Rule      string
	Operation string
	Location  string
	cause     error
}

func (e *Error) Error() string {
	parts := []string{
		"verification failed:",
		"phase=" + e.Phase,
		"artifact=" + e.Artifact,
		"rule=" + e.Rule,
	}
	if e.Operation != "" {
		parts = append(parts, "operation="+e.Operation)
	}
	if e.Location != "" {
		parts = append(parts, "location="+e.Location)
	}
	return strings.Join(parts, " ")
}

func (e *Error) Unwrap() error {
	return e.cause
}

type dependencies struct {
	validateCatalog func(guide.CatalogOptions) (guide.CatalogReport, error)
	lint            func(string) ([]openapispec.LintDiagnostic, error)
	checkFresh      func(string, string) error
	checkLive       func(string) error
	checkEvidence   func(string) error
	checkRelease    func(string) error
	checkRustSDK    func(string, string) error
}

// Verify runs the complete repository gate using only committed local files.
// It performs no guide acquisition, API request, or credential lookup.
func Verify(repositoryRoot string) (Report, error) {
	return verifyWith(repositoryRoot, dependencies{
		validateCatalog: guide.ValidateCatalog,
		lint:            openapispec.Lint,
		checkFresh:      openapispec.CheckBundleFresh,
		checkLive: func(root string) error {
			_, err := liveconformance.PreflightRepository(root)
			return err
		},
		checkEvidence: auditorprobe.ValidateEvidenceFile,
		checkRelease:  releaseguard.Check,
		checkRustSDK:  sdkgen.CheckRustFresh,
	})
}

func verifyWith(repositoryRoot string, deps dependencies) (Report, error) {
	if strings.TrimSpace(repositoryRoot) == "" {
		return Report{}, failure("options", "repository", "repository-root", nil)
	}
	absoluteRoot, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return Report{}, failure("options", "repository", "repository-root", err)
	}
	source := filepath.Join(absoluteRoot, "openapi", "openapi.yaml")
	bundle := filepath.Join(absoluteRoot, "openapi", "generated", "openapi.bundle.yaml")
	auditorEvidence := filepath.Join(absoluteRoot, "docs", "api", "evidence", "auditor-2026-07-18.json")
	rustSDKOutput := filepath.Join(absoluteRoot, "sdk", "rust", "crates", "opendart", "src", "generated")

	catalog, err := deps.validateCatalog(guide.CatalogOptions{Root: source})
	if err != nil {
		return Report{}, catalogFailure(source, err)
	}
	if err := lintArtifact(deps, phaseSourceLint, source); err != nil {
		return Report{}, err
	}
	if err := deps.checkFresh(source, bundle); err != nil {
		rule := "bundle-generation"
		switch {
		case errors.Is(err, openapispec.ErrBundleMissing):
			rule = "bundle-missing"
		case errors.Is(err, openapispec.ErrBundleStale):
			rule = "bundle-stale"
		}
		return Report{}, failure(phaseBundleFreshness, bundle, rule, err)
	}
	if err := lintArtifact(deps, phaseBundleLint, bundle); err != nil {
		return Report{}, err
	}
	if err := deps.checkRustSDK(source, rustSDKOutput); err != nil {
		rule := "sdk-generation"
		switch {
		case errors.Is(err, sdkgen.ErrGeneratedMissing):
			rule = "generated-missing"
		case errors.Is(err, sdkgen.ErrGeneratedStale):
			rule = "generated-stale"
		case errors.Is(err, sdkgen.ErrGeneratedUnexpected):
			rule = "generated-unexpected"
		case errors.Is(err, sdkgen.ErrGeneratedUnowned):
			rule = "generated-ownership"
		}
		operation, location := "", ""
		var modelError *model.Error
		if errors.As(err, &modelError) {
			rule, operation, location = modelError.Rule, modelError.Operation, modelError.Location
		}
		var surfaceError *openapispec.SDKSurfaceError
		if errors.As(err, &surfaceError) {
			rule, operation, location = surfaceError.Rule, surfaceError.Operation, surfaceError.Location
		}
		return Report{}, contextualFailure(phaseRustSDKFreshness, rustSDKOutput, rule, operation, location, err)
	}
	if err := deps.checkLive(absoluteRoot); err != nil {
		return Report{}, liveConformanceFailure(err)
	}
	if err := deps.checkEvidence(auditorEvidence); err != nil {
		return Report{}, failure(phaseAuditorEvidence, auditorEvidence, "sanitized-evidence-manifest", err)
	}
	if err := deps.checkRelease(absoluteRoot); err != nil {
		artifact, rule := "release configuration", "release-policy"
		var guardError *releaseguard.Error
		if errors.As(err, &guardError) {
			artifact, rule = guardError.Artifact, guardError.Invariant
		}
		return Report{}, failure(phaseReleaseGuard, artifact, rule, err)
	}

	return Report{
		PassedPhases: append([]string(nil), passedPhases...),
		Catalog: CatalogSummary{
			OpenAPI:          catalog.OpenAPI,
			LogicalEndpoints: catalog.LogicalEndpoints,
			PhysicalPaths:    catalog.PhysicalPaths,
			RequestArguments: catalog.RequestArguments,
			ResponseFields:   catalog.ResponseFields,
			MessageCodes:     catalog.MessageCodes,
		},
	}, nil
}

func liveConformanceFailure(err error) *Error {
	artifact, rule, operation, location := "live conformance inventory", "coverage-budget-sanitization", "", ""
	var verificationError *Error
	if errors.As(err, &verificationError) {
		artifact, rule = verificationError.Artifact, verificationError.Rule
		operation, location = verificationError.Operation, verificationError.Location
	} else {
		var conformanceError *liveconformance.Error
		if errors.As(err, &conformanceError) {
			rule = conformanceError.Failure.Code
			operation = conformanceError.Failure.Operation
			if conformanceError.Failure.CaseID != "" {
				artifact = conformanceError.Failure.CaseID
			} else if conformanceError.Failure.DiscoveryID != "" {
				artifact = string(conformanceError.Failure.DiscoveryID)
			}
		}
	}
	return contextualFailure(phaseLiveConformance, artifact, rule, operation, location, err)
}

func lintArtifact(deps dependencies, phase, artifact string) error {
	diagnostics, err := deps.lint(artifact)
	if err != nil {
		return failure(phase, artifact, "openapi-load-or-validation", err)
	}
	if len(diagnostics) == 0 {
		return nil
	}
	diagnostic := diagnostics[0]
	if diagnostic.Artifact != "" {
		artifact = diagnostic.Artifact
	}
	rule := diagnostic.Rule
	if rule == "" {
		rule = "strict-lint"
	}
	return contextualFailure(phase, artifact, rule, diagnostic.Operation, diagnostic.Location, errors.New("strict OpenAPI lint rejected the artifact"))
}

func catalogFailure(artifact string, err error) error {
	phase, rule := phaseCatalog, "catalog-validation"
	operation, location := "", ""
	var catalogError *guide.CatalogError
	if errors.As(err, &catalogError) {
		if catalogError.Diagnostic.Phase != "" {
			phase += "/" + catalogError.Diagnostic.Phase
		}
		if catalogError.Diagnostic.Rule != "" {
			rule = catalogError.Diagnostic.Rule
		}
		if catalogError.Diagnostic.Artifact != "" {
			artifact = catalogError.Diagnostic.Artifact
		}
		operation = catalogError.Diagnostic.Operation
		location = catalogError.Diagnostic.Location
	}
	return contextualFailure(phase, artifact, rule, operation, location, err)
}

func failure(phase, artifact, rule string, cause error) *Error {
	return contextualFailure(phase, artifact, rule, "", "", cause)
}

func contextualFailure(phase, artifact, rule, operation, location string, cause error) *Error {
	return &Error{
		Phase:     boundedContext(phase, "unknown"),
		Artifact:  boundedContext(artifact, "unknown-artifact"),
		Rule:      boundedContext(rule, "unknown-rule"),
		Operation: boundedOptionalContext(operation),
		Location:  boundedOptionalContext(location),
		cause:     cause,
	}
}

func boundedContext(value, fallback string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "://") || strings.ContainsAny(value, "\r\n") || len(value) > 1024 {
		return fallback
	}
	return value
}

func boundedOptionalContext(value string) string {
	value = strings.TrimSpace(value)
	if value == "" || strings.Contains(value, "://") || strings.ContainsAny(value, "\r\n") || len(value) > 1024 {
		return ""
	}
	return value
}
