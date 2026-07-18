package openapi

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// This differential gate is intentionally opt-in while Redocly remains as
// temporary parity scaffolding. Work 5 removes it with the dormant Node stack.
func TestRedoclyStrictMutationParity(t *testing.T) {
	if os.Getenv("OPENDART_REDOCly_PARITY") != "1" {
		t.Skip("set OPENDART_REDOCly_PARITY=1 to run the Redocly differential gate")
	}
	repositoryRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	redocly := filepath.Join(repositoryRoot, "node_modules", "@redocly", "cli", "bin", "cli.js")
	config := filepath.Join(repositoryRoot, "openapi", "redocly.yaml")
	if _, err := os.Stat(redocly); err != nil {
		t.Fatalf("Redocly parity dependency is unavailable: %v", err)
	}

	assertRedoclyAccepts(t, redocly, config, strictLintFixture)
	for _, mutation := range strictLintMutationCases {
		t.Run(mutation.name, func(t *testing.T) {
			source := strings.Replace(strictLintFixture, mutation.old, mutation.replacement, 1)
			if source == strictLintFixture {
				t.Fatalf("fixture does not contain %q", mutation.old)
			}
			root := filepath.Join(t.TempDir(), "openapi.yaml")
			if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			output, err := runRedocly(redocly, config, root)
			if mutation.goOnly {
				if err != nil {
					t.Fatalf("strengthening mutation unexpectedly failed Redocly: %v\n%s", err, output)
				}
				return
			}
			if err == nil {
				t.Fatalf("Redocly accepted mutation covered by Go rule %s: %s", mutation.rule, output)
			}
		})
	}
	for _, mutation := range strictValidationMutationCases {
		t.Run("structural "+mutation.name, func(t *testing.T) {
			source := strings.Replace(strictLintFixture, mutation.old, mutation.replacement, 1)
			root := filepath.Join(t.TempDir(), "openapi.yaml")
			if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			if output, err := runRedocly(redocly, config, root); err == nil {
				t.Fatalf("Redocly accepted a mutation rejected by Go structural validation: %s", output)
			}
		})
	}
}

func assertRedoclyAccepts(t *testing.T, redocly, config, source string) {
	t.Helper()
	root := filepath.Join(t.TempDir(), "openapi.yaml")
	if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if output, err := runRedocly(redocly, config, root); err != nil {
		t.Fatalf("valid parity fixture failed Redocly: %v\n%s", err, output)
	}
}

func runRedocly(redocly, config, root string) (string, error) {
	command := exec.Command("node", redocly, "lint", root, "--config", config, "--lint-config", "error", "--max-problems", "1000", "--format", "stylish")
	var output bytes.Buffer
	command.Stdout = &output
	command.Stderr = &output
	err := command.Run()
	return output.String(), err
}
