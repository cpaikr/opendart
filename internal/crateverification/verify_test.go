package crateverification

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

const (
	testPackage  = "opendart-cli"
	testVersion  = "0.1.0"
	testRevision = "0123456789abcdef0123456789abcdef01234567"
	testVCSPath  = "sdk/rust/crates/opendart-cli"
)

type testEntry struct {
	name     string
	content  string
	typeflag byte
}

func TestVerifyAcceptsEquivalentContentsWithDifferentGzipBytes(t *testing.T) {
	fixture := newFixture(t, baseEntries(false))
	candidate, candidateChecksum := writeArchive(t, "candidate.crate", fixture.entries, "candidate", time.Unix(1, 0))
	accepted, acceptedChecksum := writeArchive(t, "accepted.crate", fixture.entries, "accepted", time.Unix(2, 0))
	if candidateChecksum == acceptedChecksum {
		t.Fatal("gzip fixtures unexpectedly have the same checksum")
	}

	report, err := Verify(fixture.options(candidate, accepted, acceptedChecksum))
	if err != nil {
		t.Fatal(err)
	}
	if report.Kind != reportKind || report.SchemaVersion != 1 || report.Package != testPackage || report.Version != testVersion || report.Revision != testRevision {
		t.Fatalf("report = %#v", report)
	}
	if report.RegistryChecksum != acceptedChecksum || report.CandidateChecksum != candidateChecksum || report.FileCount != len(fixture.entries) {
		t.Fatalf("report evidence = %#v", report)
	}
}

func TestVerifyRejectsChecksumContentInventoryManifestAndVCSMismatches(t *testing.T) {
	tests := []struct {
		name          string
		candidate     []testEntry
		accepted      []testEntry
		inventory     []string
		revision      string
		checksum      string
		wantArtifact  string
		wantInvariant string
	}{
		{
			name: "registry checksum", candidate: baseEntries(false), accepted: baseEntries(false),
			checksum: strings.Repeat("0", 64), wantArtifact: "accepted", wantInvariant: "registry checksum",
		},
		{
			name: "content", candidate: baseEntries(false), accepted: replaceEntry(baseEntries(false), "README.md", "changed"),
			wantArtifact: "accepted", wantInvariant: "candidate package contents",
		},
		{
			name: "inventory", candidate: baseEntries(false), accepted: baseEntries(false),
			inventory:    []string{".cargo_vcs_info.json", "Cargo.toml", "Cargo.toml.orig"},
			wantArtifact: "candidate", wantInvariant: "reviewed package inventory",
		},
		{
			name:         "manifest identity",
			candidate:    replaceEntry(baseEntries(false), "Cargo.toml", manifest("other", testVersion)),
			accepted:     replaceEntry(baseEntries(false), "Cargo.toml", manifest("other", testVersion)),
			wantArtifact: "candidate", wantInvariant: "manifest identity",
		},
		{
			name:         "workspace-inherited manifest identity",
			candidate:    replaceEntry(baseEntries(false), "Cargo.toml", "[package]\nname.workspace = true\nversion.workspace = true\n"),
			accepted:     replaceEntry(baseEntries(false), "Cargo.toml", "[package]\nname.workspace = true\nversion.workspace = true\n"),
			wantArtifact: "candidate", wantInvariant: "manifest identity",
		},
		{
			name: "revision", candidate: baseEntries(false), accepted: baseEntries(false),
			revision: strings.Repeat("f", 40), wantArtifact: "candidate", wantInvariant: "reviewed clean revision",
		},
		{
			name:      "dirty VCS",
			candidate: baseEntries(true), accepted: baseEntries(true),
			wantArtifact: "candidate", wantInvariant: "reviewed clean revision",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFixture(t, test.candidate)
			if test.inventory != nil {
				fixture.writeInventory(t, test.inventory)
			}
			candidate, _ := writeArchive(t, "candidate.crate", test.candidate, "candidate", time.Unix(1, 0))
			accepted, acceptedChecksum := writeArchive(t, "accepted.crate", test.accepted, "accepted", time.Unix(2, 0))
			options := fixture.options(candidate, accepted, acceptedChecksum)
			if test.revision != "" {
				options.Revision = test.revision
			}
			if test.checksum != "" {
				options.RegistryChecksum = test.checksum
			}
			_, err := Verify(options)
			assertInvariant(t, err, test.wantArtifact, test.wantInvariant)
		})
	}
}

func TestVerifyRejectsUnsafeDuplicateLinkedTrailingAndOversizedArchives(t *testing.T) {
	tests := []struct {
		name                 string
		accepted             []testEntry
		appendTrailing       bool
		decompressedTrailing bool
		nonzeroPadding       bool
		bounds               limits
		wantArtifact         string
		wantInvariant        string
	}{
		{
			name:          "unsafe path",
			accepted:      append(baseEntries(false), testEntry{name: testPackage + "-" + testVersion + "/../escape", content: "x", typeflag: tar.TypeReg}),
			wantInvariant: "safe package-root paths",
		},
		{
			name:          "duplicate",
			accepted:      append(baseEntries(false), testEntry{name: rootName() + "README.md", content: "duplicate", typeflag: tar.TypeReg}),
			wantInvariant: "duplicate paths",
		},
		{
			name:          "link",
			accepted:      append(baseEntries(false), testEntry{name: rootName() + "link", content: "", typeflag: tar.TypeSymlink}),
			wantInvariant: "regular files only",
		},
		{
			name: "trailing compressed data", accepted: baseEntries(false), appendTrailing: true,
			wantInvariant: "trailing compressed data",
		},
		{
			name: "hidden decompressed tail", accepted: baseEntries(false), decompressedTrailing: true,
			wantInvariant: "hidden decompressed tar tail",
		},
		{
			name: "nonzero file padding", accepted: baseEntries(false), nonzeroPadding: true,
			wantInvariant: "zero-filled file padding",
		},
		{
			name: "expanded bound", accepted: baseEntries(false),
			bounds:        limits{maxArchiveBytes: defaultMaxArchiveBytes, maxExpandedBytes: 128, maxFileBytes: defaultMaxFileBytes, maxFiles: defaultMaxFiles},
			wantArtifact:  "candidate",
			wantInvariant: "expanded size limit",
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			fixture := newFixture(t, baseEntries(false))
			candidate, _ := writeArchive(t, "candidate.crate", baseEntries(false), "candidate", time.Unix(1, 0))
			accepted, acceptedChecksum := writeArchiveWithTail(t, "accepted.crate", test.accepted, "accepted", time.Unix(2, 0), "")
			if test.decompressedTrailing {
				accepted, acceptedChecksum = writeArchiveWithTail(t, "accepted.crate", test.accepted, "accepted", time.Unix(2, 0), "hidden")
			}
			if test.nonzeroPadding {
				corruptFirstFilePadding(t, accepted, len(test.accepted[0].content))
				acceptedChecksum = checksumFile(t, accepted)
			}
			if test.appendTrailing {
				file, err := os.OpenFile(accepted, os.O_APPEND|os.O_WRONLY, 0)
				if err != nil {
					t.Fatal(err)
				}
				if _, err := file.Write([]byte("trailing")); err != nil {
					t.Fatal(err)
				}
				if err := file.Close(); err != nil {
					t.Fatal(err)
				}
				acceptedChecksum = checksumFile(t, accepted)
			}
			options := fixture.options(candidate, accepted, acceptedChecksum)
			bounds := test.bounds
			if bounds.maxArchiveBytes == 0 {
				bounds = limits{defaultMaxArchiveBytes, defaultMaxExpandedBytes, defaultMaxFileBytes, defaultMaxFiles}
			}
			_, err := verifyWithLimits(options, bounds)
			artifact := test.wantArtifact
			if artifact == "" {
				artifact = "accepted"
			}
			assertInvariant(t, err, artifact, test.wantInvariant)
		})
	}
}

func TestVerifyRejectsSameFileAndKeepsLocalPathsOutOfErrors(t *testing.T) {
	fixture := newFixture(t, baseEntries(false))
	candidate, checksum := writeArchive(t, "secret-local-name.crate", fixture.entries, "candidate", time.Unix(1, 0))
	_, err := Verify(fixture.options(candidate, candidate, checksum))
	assertInvariant(t, err, "artifacts", "distinct candidate and accepted")
	if strings.Contains(err.Error(), candidate) || strings.Contains(err.Error(), filepath.Dir(candidate)) {
		t.Fatalf("error exposes local path: %q", err)
	}
}

func TestVerifyRejectsInventorySymlink(t *testing.T) {
	fixture := newFixture(t, baseEntries(false))
	candidate, _ := writeArchive(t, "candidate.crate", fixture.entries, "candidate", time.Unix(1, 0))
	accepted, checksum := writeArchive(t, "accepted.crate", fixture.entries, "accepted", time.Unix(2, 0))
	link := filepath.Join(fixture.directory, "inventory-link.txt")
	if err := os.Symlink(fixture.inventoryPath, link); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	options := fixture.options(candidate, accepted, checksum)
	options.InventoryPath = link
	_, err := Verify(options)
	assertInvariant(t, err, "inventory", "regular file")
}

func TestVerifyRejectsMalformedAcceptedGzip(t *testing.T) {
	fixture := newFixture(t, baseEntries(false))
	candidate, _ := writeArchive(t, "candidate.crate", fixture.entries, "candidate", time.Unix(1, 0))
	accepted := filepath.Join(t.TempDir(), "accepted.crate")
	if err := os.WriteFile(accepted, []byte("not a crate archive"), 0o600); err != nil {
		t.Fatal(err)
	}
	_, err := Verify(fixture.options(candidate, accepted, checksumFile(t, accepted)))
	assertInvariant(t, err, "accepted", "valid gzip stream")
}

type fixture struct {
	directory     string
	inventoryPath string
	entries       []testEntry
}

func newFixture(t *testing.T, entries []testEntry) fixture {
	t.Helper()
	directory := t.TempDir()
	result := fixture{directory: directory, inventoryPath: filepath.Join(directory, "inventory.txt"), entries: entries}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		name := strings.TrimPrefix(entry.name, rootName())
		if safeRelativePath(name) && !slices.Contains(names, name) {
			names = append(names, name)
		}
	}
	slices.Sort(names)
	result.writeInventory(t, names)
	return result
}

func (f fixture) writeInventory(t *testing.T, names []string) {
	t.Helper()
	if err := os.WriteFile(f.inventoryPath, []byte(strings.Join(names, "\n")+"\n"), 0o600); err != nil {
		t.Fatal(err)
	}
}

func (f fixture) options(candidate, accepted, checksum string) Options {
	return Options{
		CandidatePath: candidate, AcceptedPath: accepted, InventoryPath: f.inventoryPath,
		Package: testPackage, Version: testVersion, Revision: testRevision,
		VCSPath: testVCSPath, RegistryChecksum: checksum,
	}
}

func baseEntries(dirty bool) []testEntry {
	dirtyField := ""
	if dirty {
		dirtyField = `,"dirty":true`
	}
	return []testEntry{
		{name: rootName() + ".cargo_vcs_info.json", content: fmt.Sprintf(`{"git":{"sha1":%q%s},"path_in_vcs":%q}`, testRevision, dirtyField, testVCSPath), typeflag: tar.TypeReg},
		{name: rootName() + "Cargo.toml", content: manifest(testPackage, testVersion), typeflag: tar.TypeReg},
		{name: rootName() + "Cargo.toml.orig", content: manifest(testPackage, testVersion), typeflag: tar.TypeReg},
		{name: rootName() + "README.md", content: "reviewed\n", typeflag: tar.TypeReg},
	}
}

func manifest(name, version string) string {
	return fmt.Sprintf("[package]\nname = %q\nversion = %q\n", name, version)
}

func rootName() string {
	return testPackage + "-" + testVersion + "/"
}

func replaceEntry(entries []testEntry, name, content string) []testEntry {
	result := slices.Clone(entries)
	for index := range result {
		if strings.TrimPrefix(result[index].name, rootName()) == name {
			result[index].content = content
			return result
		}
	}
	panic("fixture entry not found: " + name)
}

func writeArchive(t *testing.T, filename string, entries []testEntry, gzipName string, modTime time.Time) (string, string) {
	t.Helper()
	return writeArchiveWithTail(t, filename, entries, gzipName, modTime, "")
}

func writeArchiveWithTail(t *testing.T, filename string, entries []testEntry, gzipName string, modTime time.Time, decompressedTail string) (string, string) {
	t.Helper()
	path := filepath.Join(t.TempDir(), filename)
	file, err := os.Create(path)
	if err != nil {
		t.Fatal(err)
	}
	compressed := gzip.NewWriter(file)
	compressed.Name = gzipName
	compressed.ModTime = modTime
	archive := tar.NewWriter(compressed)
	for _, entry := range entries {
		typeflag := entry.typeflag
		if typeflag == 0 {
			typeflag = tar.TypeReg
		}
		header := &tar.Header{Name: entry.name, Mode: 0o644, Size: int64(len(entry.content)), Typeflag: typeflag, ModTime: time.Unix(0, 0)}
		if typeflag == tar.TypeSymlink {
			header.Linkname = "README.md"
			header.Size = 0
		}
		if err := archive.WriteHeader(header); err != nil {
			t.Fatal(err)
		}
		if header.Size != 0 {
			if _, err := archive.Write([]byte(entry.content)); err != nil {
				t.Fatal(err)
			}
		}
	}
	if err := archive.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := compressed.Write([]byte(decompressedTail)); err != nil {
		t.Fatal(err)
	}
	if err := compressed.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	return path, checksumFile(t, path)
}

func checksumFile(t *testing.T, filename string) string {
	t.Helper()
	content, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	sum := sha256.Sum256(content)
	return hex.EncodeToString(sum[:])
}

func corruptFirstFilePadding(t *testing.T, filename string, fileSize int) {
	t.Helper()
	compressedBytes, err := os.ReadFile(filename)
	if err != nil {
		t.Fatal(err)
	}
	reader, err := gzip.NewReader(bytes.NewReader(compressedBytes))
	if err != nil {
		t.Fatal(err)
	}
	expanded, err := io.ReadAll(reader)
	if err != nil {
		t.Fatal(err)
	}
	if err := reader.Close(); err != nil {
		t.Fatal(err)
	}
	paddingOffset := 512 + fileSize
	if paddingOffset >= len(expanded) || fileSize%512 == 0 {
		t.Fatal("fixture does not have writable first-file padding")
	}
	expanded[paddingOffset] = 1
	file, err := os.Create(filename)
	if err != nil {
		t.Fatal(err)
	}
	writer := gzip.NewWriter(file)
	if _, err := writer.Write(expanded); err != nil {
		t.Fatal(err)
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
}

func assertInvariant(t *testing.T, err error, artifact, invariant string) {
	t.Helper()
	var verificationError *Error
	if !errorsAs(err, &verificationError) || verificationError.Artifact != artifact || !strings.Contains(verificationError.Invariant, invariant) {
		t.Fatalf("error = %#v, want %s/%s", err, artifact, invariant)
	}
}

func errorsAs(err error, target **Error) bool {
	return errors.As(err, target)
}
