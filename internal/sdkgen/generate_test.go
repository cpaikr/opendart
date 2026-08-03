package sdkgen

import (
	"bytes"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"sync"
	"testing"

	"github.com/cpaikr/opendart/internal/sdkgen/model"
	rustemitter "github.com/cpaikr/opendart/internal/sdkgen/rust"
)

func TestGenerateCLIIsDeterministicCompleteAndFresh(t *testing.T) {
	inputs := canonicalInputs(t)
	left, right := t.TempDir(), t.TempDir()
	leftReport, err := GenerateCLI(inputs, left)
	if err != nil {
		t.Fatal(err)
	}
	rightReport, err := GenerateCLI(inputs, right)
	if err != nil {
		t.Fatal(err)
	}
	if len(leftReport.Artifacts) != 2 || !reflect.DeepEqual(leftReport, rightReportWithRoot(rightReport, left, right)) {
		t.Fatalf("reports = %#v and %#v", leftReport, rightReport)
	}
	if !reflect.DeepEqual(readTestTree(t, left), readTestTree(t, right)) {
		t.Fatal("generation in distinct roots produced different bytes")
	}
	if err := CheckCLIFresh(inputs, left); err != nil {
		t.Fatal(err)
	}
	if _, err := GenerateCLI(inputs, left); err != nil {
		t.Fatalf("replace accepted owned output: %v", err)
	}
	if err := CheckCLIFresh(inputs, left); err != nil {
		t.Fatal(err)
	}
}

func rightReportWithRoot(report Report, left, right string) Report {
	for index := range report.Artifacts {
		report.Artifacts[index].Output = filepath.Join(left, stringsTrimPrefixPath(report.Artifacts[index].Output, right))
	}
	return report
}

func stringsTrimPrefixPath(path, root string) string {
	relative, err := filepath.Rel(root, path)
	if err != nil {
		return path
	}
	return relative
}

func TestGenerateCLIAcceptsOpenAPIOutsideRepositoryLayout(t *testing.T) {
	inputs := canonicalInputs(t)
	openAPIRoot := filepath.Dir(inputs.OpenAPI)
	copiedRoot := filepath.Join(t.TempDir(), "custom-contract")
	if err := os.CopyFS(copiedRoot, os.DirFS(openAPIRoot)); err != nil {
		t.Fatal(err)
	}
	inputs.OpenAPI = filepath.Join(copiedRoot, filepath.Base(inputs.OpenAPI))
	if _, err := GenerateCLI(inputs, t.TempDir()); err != nil {
		t.Fatalf("generate from independently located inputs: %v", err)
	}
}

func TestCheckCLIFreshClassifiesWrappedInterfaceInputFailure(t *testing.T) {
	inputs := canonicalInputs(t)
	inputs.Interface = t.TempDir()
	err := CheckCLIFresh(inputs, t.TempDir())
	if !errors.Is(err, ErrRustInterfaceInput) || !errors.Is(err, os.ErrNotExist) {
		t.Fatalf("error = %v", err)
	}
}

func TestCheckCLIFreshRejectsTreeDrift(t *testing.T) {
	tests := []struct {
		name string
		edit func(*testing.T, string)
		want error
	}{
		{name: "missing tree", edit: func(*testing.T, string) {}, want: ErrGeneratedMissing},
		{name: "half-published tree", edit: func(t *testing.T, output string) {
			generateTestTree(t, output)
			if err := os.RemoveAll(filepath.Join(output, "dispatch")); err != nil {
				t.Fatal(err)
			}
		}, want: ErrGeneratedMissing},
		{name: "stale interface", edit: func(t *testing.T, output string) {
			generateTestTree(t, output)
			writeTestFile(t, filepath.Join(output, "interface", "catalog.rs"), "stale")
		}, want: ErrGeneratedStale},
		{name: "stale dispatch", edit: func(t *testing.T, output string) {
			generateTestTree(t, output)
			writeTestFile(t, filepath.Join(output, "dispatch", "adapter.rs"), "stale")
		}, want: ErrGeneratedStale},
		{name: "unexpected projection", edit: func(t *testing.T, output string) {
			generateTestTree(t, output)
			writeTestFile(t, filepath.Join(output, "unexpected.rs"), "unexpected")
		}, want: ErrGeneratedUnexpected},
		{name: "unexpected owned file", edit: func(t *testing.T, output string) {
			generateTestTree(t, output)
			writeTestFile(t, filepath.Join(output, "dispatch", "unexpected.rs"), "unexpected")
		}, want: ErrGeneratedUnexpected},
		{name: "invalid marker", edit: func(t *testing.T, output string) {
			generateTestTree(t, output)
			writeTestFile(t, filepath.Join(output, "interface", ".opendart-cli-interface-generated"), "changed")
		}, want: ErrGeneratedUnowned},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			output := filepath.Join(t.TempDir(), "cli")
			test.edit(t, output)
			if err := checkTestTreeFresh(t, output); !errors.Is(err, test.want) {
				t.Fatalf("error = %v, want %v", err, test.want)
			}
		})
	}
}

func TestGenerateCLIPreflightsBothTreesBeforeReplacingEither(t *testing.T) {
	output := filepath.Join(t.TempDir(), "cli")
	generateTestTree(t, output)
	staleInterface := []byte("accepted stale bytes")
	if err := os.WriteFile(filepath.Join(output, "interface", "catalog.rs"), staleInterface, 0o644); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(output, "dispatch", ".opendart-cli-dispatch-generated"), "unowned")
	if _, err := generateTestArtifacts(t, output); !errors.Is(err, ErrGeneratedUnowned) {
		t.Fatalf("error = %v, want %v", err, ErrGeneratedUnowned)
	}
	content, err := os.ReadFile(filepath.Join(output, "interface", "catalog.rs"))
	if err != nil || !bytes.Equal(content, staleInterface) {
		t.Fatalf("interface changed before dispatch ownership passed: %q, %v", content, err)
	}
}

func TestPublishRollbackErrorPreservesBothCauses(t *testing.T) {
	publishErr, rollbackErr := errors.New("publish"), errors.New("rollback")
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
	writeTestFile(t, filepath.Join(backup, "accepted.rs"), "accepted")
	product := &stagedProduct{
		ownedProduct: ownedProduct{kind: "cli-dispatch", output: filepath.Join(root, "missing-parent", "generated")},
		backup:       backup, backupOwned: true,
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

type canonicalCLIArtifacts struct {
	generated model.ArtifactSet
	files     rustemitter.Artifacts
}

var loadCanonicalCLIArtifactsOnce = sync.OnceValues(func() (canonicalCLIArtifacts, error) {
	inputs, err := canonicalInputPaths()
	if err != nil {
		return canonicalCLIArtifacts{}, err
	}
	generated, files, err := renderCLI(inputs)
	return canonicalCLIArtifacts{generated: generated, files: files}, err
})

func canonicalTestArtifacts(t *testing.T) canonicalCLIArtifacts {
	t.Helper()
	artifacts, err := loadCanonicalCLIArtifactsOnce()
	if err != nil {
		t.Fatal(err)
	}
	return artifacts
}

func generateTestArtifacts(t *testing.T, output string) (Report, error) {
	t.Helper()
	artifacts := canonicalTestArtifacts(t)
	return generateCLIArtifacts(artifacts.generated, artifacts.files, output)
}

func generateTestTree(t *testing.T, output string) {
	t.Helper()
	if _, err := generateTestArtifacts(t, output); err != nil {
		t.Fatal(err)
	}
}

func checkTestTreeFresh(t *testing.T, output string) error {
	t.Helper()
	artifacts := canonicalTestArtifacts(t)
	return checkCLIArtifactsFresh(artifacts.generated, artifacts.files, output)
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

func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func canonicalInputs(t *testing.T) Inputs {
	t.Helper()
	inputs, err := canonicalInputPaths()
	if err != nil {
		t.Fatal(err)
	}
	return inputs
}

func canonicalInputPaths() (Inputs, error) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		return Inputs{}, errors.New("cannot locate test source")
	}
	repository := filepath.Join(filepath.Dir(current), "..", "..")
	return Inputs{
		OpenAPI:   filepath.Join(repository, "openapi", "openapi.yaml"),
		Interface: filepath.Join(repository, "sdk", "rust", "interface"),
	}, nil
}
