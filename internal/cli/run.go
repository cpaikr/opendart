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
	guidesync "github.com/cpaikr/opendart/internal/guide"
	"github.com/cpaikr/opendart/internal/liveconformance"
	"github.com/cpaikr/opendart/internal/multicompanyprobe"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
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
	case "verify":
		return runVerify(args[1:], stdout, stderr)
	case "live-conformance":
		return runLiveConformance(ctx, args[1:], stdout, stderr)
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

type livePreflightRunner func(string) (liveconformance.PreflightReport, error)
type liveRunner func(context.Context, string) (liveconformance.Report, error)

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

func runVerify(args []string, stdout, stderr io.Writer) int {
	return runVerifyWith(args, stdout, stderr, verification.Verify)
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
		"  verify         run credential-free repository verification",
		"  live-conformance  run the reviewed live matrix (use --preflight-only offline)",
		"  probe-multi-company  run the focused credentialed serialization probe",
		"  probe-auditor-evidence  emit the focused sanitized auditor evidence manifest",
	} {
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	return nil
}
