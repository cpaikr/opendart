// Package sdkgen owns deterministic SDK generation and freshness checks.
package sdkgen

import (
	"bytes"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/sdkgen/model"
	"github.com/cpaikr/opendart/internal/sdkgen/ownership"
	rustemitter "github.com/cpaikr/opendart/internal/sdkgen/rust"
)

var (
	ErrGeneratedMissing    = errors.New("generated Rust SDK is missing")
	ErrGeneratedStale      = errors.New("generated Rust SDK is stale")
	ErrGeneratedUnexpected = errors.New("generated Rust SDK contains an unexpected file")
	ErrGeneratedUnowned    = errors.New("generated Rust SDK ownership marker is invalid")
)

type Report struct {
	Language      string `json:"language"`
	SchemaVersion uint32 `json:"schemaVersion"`
	Checksum      string `json:"checksum"`
	Output        string `json:"output"`
}

// GenerateRust renders and atomically publishes the complete owned subtree.
func GenerateRust(root, output string) (Report, error) {
	generated, files, err := renderRust(root)
	if err != nil {
		return Report{}, err
	}
	absoluteOutput, err := filepath.Abs(output)
	if err != nil {
		return Report{}, fmt.Errorf("resolve Rust SDK output: %w", err)
	}
	if err := publishOwnedTree(absoluteOutput, files); err != nil {
		return Report{}, err
	}
	return Report{Language: "rust", SchemaVersion: generated.SchemaVersion, Checksum: generated.Checksum, Output: absoluteOutput}, nil
}

// CheckRustFresh renders in memory and compares the complete owned subtree
// without rewriting the working tree.
func CheckRustFresh(root, output string) error {
	_, expected, err := renderRust(root)
	if err != nil {
		return err
	}
	actual, err := readOwnedTree(output)
	if err != nil {
		return err
	}
	expectedNames := sortedNames(expected)
	actualNames := sortedNames(actual)
	for _, name := range expectedNames {
		content, exists := actual[name]
		if !exists {
			return fmt.Errorf("%w: %s", ErrGeneratedMissing, name)
		}
		if !bytes.Equal(content, expected[name]) {
			if name == ownership.Filename {
				return fmt.Errorf("%w: %s", ErrGeneratedUnowned, name)
			}
			return fmt.Errorf("%w: %s", ErrGeneratedStale, name)
		}
	}
	for _, name := range actualNames {
		if _, exists := expected[name]; !exists {
			return fmt.Errorf("%w: %s", ErrGeneratedUnexpected, name)
		}
	}
	return nil
}

func renderRust(root string) (model.Model, map[string][]byte, error) {
	document, err := openapispec.Load(root)
	if err != nil {
		return model.Model{}, nil, fmt.Errorf("load SDK OpenAPI input: %w", err)
	}
	defer document.Close()
	if err := document.ValidateDocument(); err != nil {
		return model.Model{}, nil, fmt.Errorf("validate SDK OpenAPI input: %w", err)
	}
	surface, err := document.InspectSDKSurface()
	if err != nil {
		return model.Model{}, nil, fmt.Errorf("inspect SDK OpenAPI input: %w", err)
	}
	generated, err := model.Build(surface)
	if err != nil {
		return model.Model{}, nil, err
	}
	files, err := rustemitter.Render(generated)
	if err != nil {
		return model.Model{}, nil, err
	}
	return generated, files, nil
}

func publishOwnedTree(output string, files map[string][]byte) error {
	parent := filepath.Dir(output)
	parentInfo, err := os.Stat(parent)
	if err != nil || !parentInfo.IsDir() {
		return fmt.Errorf("publish generated Rust SDK: output parent is unavailable: %w", err)
	}
	staging, err := os.MkdirTemp(parent, ".opendart-sdk-stage-")
	if err != nil {
		return fmt.Errorf("publish generated Rust SDK: create staging directory: %w", err)
	}
	stagingOwned := true
	defer func() {
		if stagingOwned {
			_ = os.RemoveAll(staging)
		}
	}()
	if err := writeTree(staging, files); err != nil {
		return err
	}
	staged, err := readOwnedTree(staging)
	if err != nil {
		return fmt.Errorf("validate staged Rust SDK: %w", err)
	}
	if !equalTrees(files, staged) {
		return errors.New("validate staged Rust SDK: rendered bytes changed during staging")
	}

	_, statErr := os.Lstat(output)
	if errors.Is(statErr, os.ErrNotExist) {
		if err := os.Rename(staging, output); err != nil {
			return fmt.Errorf("publish generated Rust SDK: %w", err)
		}
		stagingOwned = false
		return nil
	}
	if statErr != nil {
		return fmt.Errorf("inspect existing generated Rust SDK: %w", statErr)
	}
	if _, err := readReplaceableOwnedTree(output); err != nil {
		return fmt.Errorf("replace generated Rust SDK: %w", err)
	}
	backup, err := os.MkdirTemp(parent, ".opendart-sdk-backup-")
	if err != nil {
		return fmt.Errorf("replace generated Rust SDK: reserve rollback path: %w", err)
	}
	if err := os.Remove(backup); err != nil {
		return fmt.Errorf("replace generated Rust SDK: prepare rollback path: %w", err)
	}
	if err := os.Rename(output, backup); err != nil {
		return fmt.Errorf("replace generated Rust SDK: preserve accepted output: %w", err)
	}
	if err := os.Rename(staging, output); err != nil {
		if rollbackErr := os.Rename(backup, output); rollbackErr != nil {
			return publishRollbackError(err, rollbackErr)
		}
		return fmt.Errorf("replace generated Rust SDK: publish failed and accepted output was restored: %w", err)
	}
	stagingOwned = false
	if err := os.RemoveAll(backup); err != nil {
		return fmt.Errorf("replace generated Rust SDK: remove rollback copy: %w", err)
	}
	return nil
}

func publishRollbackError(publishErr, rollbackErr error) error {
	return fmt.Errorf(
		"replace generated Rust SDK: %w",
		errors.Join(
			fmt.Errorf("publish failed: %w", publishErr),
			fmt.Errorf("rollback failed: %w", rollbackErr),
		),
	)
}

func writeTree(root string, files map[string][]byte) error {
	for _, name := range sortedNames(files) {
		if !validRelativeName(name) {
			return fmt.Errorf("write generated Rust SDK: invalid owned path %q", name)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("write generated Rust SDK directory %q: %w", name, err)
		}
		if err := os.WriteFile(target, files[name], 0o644); err != nil {
			return fmt.Errorf("write generated Rust SDK file %q: %w", name, err)
		}
	}
	return nil
}

func readOwnedTree(root string) (map[string][]byte, error) {
	return readOwnedTreeWithMarker(root, false)
}

func readReplaceableOwnedTree(root string) (map[string][]byte, error) {
	return readOwnedTreeWithMarker(root, true)
}

func readOwnedTreeWithMarker(root string, acceptPreviousSchema bool) (map[string][]byte, error) {
	info, err := os.Lstat(root)
	if errors.Is(err, os.ErrNotExist) {
		return nil, ErrGeneratedMissing
	}
	if err != nil {
		return nil, err
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return nil, ErrGeneratedUnowned
	}
	files := make(map[string][]byte)
	err = filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if path == root {
			return nil
		}
		if entry.Type()&os.ModeSymlink != 0 {
			return ErrGeneratedUnowned
		}
		if entry.IsDir() {
			return nil
		}
		if !entry.Type().IsRegular() {
			return ErrGeneratedUnowned
		}
		relative, err := filepath.Rel(root, path)
		if err != nil {
			return err
		}
		name := filepath.ToSlash(relative)
		if !validRelativeName(name) {
			return ErrGeneratedUnowned
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		files[name] = content
		return nil
	})
	if err != nil {
		return nil, err
	}
	marker, exists := files[ownership.Filename]
	if !exists || (string(marker) != ownership.Marker(model.SchemaVersion) && !(acceptPreviousSchema && validPreviousOwnershipMarker(marker))) {
		return nil, ErrGeneratedUnowned
	}
	return files, nil
}

func validPreviousOwnershipMarker(marker []byte) bool {
	value, found := strings.CutPrefix(string(marker), ownership.MarkerPrefix)
	if !found || !strings.HasSuffix(value, "\n") {
		return false
	}
	value = strings.TrimSuffix(value, "\n")
	parsed, err := strconv.ParseUint(value, 10, 32)
	return err == nil && parsed > 0 && parsed < uint64(model.SchemaVersion)
}

func validRelativeName(name string) bool {
	return name != "" && !strings.HasPrefix(name, "/") && !strings.Contains(name, "\\") &&
		name != "." && name != ".." && !strings.HasPrefix(name, "../") && !strings.Contains(name, "/../")
}

func sortedNames(files map[string][]byte) []string {
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func equalTrees(left, right map[string][]byte) bool {
	if len(left) != len(right) {
		return false
	}
	for name, content := range left {
		if !bytes.Equal(content, right[name]) {
			return false
		}
	}
	return true
}
