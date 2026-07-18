package cli

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

func Run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		if err := usage(stderr); err != nil {
			return 1
		}
		return 2
	}

	switch args[0] {
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
		"  compatibility  run the temporary OpenAPI migration gate",
	} {
		if _, err := fmt.Fprintln(output, line); err != nil {
			return err
		}
	}
	return nil
}
