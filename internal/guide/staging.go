package guide

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

type commandRunner interface {
	Run(context.Context, string, ...string) error
}

type execCommandRunner struct {
	directory string
}

func (r execCommandRunner) Run(ctx context.Context, name string, args ...string) error {
	command := exec.CommandContext(ctx, name, args...)
	command.Dir = r.directory
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	if err := command.Run(); err != nil {
		message := strings.TrimSpace(output.String())
		if len(message) > 4<<10 {
			message = message[:4<<10] + "…"
		}
		if message == "" {
			return err
		}
		return fmt.Errorf("%w: %s", err, message)
	}
	return nil
}

func validateStaging(ctx context.Context, staging, repositoryRoot string, complete bool, runner commandRunner) error {
	root := filepath.Join(staging, "openapi.yaml")
	catalogArguments := []string{filepath.Join(repositoryRoot, "scripts", "check-opendart.mjs"), "--root", root}
	if !complete {
		catalogArguments = append(catalogArguments, "--structural-only")
	}
	if err := runner.Run(ctx, "node", catalogArguments...); err != nil {
		return fmt.Errorf("validate staged catalog artifact %s: %w", root, err)
	}
	redoclyArguments := []string{
		filepath.Join(repositoryRoot, "node_modules", "@redocly", "cli", "bin", "cli.js"),
		"lint", root,
		"--config", filepath.Join(repositoryRoot, "openapi", "redocly.yaml"),
		"--lint-config", "error",
		"--max-problems", "1000",
	}
	if err := runner.Run(ctx, "node", redoclyArguments...); err != nil {
		return fmt.Errorf("validate staged OpenAPI artifact %s: %w", root, err)
	}
	return nil
}

func compareStaged(staging, baseline string) error {
	comparison, err := openapispec.Compare(filepath.Join(staging, "openapi.yaml"), filepath.Join(baseline, "openapi.yaml"))
	if err != nil {
		return fmt.Errorf("compare staged OpenAPI with accepted artifact: %w", err)
	}
	if comparison.TotalChanges != 0 {
		return fmt.Errorf("staged OpenAPI differs semantically from accepted artifact at %s", firstChangeLocation(comparison))
	}
	marker := filepath.Join(staging, OutputMarker)
	markerInfo, err := os.Lstat(marker)
	if err != nil {
		return fmt.Errorf("inspect staged ownership marker: %w", err)
	}
	markerContent, err := os.ReadFile(marker)
	if err != nil {
		return fmt.Errorf("read staged ownership marker: %w", err)
	}
	if !markerInfo.Mode().IsRegular() || markerInfo.Mode()&os.ModeSymlink != 0 || string(markerContent) != OutputMarkerContent {
		return errors.New("staged ownership marker is invalid")
	}
	return nil
}

func firstChangeLocation(comparison openapispec.Comparison) string {
	if len(comparison.Details) == 0 || comparison.Details[0].Location == "" {
		return "unknown location"
	}
	return comparison.Details[0].Location
}
