package cli

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
	"time"
)

func TestParseSyncCLIOptions(t *testing.T) {
	repository := t.TempDir()
	now := time.Date(2026, 7, 17, 16, 30, 0, 0, time.UTC)
	output := filepath.Join(t.TempDir(), "partial")
	options, err := parseSyncCLIOptions([]string{
		"--output", output,
		"--checked-at", "2026-07-18",
		"--only", "DS001-2019001",
		"--only", "DS001-2019001",
		"--parity-baseline", filepath.Join(repository, "openapi"),
	}, repository, now, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if options.CheckedAt != "2026-07-18" || len(options.Only) != 1 || options.Only[0] != "DS001-2019001" {
		t.Fatalf("options = %#v", options)
	}

	defaultOptions, err := parseSyncCLIOptions(nil, repository, now, &bytes.Buffer{})
	if err != nil {
		t.Fatal(err)
	}
	if defaultOptions.CheckedAt != "2026-07-18" {
		t.Fatalf("Seoul checkedAt = %q", defaultOptions.CheckedAt)
	}
}

func TestParseSyncCLIOptionsRejectsUnsafeShapes(t *testing.T) {
	repository := t.TempDir()
	for _, args := range [][]string{
		{"--checked-at", "2026-02-30"},
		{"--only", "DS001-2019001"},
		{"positional"},
	} {
		if _, err := parseSyncCLIOptions(args, repository, time.Now(), &bytes.Buffer{}); err == nil {
			t.Fatalf("args %v passed", args)
		}
	}
}

func TestParseSyncCLIOptionsRejectsCanonicalSymlinkForPartialSync(t *testing.T) {
	repository := t.TempDir()
	canonical := filepath.Join(repository, "openapi")
	if err := os.Mkdir(canonical, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(repository, "openapi-alias")
	if err := os.Symlink(canonical, alias); err != nil {
		t.Fatal(err)
	}
	if _, err := parseSyncCLIOptions([]string{"--only", "DS001-2019001", "--output", alias}, repository, time.Now(), &bytes.Buffer{}); err == nil {
		t.Fatal("canonical symlink accepted for partial sync")
	}
}

func TestParseSyncCLIOptionsRejectsMissingCanonicalThroughSymlinkedParent(t *testing.T) {
	repository := t.TempDir()
	aliasParent := t.TempDir()
	alias := filepath.Join(aliasParent, "repository")
	if err := os.Symlink(repository, alias); err != nil {
		t.Fatal(err)
	}
	output := filepath.Join(alias, "openapi")
	if _, err := parseSyncCLIOptions([]string{"--only", "DS001-2019001", "--output", output}, repository, time.Now(), &bytes.Buffer{}); err == nil {
		t.Fatal("missing canonical output through symlinked parent accepted for partial sync")
	}
}

func TestFindRepositoryRoot(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "go.mod"), []byte("module github.com/cpaikr/opendart\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	got, err := findRepositoryRoot(child)
	if err != nil {
		t.Fatal(err)
	}
	if got != root {
		t.Fatalf("root = %q, want %q", got, root)
	}
}
