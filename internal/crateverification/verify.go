// Package crateverification verifies local Rust crate candidates against
// accepted registry artifacts without owning download or publication authority.
package crateverification

import (
	"archive/tar"
	"bufio"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"unicode/utf8"
)

const (
	reportKind = "crate-artifact-verification"

	defaultMaxArchiveBytes  = 16 << 20
	defaultMaxExpandedBytes = 64 << 20
	defaultMaxFileBytes     = 16 << 20
	defaultMaxFiles         = 4096
	maxInventoryBytes       = 1 << 20
	maxPathBytes            = 512
)

var (
	packageNamePattern = regexp.MustCompile(`^[a-z0-9]+(?:[-_][a-z0-9]+)*$`)
	versionPattern     = regexp.MustCompile(`^[0-9]+\.[0-9]+\.[0-9]+(?:[-+][0-9A-Za-z.-]+)?$`)
	revisionPattern    = regexp.MustCompile(`^[0-9a-f]{40}$`)
	checksumPattern    = regexp.MustCompile(`^[0-9a-f]{64}$`)
)

// Options identifies two local crate archives and the immutable evidence they
// must match. The caller owns acquiring the accepted artifact and checksum.
type Options struct {
	CandidatePath    string
	AcceptedPath     string
	InventoryPath    string
	Package          string
	Version          string
	Revision         string
	VCSPath          string
	RegistryChecksum string
}

// Report is the bounded, path-free evidence emitted for a matching artifact.
type Report struct {
	Kind              string `json:"kind"`
	SchemaVersion     int    `json:"schemaVersion"`
	Package           string `json:"package"`
	Version           string `json:"version"`
	Revision          string `json:"revision"`
	RegistryChecksum  string `json:"registryChecksum"`
	CandidateChecksum string `json:"candidateChecksum"`
	FileCount         int    `json:"fileCount"`
}

// Error reports a bounded artifact class and invariant without exposing local
// paths or archive contents.
type Error struct {
	Artifact  string
	Invariant string
}

func (e *Error) Error() string {
	return fmt.Sprintf("%s artifact %s", e.Artifact, e.Invariant)
}

type limits struct {
	maxArchiveBytes  int64
	maxExpandedBytes int64
	maxFileBytes     int64
	maxFiles         int
}

type crateArchive struct {
	checksum         string
	expandedChecksum [sha256.Size]byte
	files            map[string][]byte
}

type countingReader struct {
	reader io.Reader
	count  int64
}

func (r *countingReader) Read(buffer []byte) (int, error) {
	count, err := r.reader.Read(buffer)
	r.count += int64(count)
	return count, err
}

// Verify compares a reviewed local candidate with an already-accepted local
// crate artifact. It performs no network, registry, credential, or publication
// operation.
func Verify(options Options) (Report, error) {
	return verifyWithLimits(options, limits{
		maxArchiveBytes:  defaultMaxArchiveBytes,
		maxExpandedBytes: defaultMaxExpandedBytes,
		maxFileBytes:     defaultMaxFileBytes,
		maxFiles:         defaultMaxFiles,
	})
}

func verifyWithLimits(options Options, bounds limits) (Report, error) {
	if err := validateOptions(options); err != nil {
		return Report{}, err
	}
	inventory, err := readInventory(options.InventoryPath)
	if err != nil {
		return Report{}, err
	}
	candidateFile, candidateInfo, err := openArtifact("candidate", options.CandidatePath, bounds.maxArchiveBytes)
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = candidateFile.Close() }()
	acceptedFile, acceptedInfo, err := openArtifact("accepted", options.AcceptedPath, bounds.maxArchiveBytes)
	if err != nil {
		return Report{}, err
	}
	defer func() { _ = acceptedFile.Close() }()
	if os.SameFile(candidateInfo, acceptedInfo) {
		return Report{}, invariant("artifacts", "are distinct candidate and accepted files")
	}

	candidate, err := readCrate("candidate", candidateFile, candidateInfo.Size(), options, bounds)
	if err != nil {
		return Report{}, err
	}
	accepted, err := readCrate("accepted", acceptedFile, acceptedInfo.Size(), options, bounds)
	if err != nil {
		return Report{}, err
	}
	if accepted.checksum != options.RegistryChecksum {
		return Report{}, invariant("accepted", "matches the registry checksum")
	}
	if err := compareInventory("candidate", candidate.files, inventory); err != nil {
		return Report{}, err
	}
	if err := compareInventory("accepted", accepted.files, inventory); err != nil {
		return Report{}, err
	}
	if err := compareContents(candidate.files, accepted.files, inventory); err != nil {
		return Report{}, err
	}
	if candidate.expandedChecksum != accepted.expandedChecksum {
		return Report{}, invariant("candidate", "matches the accepted expanded tar stream")
	}
	for _, artifact := range []struct {
		name  string
		files map[string][]byte
	}{{"candidate", candidate.files}, {"accepted", accepted.files}} {
		if err := validateManifests(artifact.name, artifact.files, options.Package, options.Version); err != nil {
			return Report{}, err
		}
		if err := validateVCS(artifact.name, artifact.files[".cargo_vcs_info.json"], options.Revision, options.VCSPath); err != nil {
			return Report{}, err
		}
	}

	return Report{
		Kind:              reportKind,
		SchemaVersion:     1,
		Package:           options.Package,
		Version:           options.Version,
		Revision:          options.Revision,
		RegistryChecksum:  options.RegistryChecksum,
		CandidateChecksum: candidate.checksum,
		FileCount:         len(inventory),
	}, nil
}

func validateOptions(options Options) error {
	for _, input := range []struct {
		value     string
		artifact  string
		invariant string
	}{{options.CandidatePath, "candidate", "path is required"}, {options.AcceptedPath, "accepted", "path is required"}, {options.InventoryPath, "inventory", "path is required"}} {
		if strings.TrimSpace(input.value) == "" {
			return invariant(input.artifact, input.invariant)
		}
	}
	if !packageNamePattern.MatchString(options.Package) {
		return invariant("metadata", "has a valid package name")
	}
	if !versionPattern.MatchString(options.Version) {
		return invariant("metadata", "has a valid package version")
	}
	if !revisionPattern.MatchString(options.Revision) {
		return invariant("metadata", "has a full lowercase Git revision")
	}
	if !safeRelativePath(options.VCSPath) {
		return invariant("metadata", "has a safe VCS package path")
	}
	if !checksumPattern.MatchString(options.RegistryChecksum) {
		return invariant("metadata", "has a lowercase registry SHA-256")
	}
	return nil
}

func readInventory(filename string) ([]string, error) {
	pathInfo, err := os.Lstat(filename)
	if err != nil || !pathInfo.Mode().IsRegular() {
		return nil, invariant("inventory", "is a regular file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, invariant("inventory", "is readable")
	}
	defer func() { _ = file.Close() }()
	fileInfo, err := file.Stat()
	if err != nil || !fileInfo.Mode().IsRegular() || !os.SameFile(pathInfo, fileInfo) {
		return nil, invariant("inventory", "is a stable regular file")
	}
	content, err := io.ReadAll(io.LimitReader(file, maxInventoryBytes+1))
	if err != nil || len(content) > maxInventoryBytes {
		return nil, invariant("inventory", "fits the size limit")
	}
	if len(content) == 0 || content[len(content)-1] != '\n' || bytes.ContainsRune(content, '\r') {
		return nil, invariant("inventory", "uses canonical newline-delimited entries")
	}
	lines := strings.Split(strings.TrimSuffix(string(content), "\n"), "\n")
	if len(lines) == 0 || !slices.IsSorted(lines) {
		return nil, invariant("inventory", "is sorted")
	}
	for index, name := range lines {
		if !safeRelativePath(name) {
			return nil, invariant("inventory", "contains only safe relative paths")
		}
		if index > 0 && lines[index-1] == name {
			return nil, invariant("inventory", "contains no duplicate paths")
		}
	}
	return lines, nil
}

func openArtifact(artifact, filename string, maximum int64) (*os.File, os.FileInfo, error) {
	pathInfo, err := os.Lstat(filename)
	if err != nil || !pathInfo.Mode().IsRegular() || pathInfo.Size() > maximum {
		return nil, nil, invariant(artifact, "is a bounded regular file")
	}
	file, err := os.Open(filename)
	if err != nil {
		return nil, nil, invariant(artifact, "is readable")
	}
	fileInfo, err := file.Stat()
	if err != nil || !fileInfo.Mode().IsRegular() || fileInfo.Size() > maximum || !os.SameFile(pathInfo, fileInfo) {
		_ = file.Close()
		return nil, nil, invariant(artifact, "is a stable bounded regular file")
	}
	return file, fileInfo, nil
}

func readCrate(artifact string, file *os.File, size int64, options Options, bounds limits) (crateArchive, error) {
	compressedBytes, err := io.ReadAll(io.LimitReader(file, bounds.maxArchiveBytes+1))
	if err != nil || int64(len(compressedBytes)) != size || int64(len(compressedBytes)) > bounds.maxArchiveBytes {
		return crateArchive{}, invariant(artifact, "fits the archive size limit")
	}
	sum := sha256.Sum256(compressedBytes)
	checksum := hex.EncodeToString(sum[:])

	compressed := bufio.NewReader(bytes.NewReader(compressedBytes))
	reader, err := gzip.NewReader(compressed)
	if err != nil {
		return crateArchive{}, invariant(artifact, "is a valid gzip stream")
	}
	reader.Multistream(false)
	expandedBytes, err := io.ReadAll(io.LimitReader(reader, bounds.maxExpandedBytes+1))
	if err != nil || int64(len(expandedBytes)) > bounds.maxExpandedBytes {
		return crateArchive{}, invariant(artifact, "fits the expanded size limit")
	}
	if err := reader.Close(); err != nil {
		return crateArchive{}, invariant(artifact, "has a valid gzip checksum")
	}
	if _, err := compressed.Peek(1); !errors.Is(err, io.EOF) {
		return crateArchive{}, invariant(artifact, "has no trailing compressed data")
	}
	counted := &countingReader{reader: bytes.NewReader(expandedBytes)}
	archive := tar.NewReader(counted)
	files := make(map[string][]byte)
	root := options.Package + "-" + options.Version + "/"
	for {
		header, nextErr := archive.Next()
		if errors.Is(nextErr, io.EOF) {
			break
		}
		if nextErr != nil {
			return crateArchive{}, invariant(artifact, "contains a valid tar archive")
		}
		if len(files) >= bounds.maxFiles {
			return crateArchive{}, invariant(artifact, "fits the file-count limit")
		}
		if header.Typeflag != tar.TypeReg {
			return crateArchive{}, invariant(artifact, "contains regular files only")
		}
		if header.Size < 0 || header.Size > bounds.maxFileBytes {
			return crateArchive{}, invariant(artifact, "fits the per-file size limit")
		}
		if !safeArchivePath(header.Name, root) {
			return crateArchive{}, invariant(artifact, "contains only safe package-root paths")
		}
		name := strings.TrimPrefix(header.Name, root)
		if _, exists := files[name]; exists {
			return crateArchive{}, invariant(artifact, "contains no duplicate paths")
		}
		content, readErr := io.ReadAll(io.LimitReader(archive, header.Size+1))
		if readErr != nil || int64(len(content)) != header.Size {
			return crateArchive{}, invariant(artifact, "contains complete file bodies")
		}
		padding := (512 - header.Size%512) % 512
		paddingStart := counted.count
		paddingEnd := paddingStart + padding
		if paddingEnd > int64(len(expandedBytes)) || !allZero(expandedBytes[paddingStart:paddingEnd]) {
			return crateArchive{}, invariant(artifact, "has zero-filled file padding")
		}
		files[name] = content
	}
	if !canonicalTarTail(expandedBytes, counted.count) {
		return crateArchive{}, invariant(artifact, "has no hidden decompressed tar tail")
	}
	return crateArchive{checksum: checksum, expandedChecksum: sha256.Sum256(expandedBytes), files: files}, nil
}

func canonicalTarTail(content []byte, consumed int64) bool {
	if consumed < 1024 || consumed > int64(len(content)) {
		return false
	}
	terminatorStart := int(consumed) - 1024
	return allZero(content[terminatorStart:int(consumed)]) && allZero(content[int(consumed):])
}

func allZero(content []byte) bool {
	for _, value := range content {
		if value != 0 {
			return false
		}
	}
	return true
}

func safeArchivePath(name, root string) bool {
	return strings.HasPrefix(name, root) && safeRelativePath(strings.TrimPrefix(name, root))
}

func safeRelativePath(name string) bool {
	return name != "" && len(name) <= maxPathBytes && utf8.ValidString(name) &&
		!strings.ContainsAny(name, "\\:\x00") && !strings.HasPrefix(name, "/") &&
		!strings.Contains(name, "//") && path.Clean(name) == name && name != "." && name != ".." &&
		!strings.HasPrefix(name, "../")
}

func compareInventory(artifact string, files map[string][]byte, inventory []string) error {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	slices.Sort(names)
	if !slices.Equal(names, inventory) {
		return invariant(artifact, "matches the reviewed package inventory")
	}
	return nil
}

func compareContents(candidate, accepted map[string][]byte, inventory []string) error {
	for _, name := range inventory {
		if !bytes.Equal(candidate[name], accepted[name]) {
			return invariant("accepted", "matches candidate package contents")
		}
	}
	return nil
}

func validateManifests(artifact string, files map[string][]byte, expectedName, expectedVersion string) error {
	for _, name := range []string{"Cargo.toml", "Cargo.toml.orig"} {
		packageName, version, err := packageIdentity(files[name])
		if err != nil || packageName != expectedName || version != expectedVersion {
			return invariant(artifact, "has matching normalized and original manifest identity")
		}
	}
	return nil
}

func packageIdentity(content []byte) (string, string, error) {
	if len(content) == 0 || len(content) > maxInventoryBytes {
		return "", "", errors.New("manifest size")
	}
	scanner := bufio.NewScanner(bytes.NewReader(content))
	scanner.Buffer(make([]byte, 1024), maxInventoryBytes)
	inPackage := false
	var name, version string
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if strings.HasPrefix(line, "[") {
			inPackage = line == "[package]"
			continue
		}
		if !inPackage || strings.HasPrefix(line, "#") || line == "" {
			continue
		}
		key, raw, found := strings.Cut(line, "=")
		if !found {
			continue
		}
		value, err := strconv.Unquote(strings.TrimSpace(raw))
		if err != nil {
			continue
		}
		switch strings.TrimSpace(key) {
		case "name":
			if name != "" {
				return "", "", errors.New("duplicate name")
			}
			name = value
		case "version":
			if version != "" {
				return "", "", errors.New("duplicate version")
			}
			version = value
		}
	}
	if scanner.Err() != nil || name == "" || version == "" {
		return "", "", errors.New("missing identity")
	}
	return name, version, nil
}

func validateVCS(artifact string, content []byte, revision, vcsPath string) error {
	var metadata struct {
		Git struct {
			SHA1  string `json:"sha1"`
			Dirty *bool  `json:"dirty,omitempty"`
		} `json:"git"`
		PathInVCS string `json:"path_in_vcs"`
	}
	decoder := json.NewDecoder(bytes.NewReader(content))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&metadata); err != nil {
		return invariant(artifact, "has valid Cargo VCS metadata")
	}
	if decoder.Decode(&struct{}{}) != io.EOF || metadata.Git.SHA1 != revision || metadata.PathInVCS != vcsPath || (metadata.Git.Dirty != nil && *metadata.Git.Dirty) {
		return invariant(artifact, "matches the reviewed clean revision")
	}
	return nil
}

func invariant(artifact, rule string) error {
	return &Error{Artifact: artifact, Invariant: rule}
}
