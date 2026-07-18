package guide

import (
	"context"
	"errors"
	"strings"
	"testing"
)

type recordedCommand struct {
	Name string
	Args []string
}

type recordingRunner struct {
	commands []recordedCommand
	errAt    int
}

func (r *recordingRunner) Run(_ context.Context, name string, args ...string) error {
	r.commands = append(r.commands, recordedCommand{Name: name, Args: append([]string(nil), args...)})
	if r.errAt > 0 && len(r.commands) == r.errAt {
		return errors.New("injected validation failure")
	}
	return nil
}

func TestValidateStagingRunsCatalogBeforeOpenAPI(t *testing.T) {
	runner := &recordingRunner{}
	if err := validateStaging(context.Background(), "/stage", "/repo", false, runner); err != nil {
		t.Fatal(err)
	}
	if len(runner.commands) != 2 {
		t.Fatalf("commands = %#v", runner.commands)
	}
	if runner.commands[0].Name != "node" || !containsArgument(runner.commands[0].Args, "--structural-only") || !strings.Contains(runner.commands[0].Args[0], "check-opendart.mjs") {
		t.Fatalf("catalog command = %#v", runner.commands[0])
	}
	if runner.commands[1].Name != "node" || !containsArgument(runner.commands[1].Args, "lint") || !strings.Contains(runner.commands[1].Args[0], "@redocly") {
		t.Fatalf("OpenAPI command = %#v", runner.commands[1])
	}
}

func TestValidateStagingStopsBeforeOpenAPIAfterCatalogFailure(t *testing.T) {
	runner := &recordingRunner{errAt: 1}
	err := validateStaging(context.Background(), "/stage", "/repo", true, runner)
	if err == nil || !strings.Contains(err.Error(), "catalog artifact") {
		t.Fatalf("validation error = %v", err)
	}
	if len(runner.commands) != 1 {
		t.Fatalf("commands = %#v", runner.commands)
	}
}

func containsArgument(arguments []string, want string) bool {
	for _, argument := range arguments {
		if argument == want {
			return true
		}
	}
	return false
}
