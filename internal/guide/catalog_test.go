package guide

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestValidateCatalogAcceptedTree(t *testing.T) {
	root := filepath.Join("..", "..", "openapi", "openapi.yaml")
	report, err := ValidateCatalog(CatalogOptions{Root: root})
	if err != nil {
		t.Fatal(err)
	}
	wantGroups := make(map[string]int, len(Groups))
	for _, group := range Groups {
		wantGroups[group.Code] = group.ExpectedCount
	}
	if report.OpenAPI != catalogOpenAPIVersion ||
		report.LogicalEndpoints != ExpectedFullTotals.LogicalEndpoints ||
		report.PhysicalPaths != ExpectedFullTotals.PhysicalPaths ||
		report.RequestArguments != ExpectedFullTotals.RequestArguments ||
		report.ResponseFields != ExpectedFullTotals.ResponseFields ||
		report.MessageCodes != ExpectedFullTotals.MessageCodes ||
		!reflect.DeepEqual(report.GroupCounts, wantGroups) {
		t.Fatalf("report = %#v", report)
	}
	if !filepath.IsAbs(report.Root) {
		t.Fatalf("report root = %q, want absolute path", report.Root)
	}

	structural, err := ValidateCatalog(CatalogOptions{Root: root, StructuralOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(report, structural) {
		t.Fatalf("structural report = %#v, want %#v", structural, report)
	}
}

func TestCatalogDiagnosticIsStructuredAndBounded(t *testing.T) {
	_, err := ValidateCatalog(CatalogOptions{})
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) {
		t.Fatalf("error = %v, want CatalogError", err)
	}
	if catalogErr.Diagnostic.Rule != "catalog-root" || catalogErr.Diagnostic.Phase != "options" || catalogErr.Diagnostic.Message == "" {
		t.Fatalf("diagnostic = %#v", catalogErr.Diagnostic)
	}
	if strings.Contains(catalogErr.Error(), "map[") {
		t.Fatalf("diagnostic embeds unbounded context: %v", catalogErr)
	}
}

func TestValidateCatalogRejectsMutationFamilies(t *testing.T) {
	tests := []struct {
		name string
		rule string
		edit func(*testing.T, string)
	}{
		{name: "ownership marker", rule: "output-marker", edit: func(t *testing.T, root string) {
			writeCatalogTestFile(t, filepath.Join(filepath.Dir(root), OutputMarker), "changed\n")
		}},
		{name: "root contract", rule: "openapi-version", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, root, "openapi: 3.2.0", "openapi: 3.1.0")
		}},
		{name: "strict YAML", rule: "yaml", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, root, "openapi: 3.2.0", "openapi: 3.2.0\nopenapi: 3.2.0")
		}},
		{name: "path fragment shape", rule: "path-fragment-method", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml"), "get:\n", "post:\n")
		}},
		{name: "operation provenance", rule: "source-date", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml"), "checkedAt: 2026-07-17", "checkedAt: 2026-07-16")
		}},
		{name: "documented API prefix", rule: "documented-path", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml"), "/api/company.json", "/xxx/company.json")
		}},
		{name: "parameter normalization", rule: "parameter-normalization", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml"), "schema:\n        type: string", "schema:\n        type: array")
		}},
		{name: "response schema mapping", rule: "response-schema", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml"), "../../schemas/ds001/2019002.yaml", "../../schemas/ds001/2019001.yaml")
		}},
		{name: "ZIP raw schema", rule: "zip-schema", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "document.xml.yaml"), "          schema: {}\n", "")
		}},
		{name: "reference table presence", rule: "reference-tables", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "ds001", "company.json.yaml"), "    referenceTables: []\n", "")
		}},
		{name: "response source order", rule: "response-source-order", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "schemas", "ds001", "2019002.yaml"), "sourceIndex: 0", "sourceIndex: 9")
		}},
		{name: "artifact closure", rule: "path-file-closure", edit: func(t *testing.T, root string) {
			writeCatalogTestFile(t, filepath.Join(filepath.Dir(root), "paths", "orphan.yaml"), "get: {}\n")
		}},
		{name: "reference confinement", rule: "reference-escape", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, root, "./paths/ds001/company.json.yaml", "../go.mod")
		}},
		{name: "physical reference confinement", rule: "reference-physical-escape", edit: func(t *testing.T, root string) {
			outside := filepath.Join(t.TempDir(), "outside.yaml")
			writeCatalogTestFile(t, outside, "get: {}\n")
			linked := filepath.Join(filepath.Dir(root), "paths", "escape.yaml")
			if err := os.Symlink(outside, linked); err != nil {
				t.Fatal(err)
			}
			replaceCatalogTestFile(t, root, "./paths/ds001/company.json.yaml", "./paths/escape.yaml")
		}},
		{name: "shared schema policy", rule: "xml-error-schema", edit: func(t *testing.T, root string) {
			replaceCatalogTestFile(t, filepath.Join(filepath.Dir(root), "components", "schemas.yaml"), "schemaStatus: empirically-observed", "schemaStatus: changed")
		}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			root := copyCatalogTree(t)
			test.edit(t, root)
			_, err := ValidateCatalog(CatalogOptions{Root: root})
			var catalogErr *CatalogError
			if !errors.As(err, &catalogErr) {
				t.Fatalf("error = %v, want CatalogError", err)
			}
			if catalogErr.Diagnostic.Rule != test.rule {
				t.Fatalf("rule = %q, want %q; error = %v", catalogErr.Diagnostic.Rule, test.rule, err)
			}
			if catalogErr.Diagnostic.Artifact == "" || catalogErr.Diagnostic.Phase == "" {
				t.Fatalf("diagnostic lacks artifact or phase: %#v", catalogErr.Diagnostic)
			}
		})
	}
}

func TestValidateCatalogRejectsMarkerSymlinkBeforeReadingTarget(t *testing.T) {
	root := copyCatalogTree(t)
	marker := filepath.Join(filepath.Dir(root), OutputMarker)
	if err := os.Remove(marker); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(filepath.Dir(root), marker); err != nil {
		t.Fatal(err)
	}

	_, err := ValidateCatalog(CatalogOptions{Root: root})
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Diagnostic.Rule != "output-marker" ||
		catalogErr.Diagnostic.Message != "generated-output marker must be a regular non-symlink file" {
		t.Fatalf("error = %v", err)
	}
}

func TestStructuralCatalogAcceptsConsistentPartialInventory(t *testing.T) {
	root := copyCatalogTree(t)
	removeCatalogEndpoint(t, root)

	report, err := ValidateCatalog(CatalogOptions{Root: root, StructuralOnly: true})
	if err != nil {
		t.Fatal(err)
	}
	if report.LogicalEndpoints != ExpectedFullTotals.LogicalEndpoints-1 || report.PhysicalPaths != ExpectedFullTotals.PhysicalPaths-2 {
		t.Fatalf("partial report = %#v", report)
	}
	_, err = ValidateCatalog(CatalogOptions{Root: root})
	var catalogErr *CatalogError
	if !errors.As(err, &catalogErr) || catalogErr.Diagnostic.Rule != "physical-path-completeness" {
		t.Fatalf("full validation error = %v", err)
	}
}

func copyCatalogTree(t *testing.T) string {
	t.Helper()
	source := filepath.Join("..", "..", "openapi")
	target := filepath.Join(t.TempDir(), "openapi")
	err := filepath.WalkDir(source, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(source, path)
		if err != nil {
			return err
		}
		destination := filepath.Join(target, relative)
		if entry.IsDir() {
			return os.MkdirAll(destination, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(destination, data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(target, "openapi.yaml")
}

func replaceCatalogTestFile(t *testing.T, file, old, replacement string) {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	changed := strings.Replace(string(data), old, replacement, 1)
	if changed == string(data) {
		t.Fatalf("%q not found in %s", old, file)
	}
	if err := os.WriteFile(file, []byte(changed), 0o600); err != nil {
		t.Fatal(err)
	}
}

func writeCatalogTestFile(t *testing.T, file, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(file), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

func removeCatalogEndpoint(t *testing.T, root string) {
	t.Helper()
	directory := filepath.Dir(root)
	document := readGeneratedMap(t, root)
	paths := yamlMap(t, document["paths"])
	delete(paths, "/dvRs.json")
	delete(paths, "/dvRs.xml")
	metadata := yamlMap(t, document["x-opendart"])
	inventory := yamlMap(t, metadata["inventory"])
	inventory["logicalEndpointCount"] = ExpectedFullTotals.LogicalEndpoints - 1
	inventory["physicalPathCount"] = ExpectedFullTotals.PhysicalPaths - 2
	groups := yamlMap(t, inventory["groupCounts"])
	groups["DS006"] = 5
	writeCatalogYAML(t, root, document)
	for _, file := range []string{
		filepath.Join(directory, "paths", "ds006", "dvRs.json.yaml"),
		filepath.Join(directory, "paths", "ds006", "dvRs.xml.yaml"),
		filepath.Join(directory, "schemas", "ds006", "2020059.yaml"),
	} {
		if err := os.Remove(file); err != nil {
			t.Fatal(err)
		}
	}
}

func writeCatalogYAML(t *testing.T, path string, value any) {
	t.Helper()
	encoded, err := yaml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, encoded, 0o600); err != nil {
		t.Fatal(err)
	}
}
