package sdkgen

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"
)

func TestGenerateRustIsDeterministicAndFresh(t *testing.T) {
	root := canonicalRoot(t)
	left := testOutputs(t.TempDir())
	right := testOutputs(t.TempDir())
	leftReport, err := GenerateRust(root, left)
	if err != nil {
		t.Fatal(err)
	}
	rightReport, err := GenerateRust(root, right)
	if err != nil {
		t.Fatal(err)
	}
	if leftReport.SemanticChecksum == "" || leftReport.SemanticChecksum != rightReport.SemanticChecksum {
		t.Fatalf("reports = %#v and %#v", leftReport, rightReport)
	}
	for _, pair := range [][2]string{{left.SDK, right.SDK}, {left.CLI, right.CLI}} {
		if !reflect.DeepEqual(readTestTree(t, pair[0]), readTestTree(t, pair[1])) {
			t.Fatal("generation in distinct roots produced different bytes")
		}
	}
	if err := CheckRustFresh(root, left); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateRust(root, left); err != nil {
		t.Fatalf("replace accepted owned output: %v", err)
	}
	if err := CheckRustFresh(root, left); err != nil {
		t.Fatal(err)
	}
}

func TestCheckRustFreshRejectsTreeDrift(t *testing.T) {
	root := canonicalRoot(t)
	tests := []struct {
		name string
		edit func(*testing.T, RustOutputs)
		want error
	}{
		{name: "missing tree", edit: func(*testing.T, RustOutputs) {}, want: ErrGeneratedMissing},
		{name: "half-published tree", edit: func(t *testing.T, output RustOutputs) {
			generateTestTree(t, root, output)
			if err := os.RemoveAll(output.CLI); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedMissing},
		{name: "stale sdk file", edit: func(t *testing.T, output RustOutputs) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output.SDK, "mapping.rs"), []byte("stale"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedStale},
		{name: "stale cli file", edit: func(t *testing.T, output RustOutputs) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output.CLI, "catalog.rs"), []byte("stale"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedStale},
		{name: "unexpected file", edit: func(t *testing.T, output RustOutputs) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output.SDK, "unexpected.rs"), []byte("unexpected"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedUnexpected},
		{name: "unexpected cli file", edit: func(t *testing.T, output RustOutputs) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output.CLI, "unexpected.rs"), []byte("unexpected"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedUnexpected},
		{name: "invalid marker", edit: func(t *testing.T, output RustOutputs) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output.CLI, ".opendart-cli-generated"), []byte("changed"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedUnowned},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := testOutputs(t.TempDir())
			test.edit(t, output)
			if err := CheckRustFresh(root, output); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestGenerateRustRefusesToReplaceUnownedOutput(t *testing.T) {
	output := testOutputs(t.TempDir())
	if err := os.Mkdir(output.SDK, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output.SDK, "maintainer.rs"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateRust(canonicalRoot(t), output); !errors.Is(err, ErrGeneratedUnowned) {
		t.Fatalf("error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(output.SDK, "maintainer.rs"))
	if err != nil || !bytes.Equal(content, []byte("keep")) {
		t.Fatalf("unowned file changed: %q, %v", content, err)
	}
}

func TestGenerateRustPreflightsBothTreesBeforeReplacingEither(t *testing.T) {
	root := canonicalRoot(t)
	outputs := testOutputs(t.TempDir())
	generateTestTree(t, root, outputs)
	staleSDK := []byte("accepted stale bytes")
	if err := os.WriteFile(filepath.Join(outputs.SDK, "mapping.rs"), staleSDK, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(outputs.CLI, ".opendart-cli-generated"), []byte("unowned"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateRust(root, outputs); !errors.Is(err, ErrGeneratedUnowned) {
		t.Fatalf("error = %v, want %v", err, ErrGeneratedUnowned)
	}
	content, err := os.ReadFile(filepath.Join(outputs.SDK, "mapping.rs"))
	if err != nil || !bytes.Equal(content, staleSDK) {
		t.Fatalf("SDK changed before CLI ownership passed: %q, %v", content, err)
	}
}

func TestPublishRollbackErrorPreservesBothCauses(t *testing.T) {
	publishErr := errors.New("publish")
	rollbackErr := errors.New("rollback")
	err := publishRollbackError(publishErr, rollbackErr)
	if !errors.Is(err, publishErr) || !errors.Is(err, rollbackErr) {
		t.Fatalf("joined error does not preserve both causes: %v", err)
	}
}

func TestFailedRestoreKeepsRollbackCopyForManualRecovery(t *testing.T) {
	root := t.TempDir()
	backup := filepath.Join(root, "accepted-backup")
	if err := os.Mkdir(backup, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(backup, "accepted.rs"), []byte("accepted"), 0o644); err != nil {
		t.Fatal(err)
	}
	product := &stagedProduct{
		ownedProduct: ownedProduct{kind: "sdk", output: filepath.Join(root, "missing-parent", "generated")},
		backup:       backup,
		backupOwned:  true,
	}
	err := rollbackProducts([]*stagedProduct{product}, errors.New("publish"))
	if err == nil || !product.backupOwned {
		t.Fatalf("rollback = %v, product = %#v", err, product)
	}
	cleanupProducts([]*stagedProduct{product})
	content, readErr := os.ReadFile(filepath.Join(backup, "accepted.rs"))
	if readErr != nil || !bytes.Equal(content, []byte("accepted")) {
		t.Fatalf("manual recovery copy changed: %q, %v", content, readErr)
	}
}

func TestGenerateRustReplacesOnlyOlderOwnedSchemas(t *testing.T) {
	root := canonicalRoot(t)

	t.Run("previous schema", func(t *testing.T) {
		output := testOutputs(t.TempDir())
		generateTestTree(t, root, output)
		marker := filepath.Join(output.SDK, ".opendart-sdk-generated")
		if err := os.WriteFile(marker, []byte("opendart-sdk-generator-schema=1\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if _, err := GenerateRust(root, output); err != nil {
			t.Fatalf("replace previous owned schema: %v", err)
		}
		content, err := os.ReadFile(marker)
		if err != nil || string(content) != "opendart-sdk-generator-schema=2\n" {
			t.Fatalf("marker = %q, %v", content, err)
		}
	})

	for _, marker := range []string{
		"opendart-sdk-generator-schema=0\n",
		"opendart-sdk-generator-schema=3\n",
		"opendart-sdk-generator-schema=1",
		"opendart-sdk-generator-schema=1\nextra",
	} {
		t.Run("reject "+marker, func(t *testing.T) {
			output := testOutputs(t.TempDir())
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output.SDK, ".opendart-sdk-generated"), []byte(marker), 0o644); err != nil {
				t.Fatal(err)
			}
			if _, err := GenerateRust(root, output); !errors.Is(err, ErrGeneratedUnowned) {
				t.Fatalf("error = %v, want %v", err, ErrGeneratedUnowned)
			}
		})
	}
}

func generateTestTree(t *testing.T, root string, output RustOutputs) {
	t.Helper()
	if _, err := GenerateRust(root, output); err != nil {
		t.Fatal(err)
	}
}

func testOutputs(root string) RustOutputs {
	return RustOutputs{SDK: filepath.Join(root, "sdk"), CLI: filepath.Join(root, "cli")}
}

func readTestTree(t *testing.T, root string) map[string]string {
	t.Helper()
	files := make(map[string]string)
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil || entry.IsDir() {
			return err
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(relative)] = string(content)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func canonicalRoot(t *testing.T) string {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	return filepath.Join(filepath.Dir(current), "..", "..", "openapi", "openapi.yaml")
}
