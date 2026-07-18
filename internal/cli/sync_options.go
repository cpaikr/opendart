package cli

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	guidesync "github.com/cpaikr/opendart/internal/guide"
)

type syncCLIOptions struct {
	RepositoryRoot string
	Output         string
	CheckedAt      string
	Only           []string
	ParityBaseline string
}

type repeatedStrings []string

func (values *repeatedStrings) String() string {
	return strings.Join(*values, ",")
}

func (values *repeatedStrings) Set(value string) error {
	*values = append(*values, value)
	return nil
}

func parseSyncCLIOptions(args []string, repositoryRoot string, now time.Time, stderr io.Writer) (syncCLIOptions, error) {
	checkedAt, err := checkedAtInSeoul(now)
	if err != nil {
		return syncCLIOptions{}, err
	}
	canonicalOutput := filepath.Join(repositoryRoot, "openapi")
	flags := flag.NewFlagSet("sync", flag.ContinueOnError)
	flags.SetOutput(stderr)
	output := flags.String("output", canonicalOutput, "generated OpenAPI output directory")
	checkedAtFlag := flags.String("checked-at", checkedAt, "official guide check date in YYYY-MM-DD")
	parityBaseline := flags.String("parity-baseline", "", "accepted OpenAPI directory for one-time semantic parity proof")
	var only repeatedStrings
	flags.Var(&only, "only", "logical operation identity to refresh; repeatable")
	if err := flags.Parse(args); err != nil {
		return syncCLIOptions{}, err
	}
	if flags.NArg() != 0 {
		return syncCLIOptions{}, errors.New("sync does not accept positional arguments")
	}
	if !validCalendarDate(*checkedAtFlag) {
		return syncCLIOptions{}, errors.New("--checked-at must use YYYY-MM-DD")
	}
	absoluteOutput, err := filepath.Abs(*output)
	if err != nil {
		return syncCLIOptions{}, fmt.Errorf("resolve --output: %w", err)
	}
	if err := guidesync.ValidateSyncTarget(repositoryRoot, absoluteOutput, len(only) > 0); err != nil {
		return syncCLIOptions{}, err
	}
	return syncCLIOptions{
		RepositoryRoot: repositoryRoot,
		Output:         absoluteOutput,
		CheckedAt:      *checkedAtFlag,
		Only:           deduplicateStrings(only),
		ParityBaseline: *parityBaseline,
	}, nil
}

func checkedAtInSeoul(now time.Time) (string, error) {
	location, err := time.LoadLocation("Asia/Seoul")
	if err != nil {
		return "", fmt.Errorf("load Asia/Seoul timezone: %w", err)
	}
	return now.In(location).Format(time.DateOnly), nil
}

func validCalendarDate(value string) bool {
	parsed, err := time.Parse(time.DateOnly, value)
	return err == nil && parsed.Format(time.DateOnly) == value
}

func deduplicateStrings(values []string) []string {
	seen := make(map[string]bool, len(values))
	result := make([]string, 0, len(values))
	for _, value := range values {
		if !seen[value] {
			seen[value] = true
			result = append(result, value)
		}
	}
	return result
}

func findRepositoryRoot(start string) (string, error) {
	current, err := filepath.Abs(start)
	if err != nil {
		return "", fmt.Errorf("resolve working directory: %w", err)
	}
	for {
		module, err := os.ReadFile(filepath.Join(current, "go.mod"))
		if err == nil && strings.Contains(string(module), "module github.com/cpaikr/opendart\n") {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", errors.New("repository root containing the OpenDART Go module was not found")
		}
		current = parent
	}
}
