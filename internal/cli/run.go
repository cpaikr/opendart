package cli

import (
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"time"

	guidesync "github.com/cpaikr/opendart/internal/guide"
	openapispec "github.com/cpaikr/opendart/internal/openapi"
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
	case "compatibility":
		return runCompatibility(args[1:], stdout, stderr)
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

func runCompatibility(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("compatibility", flag.ContinueOnError)
	flags.SetOutput(stderr)
	root := flags.String("root", "openapi/openapi.yaml", "root OpenAPI document")
	baseline := flags.String("baseline", "", "accepted bundle to compare semantically")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		if _, err := fmt.Fprintln(stderr, "compatibility does not accept positional arguments"); err != nil {
			return 1
		}
		return 2
	}

	report, err := openapispec.RunCompatibilityGate(*root, *baseline)
	if err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "compatibility: %v\n", err); writeErr != nil {
			return 1
		}
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		if _, writeErr := fmt.Fprintf(stderr, "write compatibility report: %v\n", err); writeErr != nil {
			return 1
		}
		return 1
	}
	return 0
}

func usage(output io.Writer) error {
	for _, line := range []string{
		"usage: opendart-tool <command> [options]",
		"commands:",
		"  sync           synchronize the official guide into OpenAPI source",
		"  compatibility  run the temporary OpenAPI migration gate",
	} {
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	return nil
}
