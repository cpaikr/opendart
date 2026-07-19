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
	left := filepath.Join(t.TempDir(), "generated")
	right := filepath.Join(t.TempDir(), "generated")
	leftReport, err := GenerateRust(root, left)
	if err != nil {
		t.Fatal(err)
	}
	rightReport, err := GenerateRust(root, right)
	if err != nil {
		t.Fatal(err)
	}
	if leftReport.Checksum == "" || leftReport.Checksum != rightReport.Checksum {
		t.Fatalf("reports = %#v and %#v", leftReport, rightReport)
	}
	if !reflect.DeepEqual(readTestTree(t, left), readTestTree(t, right)) {
		t.Fatal("generation in distinct roots produced different bytes")
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
		edit func(*testing.T, string)
		want error
	}{
		{name: "missing tree", edit: func(*testing.T, string) {}, want: ErrGeneratedMissing},
		{name: "stale file", edit: func(t *testing.T, output string) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output, "mapping.rs"), []byte("stale"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedStale},
		{name: "unexpected file", edit: func(t *testing.T, output string) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output, "unexpected.rs"), []byte("unexpected"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedUnexpected},
		{name: "invalid marker", edit: func(t *testing.T, output string) {
			generateTestTree(t, root, output)
			if err := os.WriteFile(filepath.Join(output, ".opendart-sdk-generated"), []byte("changed"), 0o644); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedUnowned},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "generated")
			test.edit(t, output)
			if err := CheckRustFresh(root, output); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestGenerateRustRefusesToReplaceUnownedOutput(t *testing.T) {
	output := filepath.Join(t.TempDir(), "generated")
	if err := os.Mkdir(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "maintainer.rs"), []byte("keep"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateRust(canonicalRoot(t), output); !errors.Is(err, ErrGeneratedUnowned) {
		t.Fatalf("error = %v", err)
	}
	content, err := os.ReadFile(filepath.Join(output, "maintainer.rs"))
	if err != nil || !bytes.Equal(content, []byte("keep")) {
		t.Fatalf("unowned file changed: %q, %v", content, err)
	}
}

func generateTestTree(t *testing.T, root, output string) {
	t.Helper()
	if _, err := GenerateRust(root, output); err != nil {
		t.Fatal(err)
	}
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
