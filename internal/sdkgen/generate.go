// Package sdkgen owns deterministic CLI projection generation and freshness checks.
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
	"github.com/cpaikr/opendart/internal/rustinterface"
	"github.com/cpaikr/opendart/internal/sdkgen/model"
	"github.com/cpaikr/opendart/internal/sdkgen/ownership"
	rustemitter "github.com/cpaikr/opendart/internal/sdkgen/rust"
)

var (
	ErrGeneratedMissing    = errors.New("generated CLI projection is missing")
	ErrGeneratedStale      = errors.New("generated CLI projection is stale")
	ErrGeneratedUnexpected = errors.New("generated CLI projection contains an unexpected file")
	ErrGeneratedUnowned    = errors.New("generated CLI projection ownership marker is invalid")
	ErrRustInterfaceInput  = errors.New("Rust interface input is invalid")
)

// ArtifactError identifies which owned projection failed freshness validation.
type ArtifactError struct {
	Kind   string
	Output string
	cause  error
}

func (e *ArtifactError) Error() string {
	return fmt.Sprintf("check generated Rust %s: %v", e.Kind, e.cause)
}

func (e *ArtifactError) Unwrap() error {
	return e.cause
}

// Inputs identifies every input to CLI projection generation.
type Inputs struct {
	OpenAPI   string
	Interface string
}

// ArtifactReport identifies one generated projection and its published tree.
type ArtifactReport struct {
	Kind          string `json:"kind"`
	SchemaVersion uint32 `json:"schemaVersion"`
	Checksum      string `json:"checksum"`
	Output        string `json:"output"`
}

// Report describes both independently identified CLI projections.
type Report struct {
	Artifacts []ArtifactReport `json:"artifacts"`
}

type ownedProduct struct {
	kind          string
	markerName    string
	markerPrefix  string
	schemaVersion uint32
	checksum      string
	output        string
	files         map[string][]byte
}

type stagedProduct struct {
	ownedProduct
	staging      string
	backup       string
	existed      bool
	stagingOwned bool
	backupOwned  bool
	published    bool
}

// GenerateCLI renders, validates, and publishes both CLI projection trees.
// All trees pass ownership preflight before any accepted tree is replaced.
func GenerateCLI(inputs Inputs, output string) (Report, error) {
	generated, files, err := renderCLI(inputs)
	if err != nil {
		return Report{}, err
	}
	return generateCLIArtifacts(generated, files, output)
}

func generateCLIArtifacts(generated model.ArtifactSet, files rustemitter.Artifacts, output string) (Report, error) {
	products := cliProducts(generated, files, output)
	if err := validateProductOutputs(products); err != nil {
		return Report{}, err
	}
	if err := prepareCLIOutputRoot(output, true); err != nil {
		return Report{}, err
	}
	staged, err := stageProducts(products)
	if err != nil {
		return Report{}, err
	}
	defer cleanupProducts(staged)
	if err := publishProducts(staged); err != nil {
		return Report{}, err
	}

	report := Report{Artifacts: make([]ArtifactReport, 0, len(staged))}
	for _, product := range staged {
		report.Artifacts = append(report.Artifacts, ArtifactReport{
			Kind: product.kind, SchemaVersion: product.schemaVersion,
			Checksum: product.checksum, Output: product.output,
		})
	}
	return report, nil
}

// CheckCLIFresh renders both projections in memory and compares their complete
// owned subtrees without rewriting the working tree.
func CheckCLIFresh(inputs Inputs, output string) error {
	generated, files, err := renderCLI(inputs)
	if err != nil {
		return err
	}
	return checkCLIArtifactsFresh(generated, files, output)
}

func checkCLIArtifactsFresh(generated model.ArtifactSet, files rustemitter.Artifacts, output string) error {
	products := cliProducts(generated, files, output)
	if err := validateProductOutputs(products); err != nil {
		return err
	}
	if err := prepareCLIOutputRoot(output, false); err != nil {
		return err
	}
	for _, product := range products {
		actual, err := readOwnedTree(product.output, product, false)
		if err != nil {
			return &ArtifactError{Kind: product.kind, Output: product.output, cause: err}
		}
		if err := compareTrees(product.files, actual, product.markerName); err != nil {
			return &ArtifactError{Kind: product.kind, Output: product.output, cause: err}
		}
	}
	return nil
}

func renderCLI(inputs Inputs) (model.ArtifactSet, rustemitter.Artifacts, error) {
	if strings.TrimSpace(inputs.OpenAPI) == "" {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, errors.New("CLI projection OpenAPI input is required")
	}
	if strings.TrimSpace(inputs.Interface) == "" {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, errors.New("Rust interface input directory is required")
	}
	document, err := openapispec.Load(inputs.OpenAPI)
	if err != nil {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, fmt.Errorf("load CLI projection OpenAPI input: %w", err)
	}
	defer document.Close()
	if err := document.ValidateDocument(); err != nil {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, fmt.Errorf("validate CLI projection OpenAPI input: %w", err)
	}
	surface, err := document.InspectSDKSurface()
	if err != nil {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, fmt.Errorf("inspect CLI projection OpenAPI input: %w", err)
	}
	batches, err := rustinterface.ReadAll(inputs.Interface)
	if err != nil {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, fmt.Errorf("%w: %w", ErrRustInterfaceInput, err)
	}
	generated, err := model.BuildArtifacts(surface, batches)
	if err != nil {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, err
	}
	files, err := rustemitter.RenderArtifacts(generated)
	if err != nil {
		return model.ArtifactSet{}, rustemitter.Artifacts{}, err
	}
	return generated, files, nil
}

func cliProducts(generated model.ArtifactSet, files rustemitter.Artifacts, output string) []ownedProduct {
	return []ownedProduct{
		{kind: "cli-interface", markerName: ownership.CLIInterfaceFilename, markerPrefix: ownership.CLIInterfaceMarkerPrefix, schemaVersion: generated.CLIInterface.SchemaVersion, checksum: generated.CLIInterface.Checksum, output: filepath.Join(output, "interface"), files: files.CLIInterface},
		{kind: "cli-dispatch", markerName: ownership.CLIDispatchFilename, markerPrefix: ownership.CLIDispatchMarkerPrefix, schemaVersion: generated.CLIDispatch.SchemaVersion, checksum: generated.CLIDispatch.Checksum, output: filepath.Join(output, "dispatch"), files: files.CLIDispatch},
	}
}

func prepareCLIOutputRoot(root string, create bool) error {
	if strings.TrimSpace(root) == "" {
		return &ArtifactError{Kind: "cli", Output: root, cause: errors.New("generated Rust CLI output is required")}
	}
	absolute, err := filepath.Abs(root)
	if err != nil {
		return &ArtifactError{Kind: "cli", Output: root, cause: err}
	}
	info, err := os.Lstat(absolute)
	if errors.Is(err, os.ErrNotExist) && create {
		if err := os.MkdirAll(absolute, 0o755); err != nil {
			return &ArtifactError{Kind: "cli", Output: absolute, cause: err}
		}
		return nil
	}
	if errors.Is(err, os.ErrNotExist) {
		return &ArtifactError{Kind: "cli", Output: absolute, cause: ErrGeneratedMissing}
	}
	if err != nil {
		return &ArtifactError{Kind: "cli", Output: absolute, cause: err}
	}
	if !info.IsDir() || info.Mode()&os.ModeSymlink != 0 {
		return &ArtifactError{Kind: "cli", Output: absolute, cause: ErrGeneratedUnowned}
	}
	entries, err := os.ReadDir(absolute)
	if err != nil {
		return &ArtifactError{Kind: "cli", Output: absolute, cause: err}
	}
	for _, entry := range entries {
		if entry.Name() != "interface" && entry.Name() != "dispatch" {
			return &ArtifactError{Kind: "cli", Output: absolute, cause: fmt.Errorf("%w: %s", ErrGeneratedUnexpected, entry.Name())}
		}
	}
	return nil
}

func stageProducts(products []ownedProduct) ([]*stagedProduct, error) {
	if err := validateProductOutputs(products); err != nil {
		return nil, err
	}
	staged := make([]*stagedProduct, 0, len(products))
	for _, product := range products {
		absolute, err := filepath.Abs(product.output)
		if err != nil {
			cleanupProducts(staged)
			return nil, fmt.Errorf("resolve generated Rust %s output: %w", product.kind, err)
		}
		product.output = absolute
		entry := &stagedProduct{ownedProduct: product}
		staged = append(staged, entry)
		parent := filepath.Dir(product.output)
		parentInfo, err := os.Stat(parent)
		if err != nil {
			cleanupProducts(staged)
			return nil, fmt.Errorf("stage generated Rust %s: output parent is unavailable: %w", product.kind, err)
		}
		if !parentInfo.IsDir() {
			cleanupProducts(staged)
			return nil, fmt.Errorf("stage generated Rust %s: output parent is not a directory", product.kind)
		}
		entry.staging, err = os.MkdirTemp(parent, ".opendart-"+product.kind+"-stage-")
		if err != nil {
			cleanupProducts(staged)
			return nil, fmt.Errorf("stage generated Rust %s: %w", product.kind, err)
		}
		entry.stagingOwned = true
		if err := writeTree(entry.staging, product); err != nil {
			cleanupProducts(staged)
			return nil, err
		}
		written, err := readOwnedTree(entry.staging, product, false)
		if err != nil || !equalTrees(product.files, written) {
			cleanupProducts(staged)
			if err != nil {
				return nil, fmt.Errorf("validate staged Rust %s: %w", product.kind, err)
			}
			return nil, fmt.Errorf("validate staged Rust %s: rendered bytes changed during staging", product.kind)
		}
		_, statErr := os.Lstat(product.output)
		switch {
		case errors.Is(statErr, os.ErrNotExist):
		case statErr != nil:
			cleanupProducts(staged)
			return nil, fmt.Errorf("inspect generated Rust %s: %w", product.kind, statErr)
		default:
			entry.existed = true
			if _, err := readOwnedTree(product.output, product, true); err != nil {
				cleanupProducts(staged)
				return nil, fmt.Errorf("replace generated Rust %s: %w", product.kind, err)
			}
		}
	}
	return staged, nil
}

func publishProducts(products []*stagedProduct) error {
	for _, product := range products {
		if !product.existed {
			continue
		}
		backup, err := os.MkdirTemp(filepath.Dir(product.output), ".opendart-"+product.kind+"-backup-")
		if err != nil {
			return rollbackProducts(products, fmt.Errorf("reserve Rust %s rollback path: %w", product.kind, err))
		}
		if err := os.Remove(backup); err != nil {
			return rollbackProducts(products, fmt.Errorf("prepare Rust %s rollback path: %w", product.kind, err))
		}
		product.backup = backup
		if err := os.Rename(product.output, product.backup); err != nil {
			return rollbackProducts(products, fmt.Errorf("preserve generated Rust %s: %w", product.kind, err))
		}
		product.backupOwned = true
	}
	for _, product := range products {
		if err := os.Rename(product.staging, product.output); err != nil {
			return rollbackProducts(products, fmt.Errorf("publish generated Rust %s: %w", product.kind, err))
		}
		product.stagingOwned = false
		product.published = true
	}
	for _, product := range products {
		if product.backupOwned {
			if err := os.RemoveAll(product.backup); err != nil {
				return fmt.Errorf("remove generated Rust %s rollback copy: %w", product.kind, err)
			}
			product.backupOwned = false
		}
	}
	return nil
}

func rollbackProducts(products []*stagedProduct, cause error) error {
	var rollbackErrors []error
	for index := len(products) - 1; index >= 0; index-- {
		product := products[index]
		if product.published {
			if err := os.RemoveAll(product.output); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("remove published Rust %s: %w", product.kind, err))
				continue
			}
			product.published = false
		}
		if product.backupOwned {
			if err := os.Rename(product.backup, product.output); err != nil {
				rollbackErrors = append(rollbackErrors, fmt.Errorf("restore Rust %s: %w", product.kind, err))
			} else {
				product.backupOwned = false
			}
		}
	}
	if len(rollbackErrors) == 0 {
		return cause
	}
	return publishRollbackError(cause, errors.Join(rollbackErrors...))
}

func publishRollbackError(publishErr, rollbackErr error) error {
	return errors.Join(fmt.Errorf("publish failed: %w", publishErr), fmt.Errorf("rollback failed: %w", rollbackErr))
}

func cleanupProducts(products []*stagedProduct) {
	for _, product := range products {
		if product.stagingOwned {
			_ = os.RemoveAll(product.staging)
		}
	}
}

func validateProductOutputs(products []ownedProduct) error {
	for _, product := range products {
		if strings.TrimSpace(product.output) == "" {
			return fmt.Errorf("generated Rust %s output is required", product.kind)
		}
	}
	for left := 0; left < len(products); left++ {
		leftPath, _ := filepath.Abs(products[left].output)
		for right := left + 1; right < len(products); right++ {
			rightPath, _ := filepath.Abs(products[right].output)
			if pathsOverlap(leftPath, rightPath) {
				return fmt.Errorf("generated Rust outputs must be distinct non-nested directories")
			}
		}
	}
	return nil
}

func pathsOverlap(left, right string) bool {
	leftToRight, leftErr := filepath.Rel(left, right)
	rightToLeft, rightErr := filepath.Rel(right, left)
	return left == right || (leftErr == nil && leftToRight != ".." && !strings.HasPrefix(leftToRight, ".."+string(filepath.Separator))) ||
		(rightErr == nil && rightToLeft != ".." && !strings.HasPrefix(rightToLeft, ".."+string(filepath.Separator)))
}

func writeTree(root string, product ownedProduct) error {
	for _, name := range sortedNames(product.files) {
		if !validRelativeName(name) {
			return fmt.Errorf("write generated Rust %s: invalid owned path %q", product.kind, name)
		}
		target := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return fmt.Errorf("write generated Rust %s directory %q: %w", product.kind, name, err)
		}
		if err := os.WriteFile(target, product.files[name], 0o644); err != nil {
			return fmt.Errorf("write generated Rust %s file %q: %w", product.kind, name, err)
		}
	}
	return nil
}

func readOwnedTree(root string, product ownedProduct, acceptPreviousSchema bool) (map[string][]byte, error) {
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
	marker, exists := files[product.markerName]
	if !exists || !validOwnershipMarker(marker, product, acceptPreviousSchema) {
		return nil, ErrGeneratedUnowned
	}
	return files, nil
}

func validOwnershipMarker(marker []byte, product ownedProduct, acceptPreviousSchema bool) bool {
	expected := product.markerPrefix + strconv.FormatUint(uint64(product.schemaVersion), 10) + "\n"
	if string(marker) == expected {
		return true
	}
	if !acceptPreviousSchema {
		return false
	}
	value, found := strings.CutPrefix(string(marker), product.markerPrefix)
	if !found || !strings.HasSuffix(value, "\n") {
		return false
	}
	parsed, err := strconv.ParseUint(strings.TrimSuffix(value, "\n"), 10, 32)
	return err == nil && parsed > 0 && parsed < uint64(product.schemaVersion)
}

func compareTrees(expected, actual map[string][]byte, markerName string) error {
	for _, name := range sortedNames(expected) {
		content, exists := actual[name]
		if !exists {
			return fmt.Errorf("%w: %s", ErrGeneratedMissing, name)
		}
		if !bytes.Equal(content, expected[name]) {
			if name == markerName {
				return fmt.Errorf("%w: %s", ErrGeneratedUnowned, name)
			}
			return fmt.Errorf("%w: %s", ErrGeneratedStale, name)
		}
	}
	for _, name := range sortedNames(actual) {
		if _, exists := expected[name]; !exists {
			return fmt.Errorf("%w: %s", ErrGeneratedUnexpected, name)
		}
	}
	return nil
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
