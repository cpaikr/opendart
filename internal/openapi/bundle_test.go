package openapi

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestGenerateBundleIsDeterministicAndPortable(t *testing.T) {
	root := fixturePath(t, "openapi.yaml")
	first, err := GenerateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	second, err := GenerateBundle(root)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Fatal("GenerateBundle returned different bytes for unchanged source")
	}
	var value map[string]any
	if err := yaml.Unmarshal(first, &value); err != nil {
		t.Fatal(err)
	}
	assertBundleReferencesAreInternal(t, value)
	companySchema := nestedMap(t, value, "paths", "/company.xml", "get", "responses", "200", "content", "application/xml", "schema")
	xmlMetadata := nestedMap(t, companySchema, "xml")
	if xmlMetadata["nodeType"] != "element" || xmlMetadata["name"] != "result" {
		t.Fatalf("XML metadata = %#v", xmlMetadata)
	}
	extension := nestedMap(t, companySchema, "x-opendart")
	if extension["schemaStatus"] != "source-derived-unverified" || extension["sourceRootKey"] != "result" {
		t.Fatalf("x-opendart = %#v", extension)
	}
}

func assertBundleReferencesAreInternal(t *testing.T, value any) {
	t.Helper()
	switch current := value.(type) {
	case map[string]any:
		for key, child := range current {
			if key == "$ref" {
				reference, ok := child.(string)
				if !ok || !strings.HasPrefix(reference, "#/") {
					t.Fatalf("bundle contains external reference %#v", child)
				}
			}
			assertBundleReferencesAreInternal(t, child)
		}
	case []any:
		for _, child := range current {
			assertBundleReferencesAreInternal(t, child)
		}
	}
}

func TestWriteBundleRequiresExplicitOutputAndPublishesFreshArtifact(t *testing.T) {
	root := fixturePath(t, "openapi.yaml")
	if err := WriteBundle(root, ""); err == nil || !strings.Contains(err.Error(), "output path is required") {
		t.Fatalf("WriteBundle empty output error = %v", err)
	}

	output := filepath.Join(t.TempDir(), "generated", "openapi.bundle.yaml")
	if err := WriteBundle(root, output); err != nil {
		t.Fatal(err)
	}
	if err := CheckBundleFresh(root, output); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(output)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o644 {
		t.Fatalf("bundle mode = %o, want 644", info.Mode().Perm())
	}
}

func TestWriteFileAtomicallyPreservesExistingArtifactOnReplacementFailure(t *testing.T) {
	directory := t.TempDir()
	output := filepath.Join(directory, "openapi.bundle.yaml")
	if err := os.WriteFile(output, []byte("accepted"), 0o644); err != nil {
		t.Fatal(err)
	}
	replacementError := errors.New("replacement failed")
	err := writeFileAtomically(output, []byte("replacement"), func(_, _ string) error {
		return replacementError
	})
	if !errors.Is(err, replacementError) {
		t.Fatalf("writeFileAtomically error = %v", err)
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "accepted" {
		t.Fatalf("existing artifact = %q", content)
	}
	temporary, err := filepath.Glob(filepath.Join(directory, ".openapi.bundle.yaml.tmp-*"))
	if err != nil {
		t.Fatal(err)
	}
	if len(temporary) != 0 {
		t.Fatalf("temporary bundle remains after failure: %v", temporary)
	}
}

func TestWriteBundleDoesNotTruncateOutputWhenGenerationFails(t *testing.T) {
	output := filepath.Join(t.TempDir(), "openapi.bundle.yaml")
	if err := os.WriteFile(output, []byte("accepted"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteBundle("missing-openapi.yaml", output); err == nil {
		t.Fatal("WriteBundle succeeded with missing source")
	}
	content, err := os.ReadFile(output)
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "accepted" {
		t.Fatalf("existing artifact = %q", content)
	}
}

func TestCheckBundleFreshReportsMissingAndStaleArtifacts(t *testing.T) {
	root := fixturePath(t, "openapi.yaml")
	directory := t.TempDir()
	missing := filepath.Join(directory, "missing.yaml")
	if err := CheckBundleFresh(root, missing); !errors.Is(err, ErrBundleMissing) || !strings.Contains(err.Error(), missing) {
		t.Fatalf("missing error = %v", err)
	}

	stale := filepath.Join(directory, "stale.yaml")
	if err := os.WriteFile(stale, []byte("stale"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := CheckBundleFresh(root, stale); !errors.Is(err, ErrBundleStale) || !strings.Contains(err.Error(), stale) {
		t.Fatalf("stale error = %v", err)
	}
}
