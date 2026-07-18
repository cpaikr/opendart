package guide

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestPublishGeneratedRefusesOutputSymlink(t *testing.T) {
	root := t.TempDir()
	physicalOutput := filepath.Join(root, "physical")
	output := filepath.Join(root, "openapi")
	staging := filepath.Join(root, "stage")
	writeManagedFixture(t, physicalOutput, "old")
	writeManagedFixture(t, staging, "new")
	if err := os.Symlink(physicalOutput, output); err != nil {
		t.Fatal(err)
	}
	if err := publishGenerated(staging, output, filepath.Join(root, "repository")); err == nil {
		t.Fatal("output symlink was accepted")
	}
	assertFileContent(t, filepath.Join(physicalOutput, "openapi.yaml"), "old")
}

func TestPublishGeneratedRefusesInvalidStagingMarker(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	staging := filepath.Join(root, "stage")
	writeManagedFixture(t, output, "old")
	writeManagedFixture(t, staging, "new")
	if err := os.WriteFile(filepath.Join(staging, OutputMarker), []byte("invalid\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := publishGenerated(staging, output, filepath.Join(root, "repository")); err == nil {
		t.Fatal("invalid staging marker was accepted")
	}
	assertFileContent(t, filepath.Join(output, "openapi.yaml"), "old")
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

func TestPublishGeneratedKeepsOwnershipMarkerDuringReplacement(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	staging := filepath.Join(root, "stage")
	writeManagedFixture(t, staging, "new")
	observed := false
	err := publishGeneratedWithHook(staging, output, filepath.Join(root, "repository"), func(phase, name string) error {
		if phase == "before-new" && name == "paths" {
			assertFileContent(t, filepath.Join(output, OutputMarker), OutputMarkerContent)
			observed = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if !observed {
		t.Fatal("publication hook did not observe the ownership marker")
	}
}

func TestPublishGeneratedRollsBackBundleInvalidationFailure(t *testing.T) {
	root := t.TempDir()
	output := filepath.Join(root, "openapi")
	staging := filepath.Join(root, "stage")
	writeManagedFixture(t, output, "old")
	writeManagedFixture(t, staging, "new")
	bundle := filepath.Join(output, "generated", "openapi.bundle.yaml")
	if err := os.MkdirAll(bundle, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(bundle, "keep"), []byte("not removable as a file"), 0o600); err != nil {
		t.Fatal(err)
	}

	err := publishGenerated(staging, output, filepath.Join(root, "repository"))
	if err == nil || !strings.Contains(err.Error(), "invalidate generated bundle") {
		t.Fatalf("error = %v", err)
	}
	assertFileContent(t, filepath.Join(output, "openapi.yaml"), "old")
	assertFileContent(t, filepath.Join(output, OutputMarker), OutputMarkerContent)
	assertFileContent(t, filepath.Join(bundle, "keep"), "not removable as a file")
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
