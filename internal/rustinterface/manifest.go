// Package rustinterface decodes the reviewed Rust and CLI product interface
// manifests without assigning them wire or runtime authority.
package rustinterface

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pelletier/go-toml/v2"
)

const (
	SchemaVersion   = 1
	maxManifestSize = 1 << 20
)

// Accessors contains the shared handwritten response vocabulary.
type Accessors struct {
	Source  string `toml:"source"`
	Status  string `toml:"status"`
	Message string `toml:"message"`
	Items   string `toml:"items"`
	Field   string `toml:"field"`
}

// ReadAll loads the six fixed DS-family interface inputs in canonical order.
func ReadAll(directory string) ([]Batch, error) {
	batches := make([]Batch, 0, 6)
	for number := 1; number <= 6; number++ {
		name := fmt.Sprintf("ds%03d.toml", number)
		batch, err := ReadBatch(filepath.Join(directory, name))
		if err != nil {
			return nil, fmt.Errorf("read Rust interface %s: %w", name, err)
		}
		batches = append(batches, batch)
	}
	return batches, nil
}

// Batch contains one reviewed DS-family product interface.
type Batch struct {
	SchemaVersion int         `toml:"schema_version"`
	Family        string      `toml:"family"`
	Accessors     Accessors   `toml:"accessors"`
	Operations    []Operation `toml:"operation"`
}

// Operation binds one canonical logical identity to independently reviewed
// Rust and CLI product names.
type Operation struct {
	LogicalID     string         `toml:"logical_id"`
	RustModule    string         `toml:"rust_module"`
	RustInput     string         `toml:"rust_input"`
	CLICommand    string         `toml:"cli_command"`
	CLIAlias      string         `toml:"cli_alias"`
	Physical      []Physical     `toml:"physical"`
	Parameters    []Parameter    `toml:"parameter"`
	ResponseViews []ResponseView `toml:"response_view"`
}

// ResponseView contains reviewed handwritten response names for one source
// schema location. The CLI projection does not consume these Rust names.
type ResponseView struct {
	Path      string            `toml:"path"`
	RustType  string            `toml:"rust_type"`
	Accessors map[string]string `toml:"accessors"`
}

// Physical binds a canonical physical identity to handwritten Rust symbols.
// Only the CLI's private dispatch projection consumes these names.
type Physical struct {
	OperationID  string `toml:"operation_id"`
	RustMethod   string `toml:"rust_method"`
	RustResponse string `toml:"rust_response"`
}

// Parameter binds one canonical OpenAPI parameter to reviewed Rust and CLI
// product names.
type Parameter struct {
	OpenAPIName string `toml:"openapi_name"`
	RustName    string `toml:"rust_name"`
	CLIFlag     string `toml:"cli_flag"`
}

// ReadBatch strictly decodes one bounded, regular interface manifest.
func ReadBatch(path string) (Batch, error) {
	info, err := os.Lstat(path)
	if err != nil {
		return Batch{}, err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxManifestSize {
		return Batch{}, errors.New("manifest must be a bounded regular file")
	}
	// #nosec G304 -- callers supply fixed paths below the repository root.
	body, err := os.ReadFile(path)
	if err != nil {
		return Batch{}, err
	}
	var batch Batch
	decoder := toml.NewDecoder(bytes.NewReader(body))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(&batch); err != nil {
		return Batch{}, err
	}
	return batch, nil
}
