package guide

import (
	"errors"
	"os"
	"path/filepath"
	"testing"
)

func TestPublishGeneratedReplacesOwnedOutputAndInvalidatesBundle(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	staging := filepath.Join(root, "stage")
	writeManagedFixture(t, output, "old")
	writeManagedFixture(t, staging, "new")
	if err := os.MkdirAll(filepath.Join(output, "generated"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "generated", "openapi.bundle.yaml"), []byte("stale"), 0o600); err != nil {
		t.Fatal(err)
	}

	if err := publishGenerated(staging, output, filepath.Join(root, "repository")); err != nil {
		t.Fatal(err)
	}
	assertFileContent(t, filepath.Join(output, "openapi.yaml"), "new")
	if _, err := os.Stat(filepath.Join(output, "generated", "openapi.bundle.yaml")); !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("bundle still exists: %v", err)
	}
}

func TestPublishGeneratedRefusesUnownedOutput(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	staging := filepath.Join(root, "stage")
	if err := os.MkdirAll(output, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(output, "openapi.yaml"), []byte("owned-by-user"), 0o600); err != nil {
		t.Fatal(err)
	}
	writeManagedFixture(t, staging, "new")
	if err := publishGenerated(staging, output, filepath.Join(root, "repository")); err == nil {
		t.Fatal("unowned output was replaced")
	}
	assertFileContent(t, filepath.Join(output, "openapi.yaml"), "owned-by-user")
}

func TestPublishGeneratedRollsBackFailure(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	staging := filepath.Join(root, "stage")
	writeManagedFixture(t, output, "old")
	writeManagedFixture(t, staging, "new")
	failure := errors.New("injected publish failure")
	err := publishGeneratedWithHook(staging, output, filepath.Join(root, "repository"), func(phase, name string) error {
		if phase == "before-new" && name == "schemas" {
			return failure
		}
		return nil
	})
	if !errors.Is(err, failure) {
		t.Fatalf("publish error = %v", err)
	}
	assertFileContent(t, filepath.Join(output, "openapi.yaml"), "old")
	assertFileContent(t, filepath.Join(output, OutputMarker), OutputMarkerContent)
}

func writeManagedFixture(t *testing.T, directory, value string) {
	t.Helper()
	for _, name := range []string{"paths", "schemas", "components"} {
		if err := os.MkdirAll(filepath.Join(directory, name), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(directory, name, "fixture.yaml"), []byte(value), 0o600); err != nil {
			t.Fatal(err)
		}
	}
	if err := os.WriteFile(filepath.Join(directory, "openapi.yaml"), []byte(value), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(directory, OutputMarker), []byte(OutputMarkerContent), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertFileContent(t *testing.T, path, want string) {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != want {
		t.Fatalf("%s = %q, want %q", path, data, want)
	}
}
