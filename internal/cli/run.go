package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"
	"time"

	"github.com/cpaikr/opendart/internal/auditorprobe"
	"github.com/cpaikr/opendart/internal/crateverification"
	"github.com/cpaikr/opendart/internal/driftnotifier"
	guidesync "github.com/cpaikr/opendart/internal/guide"
	"github.com/cpaikr/opendart/internal/liveconformance"
	"github.com/cpaikr/opendart/internal/livenotifier"
	"github.com/cpaikr/opendart/internal/multicompanyprobe"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/sdkgen"
	"github.com/cpaikr/opendart/internal/verification"
)

func Run(args []string, stdout, stderr io.Writer) int {
	return RunContext(context.Background(), args, stdout, stderr)
}

func RunContext(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		if err := usage(stderr); err != nil {
			return 1
		}
		return 2
	}

	switch args[0] {
	case "sync":
		return runSync(ctx, args[1:], stdout, stderr)
	case "catalog":
		return runCatalog(args[1:], stdout, stderr)
	case "lint":
		return runLint(args[1:], stdout, stderr)
	case "bundle":
		return runBundle(args[1:], stdout, stderr)
	case "generate-sdk":
		return runGenerateSDK(args[1:], stdout, stderr)
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "guide-drift":
		return runGuideDrift(ctx, args[1:], stdout, stderr)
	case "guide-drift-notify":
		return runGuideDriftNotify(ctx, args[1:], stdout, stderr)
	case "verify-crate-artifact":
		return runVerifyCrateArtifact(args[1:], stdout, stderr)
	case "live-conformance":
		return runLiveConformance(ctx, args[1:], stdout, stderr)
	case "live-conformance-notify":
		return runLiveConformanceNotify(ctx, args[1:], stdout, stderr)
	case "probe-multi-company":
		return runProbeMultiCompany(ctx, args[1:], stdout, stderr)
	case "probe-auditor-evidence":
		return runProbeAuditorEvidence(ctx, args[1:], stdout, stderr)
	case "help", "-h", "--help":
		if err := usage(stdout); err != nil {
			return 1
		}
		return 0
	default:
		if _, err := fmt.Fprintf(stderr, "unknown command %q\n", args[0]); err != nil {
			return 1
		}
		if err := usage(stderr); err != nil {
			return 1
		}
		return 2
	}
}

type catalogRunner func(guidesync.CatalogOptions) (guidesync.CatalogReport, error)

func runCatalog(args []string, stdout, stderr io.Writer) int {
	return runCatalogWith(args, stdout, stderr, guidesync.ValidateCatalog)
}

func runCatalogWith(args []string, stdout, stderr io.Writer, runner catalogRunner) int {
	flags := flag.NewFlagSet("catalog", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "openapi/openapi.yaml", "root OpenAPI document")
	structuralOnly := flags.Bool("structural-only", false, "allow an intentionally partial endpoint inventory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("catalog", flags, stderr); code != 0 {
		return code
	}
	report, err := runner(guidesync.CatalogOptions{Root: *root, StructuralOnly: *structuralOnly})
	if err != nil {
		var catalogError *guidesync.CatalogError
		if errors.As(err, &catalogError) {
			if encodeErr := writeJSON(stderr, catalogError.Diagnostic); encodeErr != nil {
				return 1
			}
			return 1
		}
		return writeCommandError(stderr, "catalog", err, 1)
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write catalog report", err, 1)
	}
	return 0
}

type lintRunner func(string) ([]openapispec.LintDiagnostic, error)

func runLint(args []string, stdout, stderr io.Writer) int {
	return runLintWith(args, stdout, stderr, openapispec.Lint)
}

func runLintWith(args []string, stdout, stderr io.Writer, runner lintRunner) int {
	flags := flag.NewFlagSet("lint", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "openapi/openapi.yaml", "root OpenAPI document")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("lint", flags, stderr); code != 0 {
		return code
	}
	diagnostics, err := runner(*root)
	if err != nil {
		return writeCommandError(stderr, "lint", err, 1)
	}
	if len(diagnostics) != 0 {
		if err := writeJSON(stderr, diagnostics); err != nil {
			return 1
		}
		return 1
	}
	if err := writeJSON(stdout, struct {
		Root  string `json:"root"`
		Valid bool   `json:"valid"`
	}{Root: *root, Valid: true}); err != nil {
		return writeCommandError(stderr, "write lint report", err, 1)
	}
	return 0
}

type bundleRunner func(string, string) error

func runBundle(args []string, stdout, stderr io.Writer) int {
	return runBundleWith(args, stdout, stderr, openapispec.WriteBundle)
}

type verificationRunner func(string) (verification.Report, error)
type crateVerificationRunner func(crateverification.Options) (crateverification.Report, error)

type sdkGenerationRunner func(string, sdkgen.RustOutputs) (sdkgen.Report, error)

func runGenerateSDK(args []string, stdout, stderr io.Writer) int {
	return runGenerateSDKWith(args, stdout, stderr, sdkgen.GenerateRust)
}

func runGenerateSDKWith(args []string, stdout, stderr io.Writer, runner sdkGenerationRunner) int {
	flags := flag.NewFlagSet("generate-sdk", flag.ContinueOnError)
	flags.SetOutput(stderr)
	language := flags.String("language", "", "SDK language (rust)")
	root := flags.String("root", "openapi/openapi.yaml", "root OpenAPI document")
	output := flags.String("output", "", "owned generated source directory")
	cliOutput := flags.String("cli-output", "", "owned generated CLI source directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("generate-sdk", flags, stderr); code != 0 {
		return code
	}
	if *language != "rust" {
		return writeCommandError(stderr, "generate-sdk", errors.New("--language must be rust"), 2)
	}
	if strings.TrimSpace(*output) == "" {
		return writeCommandError(stderr, "generate-sdk", errors.New("--output is required"), 2)
	}
	if strings.TrimSpace(*cliOutput) == "" {
		return writeCommandError(stderr, "generate-sdk", errors.New("--cli-output is required"), 2)
	}
	report, err := runner(*root, sdkgen.RustOutputs{SDK: *output, CLI: *cliOutput})
	if err != nil {
		return writeCommandError(stderr, "generate-sdk", err, 1)
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write generate-sdk report", err, 1)
	}
	return 0
}

type livePreflightRunner func(string) (liveconformance.PreflightReport, error)
type liveRunner func(context.Context, string) (liveconformance.Report, error)
type liveNotifierRunner func(context.Context, livenotifier.Options) (livenotifier.Result, error)
type guideDriftRunner func(context.Context, string) (guidesync.DriftReport, error)
type driftNotifierRunner func(context.Context, driftnotifier.Options) (driftnotifier.Result, error)

func runGuideDrift(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runGuideDriftWith(ctx, args, stdout, stderr, guidesync.Drift)
}

func runGuideDriftWith(ctx context.Context, args []string, stdout, stderr io.Writer, runner guideDriftRunner) int {
	flags := flag.NewFlagSet("guide-drift", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repositoryRoot := flags.String("repository-root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("guide-drift", flags, stderr); code != 0 {
		return code
	}
	report, err := runner(ctx, *repositoryRoot)
	if err != nil {
		if report.Kind == guidesync.DriftReportKind {
			if encodeErr := writeJSON(stdout, report); encodeErr != nil {
				return 1
			}
		}
		return writeCommandError(stderr, "guide-drift", errors.New("processing failed"), 1)
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write guide drift report", err, 1)
	}
	return 0
}

func runGuideDriftNotify(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runGuideDriftNotifyWith(ctx, args, stdout, stderr, os.Getenv, driftnotifier.Notify)
}

func runGuideDriftNotifyWith(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string, runner driftNotifierRunner) int {
	flags := flag.NewFlagSet("guide-drift-notify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	reportPath := flags.String("report", driftnotifier.DefaultReportPath, "downloaded guide drift report")
	repository := flags.String("repository", "", "GitHub owner/repository")
	producerConclusion := flags.String("producer-conclusion", "", "producer workflow conclusion")
	artifactOutcome := flags.String("artifact-outcome", "", "artifact download step outcome")
	runID := flags.Uint64("run-id", 0, "producer workflow run ID")
	runAttempt := flags.Uint64("run-attempt", 0, "producer workflow run attempt")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("guide-drift-notify", flags, stderr); code != 0 {
		return code
	}
	result, err := runner(ctx, driftnotifier.Options{
		ReportPath:         *reportPath,
		Repository:         *repository,
		ProducerConclusion: *producerConclusion,
		ArtifactOutcome:    *artifactOutcome,
		RunID:              *runID,
		RunAttempt:         *runAttempt,
		Token:              getenv("GITHUB_TOKEN"),
	})
	if err != nil {
		return writeCommandError(stderr, "guide-drift-notify", errors.New("notification failed"), 1)
	}
	if err := writeJSON(stdout, result); err != nil {
		return writeCommandError(stderr, "write guide drift notification result", err, 1)
	}
	return 0
}

func runLiveConformance(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runLiveConformanceWith(ctx, args, stdout, stderr, liveconformance.PreflightRepository, liveconformance.RunRepository)
}

func runLiveConformanceWith(ctx context.Context, args []string, stdout, stderr io.Writer, preflight livePreflightRunner, runner liveRunner) int {
	flags := flag.NewFlagSet("live-conformance", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repositoryRoot := flags.String("repository-root", ".", "repository root")
	preflightOnly := flags.Bool("preflight-only", false, "run credential-free coverage, budget, and sanitization gates")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("live-conformance", flags, stderr); code != 0 {
		return code
	}
	if *preflightOnly {
		report, err := preflight(*repositoryRoot)
		if err != nil {
			return writeCommandError(stderr, "live-conformance", errors.New("preflight failed"), 1)
		}
		if err := writeJSON(stdout, report); err != nil {
			return writeCommandError(stderr, "write live conformance preflight report", err, 1)
		}
		return 0
	}
	report, err := runner(ctx, *repositoryRoot)
	if err != nil {
		if report.Kind == liveconformance.ReportKind {
			if encodeErr := writeJSON(stdout, report); encodeErr != nil {
				return 1
			}
		}
		return writeCommandError(stderr, "live-conformance", errors.New("execution failed"), 1)
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write live conformance report", err, 1)
	}
	return 0
}

func runLiveConformanceNotify(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	return runLiveConformanceNotifyWith(ctx, args, stdout, stderr, os.Getenv, livenotifier.Notify)
}

func runLiveConformanceNotifyWith(ctx context.Context, args []string, stdout, stderr io.Writer, getenv func(string) string, runner liveNotifierRunner) int {
	flags := flag.NewFlagSet("live-conformance-notify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	reportPath := flags.String("report", livenotifier.DefaultReportPath, "downloaded live conformance report")
	repository := flags.String("repository", "", "GitHub owner/repository")
	producerConclusion := flags.String("producer-conclusion", "", "producer workflow conclusion")
	artifactOutcome := flags.String("artifact-outcome", "", "artifact download step outcome")
	runID := flags.Uint64("run-id", 0, "producer workflow run ID")
	runAttempt := flags.Uint64("run-attempt", 0, "producer workflow run attempt")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("live-conformance-notify", flags, stderr); code != 0 {
		return code
	}
	result, err := runner(ctx, livenotifier.Options{
		ReportPath:         *reportPath,
		Repository:         *repository,
		ProducerConclusion: *producerConclusion,
		ArtifactOutcome:    *artifactOutcome,
		RunID:              *runID,
		RunAttempt:         *runAttempt,
		Token:              getenv("GITHUB_TOKEN"),
	})
	if err != nil {
		return writeCommandError(stderr, "live-conformance-notify", errors.New("notification failed"), 1)
	}
	if err := writeJSON(stdout, result); err != nil {
		return writeCommandError(stderr, "write live conformance notification result", err, 1)
	}
	return 0
}

func runVerify(args []string, stdout, stderr io.Writer) int {
	return runVerifyWith(args, stdout, stderr, verification.Verify)
}

func runVerifyCrateArtifact(args []string, stdout, stderr io.Writer) int {
	return runVerifyCrateArtifactWith(args, stdout, stderr, crateverification.Verify)
}

func runVerifyCrateArtifactWith(args []string, stdout, stderr io.Writer, runner crateVerificationRunner) int {
	flags := flag.NewFlagSet("verify-crate-artifact", flag.ContinueOnError)
	flags.SetOutput(stderr)
	candidate := flags.String("candidate", "", "local reviewed .crate candidate")
	accepted := flags.String("accepted", "", "local accepted .crate artifact")
	inventory := flags.String("inventory", "", "reviewed newline-delimited package inventory")
	packageName := flags.String("package", "", "expected Cargo package name")
	version := flags.String("version", "", "expected Cargo package version")
	revision := flags.String("revision", "", "expected full Git revision")
	vcsPath := flags.String("vcs-path", "", "expected package path in Cargo VCS metadata")
	registryChecksum := flags.String("registry-checksum", "", "accepted registry SHA-256")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("verify-crate-artifact", flags, stderr); code != 0 {
		return code
	}
	options := crateverification.Options{
		CandidatePath: *candidate, AcceptedPath: *accepted, InventoryPath: *inventory,
		Package: *packageName, Version: *version, Revision: *revision,
		VCSPath: *vcsPath, RegistryChecksum: *registryChecksum,
	}
	for _, required := range []struct {
		name  string
		value string
	}{
		{"candidate", options.CandidatePath}, {"accepted", options.AcceptedPath},
		{"inventory", options.InventoryPath}, {"package", options.Package},
		{"version", options.Version}, {"revision", options.Revision},
		{"vcs-path", options.VCSPath}, {"registry-checksum", options.RegistryChecksum},
	} {
		if strings.TrimSpace(required.value) == "" {
			return writeCommandError(stderr, "verify-crate-artifact", fmt.Errorf("--%s is required", required.name), 2)
		}
	}
	report, err := runner(options)
	if err != nil {
		var verificationError *crateverification.Error
		if errors.As(err, &verificationError) {
			diagnostic := struct {
				Artifact  string `json:"artifact"`
				Invariant string `json:"invariant"`
			}{Artifact: verificationError.Artifact, Invariant: verificationError.Invariant}
			if encodeErr := writeJSON(stderr, diagnostic); encodeErr != nil {
				return 1
			}
			return 1
		}
		return writeCommandError(stderr, "verify-crate-artifact", errors.New("unexpected artifact verification failure"), 1)
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write crate artifact verification report", err, 1)
	}
	return 0
}

func runVerifyWith(args []string, stdout, stderr io.Writer, runner verificationRunner) int {
	flags := flag.NewFlagSet("verify", flag.ContinueOnError)
	flags.SetOutput(stderr)
	repositoryRoot := flags.String("repository-root", ".", "repository root")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("verify", flags, stderr); code != 0 {
		return code
	}
	report, err := runner(*repositoryRoot)
	if err != nil {
		var verificationError *verification.Error
		if errors.As(err, &verificationError) {
			diagnostic := struct {
				Phase     string `json:"phase"`
				Artifact  string `json:"artifact"`
				Rule      string `json:"rule"`
				Operation string `json:"operation,omitempty"`
				Location  string `json:"location,omitempty"`
			}{
				Phase:     verificationError.Phase,
				Artifact:  verificationError.Artifact,
				Rule:      verificationError.Rule,
				Operation: verificationError.Operation,
				Location:  verificationError.Location,
			}
			if encodeErr := writeJSON(stderr, diagnostic); encodeErr != nil {
				return 1
			}
			return 1
		}
		return writeCommandError(stderr, "verify", err, 1)
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write verification report", err, 1)
	}
	return 0
}

func runBundleWith(args []string, _ io.Writer, stderr io.Writer, runner bundleRunner) int {
	flags := flag.NewFlagSet("bundle", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "openapi/openapi.yaml", "root OpenAPI document")
	output := flags.String("output", "", "portable bundle output file")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("bundle", flags, stderr); code != 0 {
		return code
	}
	if strings.TrimSpace(*output) == "" {
		return writeCommandError(stderr, "bundle", errors.New("--output is required"), 2)
	}
	if err := runner(*root, *output); err != nil {
		return writeCommandError(stderr, "bundle", err, 1)
	}
	return 0
}

func rejectPositionalArguments(command string, flags *flag.FlagSet, stderr io.Writer) int {
	if flags.NArg() == 0 {
		return 0
	}
	if _, err := fmt.Fprintf(stderr, "%s does not accept positional arguments\n", command); err != nil {
		return 1
	}
	return 2
}

func writeJSON(output io.Writer, value any) error {
	encoder := json.NewEncoder(output)
	encoder.SetIndent("", "  ")
	return encoder.Encode(value)
}

func runSync(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return writeCommandError(stderr, "sync", fmt.Errorf("read working directory: %w", err), 1)
	}
	repositoryRoot, err := findRepositoryRoot(workingDirectory)
	if err != nil {
		return writeCommandError(stderr, "sync", err, 1)
	}
	return runSyncWith(ctx, args, stdout, stderr, repositoryRoot, time.Now(), guidesync.Sync)
}

type syncRunner func(context.Context, guidesync.SyncOptions) (guidesync.SyncReport, error)

func runSyncWith(ctx context.Context, args []string, stdout, stderr io.Writer, repositoryRoot string, now time.Time, runner syncRunner) int {
	options, err := parseSyncCLIOptions(args, repositoryRoot, now, stderr)
	if err != nil {
		return writeCommandError(stderr, "sync", err, 2)
	}
	report, err := runner(ctx, guidesync.SyncOptions{
		RepositoryRoot: options.RepositoryRoot,
		Output:         options.Output,
		CheckedAt:      options.CheckedAt,
		Only:           options.Only,
		ParityBaseline: options.ParityBaseline,
	})
	if err != nil {
		var sourceError *guidesync.SourceError
		if errors.As(err, &sourceError) {
			diagnostic := map[string]any{"message": err.Error()}
			for current := err; current != nil; current = errors.Unwrap(current) {
				if source, ok := current.(*guidesync.SourceError); ok {
					for key, value := range source.Context {
						diagnostic[key] = value
					}
				}
			}
			encoder := json.NewEncoder(stderr)
			encoder.SetIndent("", "  ")
			if writeErr := encoder.Encode(diagnostic); writeErr != nil {
				return 1
			}
			return 1
		}
		return writeCommandError(stderr, "sync", err, 1)
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		return writeCommandError(stderr, "write sync report", err, 1)
	}
	return 0
}

func writeCommandError(output io.Writer, command string, err error, code int) int {
	if _, writeErr := fmt.Fprintf(output, "%s: %v\n", command, err); writeErr != nil {
		return 1
	}
	return code
}

type probeMultiCompanyRunner func(context.Context, string) (multicompanyprobe.Report, error)

func runProbeMultiCompany(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return writeCommandError(stderr, "probe-multi-company", fmt.Errorf("read working directory: %w", err), 1)
	}
	repositoryRoot, err := findRepositoryRoot(workingDirectory)
	if err != nil {
		return writeCommandError(stderr, "probe-multi-company", err, 1)
	}
	return runProbeMultiCompanyWith(ctx, args, stdout, stderr, repositoryRoot, multicompanyprobe.Run)
}

func runProbeMultiCompanyWith(ctx context.Context, args []string, stdout, stderr io.Writer, repositoryRoot string, runner probeMultiCompanyRunner) int {
	flags := flag.NewFlagSet("probe-multi-company", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("probe-multi-company", flags, stderr); code != 0 {
		return code
	}
	report, err := runner(ctx, repositoryRoot)
	if err != nil {
		var probeError *multicompanyprobe.Error
		if errors.As(err, &probeError) {
			diagnostic := struct {
				Error   string         `json:"error"`
				Message string         `json:"message"`
				Context map[string]any `json:"context"`
			}{Error: "ProbeError", Message: probeError.Message, Context: probeError.Context}
			if encodeErr := writeJSON(stderr, diagnostic); encodeErr != nil {
				return 1
			}
			return 1
		}
		diagnostic := struct {
			Error   string         `json:"error"`
			Message string         `json:"message"`
			Context map[string]any `json:"context"`
		}{Error: "ProbeError", Message: "Unexpected serialization probe failure", Context: map[string]any{}}
		if encodeErr := writeJSON(stderr, diagnostic); encodeErr != nil {
			return 1
		}
		return 1
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write probe-multi-company report", err, 1)
	}
	return 0
}

type probeAuditorEvidenceRunner func(context.Context, string) (auditorprobe.Report, error)

func runProbeAuditorEvidence(ctx context.Context, args []string, stdout, stderr io.Writer) int {
	workingDirectory, err := os.Getwd()
	if err != nil {
		return writeCommandError(stderr, "probe-auditor-evidence", fmt.Errorf("read working directory: %w", err), 1)
	}
	repositoryRoot, err := findRepositoryRoot(workingDirectory)
	if err != nil {
		return writeCommandError(stderr, "probe-auditor-evidence", err, 1)
	}
	return runProbeAuditorEvidenceWith(ctx, args, stdout, stderr, repositoryRoot, auditorprobe.Run)
}

func runProbeAuditorEvidenceWith(ctx context.Context, args []string, stdout, stderr io.Writer, repositoryRoot string, runner probeAuditorEvidenceRunner) int {
	flags := flag.NewFlagSet("probe-auditor-evidence", flag.ContinueOnError)
	flags.SetOutput(stderr)
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if code := rejectPositionalArguments("probe-auditor-evidence", flags, stderr); code != 0 {
		return code
	}
	report, err := runner(ctx, repositoryRoot)
	if err != nil {
		var probeError *auditorprobe.Error
		if errors.As(err, &probeError) {
			diagnostic := struct {
				Error   string                          `json:"error"`
				Message string                          `json:"message"`
				Request *auditorprobe.RequestCoordinate `json:"request,omitempty"`
			}{Error: "ProbeError", Message: probeError.Message, Request: probeError.Request}
			if encodeErr := writeJSON(stderr, diagnostic); encodeErr != nil {
				return 1
			}
			return 1
		}
		diagnostic := struct {
			Error   string `json:"error"`
			Message string `json:"message"`
		}{Error: "ProbeError", Message: "Unexpected auditor evidence probe failure"}
		if encodeErr := writeJSON(stderr, diagnostic); encodeErr != nil {
			return 1
		}
		return 1
	}
	if err := writeJSON(stdout, report); err != nil {
		return writeCommandError(stderr, "write probe-auditor-evidence report", err, 1)
	}
	return 0
}

func usage(output io.Writer) error {
	for _, line := range []string{
		"usage: opendart-tool <command> [options]",
		"commands:",
		"  sync           synchronize the official guide into OpenAPI source",
		"  catalog        validate generated catalog and reference invariants",
		"  lint           apply strict OpenAPI policy",
		"  bundle         write the portable OpenAPI bundle",
		"  generate-sdk   generate the owned Rust SDK and CLI source trees",
		"  verify         run credential-free repository verification",
		"  guide-drift    compare the current public guide with the committed contract",
		"  guide-drift-notify  update the isolated public guide drift issue",
		"  verify-crate-artifact  compare local candidate and accepted Rust crate artifacts",
		"  live-conformance  run the reviewed live matrix (use --preflight-only offline)",
		"  live-conformance-notify  update the isolated live failure issue",
		"  probe-multi-company  run the focused credentialed serialization probe",
		"  probe-auditor-evidence  emit the focused sanitized auditor evidence manifest",
	} {
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	return nil
}
