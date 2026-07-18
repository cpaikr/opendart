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
		usage(stderr)
		return 2
	}

	switch args[0] {
	case "compatibility":
		return runCompatibility(args[1:], stdout, stderr)
	case "help", "-h", "--help":
		usage(stdout)
		return 0
	default:
		fmt.Fprintf(stderr, "unknown command %q\n", args[0])
		usage(stderr)
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
		fmt.Fprintln(stderr, "compatibility does not accept positional arguments")
		return 2
	}

	report, err := openapispec.RunCompatibilityGate(*root, *baseline)
	if err != nil {
		fmt.Fprintf(stderr, "compatibility: %v\n", err)
		return 1
	}
	encoder := json.NewEncoder(stdout)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(report); err != nil {
		fmt.Fprintf(stderr, "write compatibility report: %v\n", err)
		return 1
	}
	return 0
}

func usage(output io.Writer) {
	fmt.Fprintln(output, "usage: opendart-tool <command> [options]")
	fmt.Fprintln(output, "commands:")
	fmt.Fprintln(output, "  compatibility  run the temporary OpenAPI migration gate")
}
