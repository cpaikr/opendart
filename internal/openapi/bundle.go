package openapi

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

var (
	// ErrBundleMissing identifies a freshness failure caused by an absent artifact.
	ErrBundleMissing = errors.New("committed OpenAPI bundle is missing")
	// ErrBundleStale identifies an artifact whose bytes differ from a fresh bundle.
	ErrBundleStale = errors.New("committed OpenAPI bundle is stale")
)

// GenerateBundle renders a portable bundle and verifies that repeated builds
// from the same source produce identical bytes.
func GenerateBundle(root string) ([]byte, error) {
	document, err := Load(root)
	if err != nil {
		return nil, err
	}
	defer document.Close()

	first, err := document.Bundle()
	if err != nil {
		return nil, err
	}
	second, err := document.Bundle()
	if err != nil {
		return nil, err
	}
	if !bytes.Equal(first, second) {
		return nil, errors.New("OpenAPI bundling is nondeterministic")
	}
	return first, nil
}

// WriteBundle generates the bundle completely before atomically replacing the
// explicitly selected output file.
func WriteBundle(root, output string) error {
	if strings.TrimSpace(output) == "" {
		return errors.New("bundle output path is required")
	}
	bundle, err := GenerateBundle(root)
	if err != nil {
		return err
	}
	if err := writeFileAtomically(output, bundle, os.Rename); err != nil {
		return fmt.Errorf("write OpenAPI bundle %s: %w", output, err)
	}
	return nil
}

// CheckBundleFresh requires the committed artifact to exactly match a fresh,
// deterministic bundle.
func CheckBundleFresh(root, committed string) error {
	if strings.TrimSpace(committed) == "" {
		return errors.New("committed bundle path is required")
	}
	generated, err := GenerateBundle(root)
	if err != nil {
		return err
	}
	accepted, err := os.ReadFile(committed)
	if err != nil {
		if errors.Is(err, os.ErrNotExist) {
			return fmt.Errorf("%w: %s", ErrBundleMissing, committed)
		}
		return fmt.Errorf("read committed OpenAPI bundle %s: %w", committed, err)
	}
	if !bytes.Equal(accepted, generated) {
		return fmt.Errorf("%w: %s", ErrBundleStale, committed)
	}
	return nil
}

func writeFileAtomically(output string, content []byte, rename func(string, string) error) error {
	directory := filepath.Dir(output)
	if err := os.MkdirAll(directory, 0o755); err != nil {
		return fmt.Errorf("create output directory: %w", err)
	}
	temporary, err := os.CreateTemp(directory, "."+filepath.Base(output)+".tmp-")
	if err != nil {
		return fmt.Errorf("create temporary bundle: %w", err)
	}
	temporaryPath := temporary.Name()
	defer func() {
		_ = os.Remove(temporaryPath)
	}()

	if err := temporary.Chmod(0o644); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("set temporary bundle permissions: %w", err)
	}
	if _, err := temporary.Write(content); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("write temporary bundle: %w", err)
	}
	if err := temporary.Sync(); err != nil {
		_ = temporary.Close()
		return fmt.Errorf("sync temporary bundle: %w", err)
	}
	if err := temporary.Close(); err != nil {
		return fmt.Errorf("close temporary bundle: %w", err)
	}
	if err := rename(temporaryPath, output); err != nil {
		return fmt.Errorf("replace output: %w", err)
	}
	return nil
}
