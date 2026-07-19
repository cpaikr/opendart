package openapi

import (
	"archive/zip"
	"bytes"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"go.yaml.in/yaml/v4"
)

func TestDocumentFixturePreservesOpenAPIFeatures(t *testing.T) {
	root := fixturePath(t, "openapi.yaml")
	document, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	if document.Version() != "3.2.0" {
		t.Fatalf("Version() = %q", document.Version())
	}
	if err := document.ValidateDocument(); err != nil {
		t.Fatal(err)
	}
	renderedA, err := document.Render()
	if err != nil {
		t.Fatal(err)
	}
	renderedB, err := document.Render()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(renderedA, renderedB) {
		t.Fatal("rendering is not deterministic")
	}
	text := string(renderedA)
	for _, expected := range []string{"openapi: 3.2.0", "x-opendart:"} {
		if !strings.Contains(text, expected) {
			t.Errorf("rendered document does not contain %q", expected)
		}
	}

	bundleA, err := document.Bundle()
	if err != nil {
		t.Fatal(err)
	}
	bundleB, err := document.Bundle()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(bundleA, bundleB) {
		t.Fatal("bundling is not deterministic")
	}
	if strings.Contains(string(bundleA), "$ref: ./") || strings.Contains(string(bundleA), "$ref: ../") {
		t.Fatal("bundle retains an external local reference")
	}
	for _, expected := range []string{"x-opendart:", "nodeType: element", "name: result"} {
		if !strings.Contains(string(bundleA), expected) {
			t.Errorf("bundled document does not contain %q", expected)
		}
	}
	var bundleValue map[string]any
	if err := yaml.Unmarshal(bundleA, &bundleValue); err != nil {
		t.Fatal(err)
	}
	companySchema := nestedResolvedMap(t, bundleValue, bundleValue, "paths", "/company.xml", "get", "responses", "200", "content", "application/xml", "schema")
	xmlMetadata := nestedMap(t, companySchema, "xml")
	if xmlMetadata["nodeType"] != "element" || xmlMetadata["name"] != "result" {
		t.Fatalf("XML metadata = %#v", xmlMetadata)
	}
	extension := nestedMap(t, companySchema, "x-opendart")
	if extension["schemaStatus"] != "source-derived-unverified" || extension["sourceRootKey"] != "result" {
		t.Fatalf("x-opendart = %#v", extension)
	}
}

func TestSemanticComparisonIgnoresFormattingAndDetectsContractChanges(t *testing.T) {
	root := fixturePath(t, "openapi.yaml")
	temporaryRoot := copyFixture(t)
	formatted := filepath.Join(temporaryRoot, "openapi.yaml")
	source, err := os.ReadFile(formatted)
	if err != nil {
		t.Fatal(err)
	}
	source = bytes.Replace(source,
		[]byte("info:\n  title: OpenDART document fixture\n  version: 1.0.0"),
		[]byte("info:\n  version: 1.0.0\n  title: OpenDART document fixture"), 1)
	if err := os.WriteFile(formatted, source, 0o600); err != nil {
		t.Fatal(err)
	}
	comparison, err := Compare(root, formatted)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.TotalChanges != 0 {
		t.Fatalf("format-only comparison = %+v", comparison)
	}

	t.Run("XML metadata", func(t *testing.T) {
		changedRoot := copyFixture(t)
		replaceInFile(t, filepath.Join(changedRoot, "schemas", "company.yaml"), "nodeType: element", "nodeType: attribute")
		comparison, err := Compare(root, filepath.Join(changedRoot, "openapi.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		assertChangeLocation(t, comparison, "/xml/nodeType")
	})
	t.Run("extension", func(t *testing.T) {
		changedRoot := copyFixture(t)
		replaceInFile(t, filepath.Join(changedRoot, "schemas", "company.yaml"), "source-derived-unverified", "changed-status")
		comparison, err := Compare(root, filepath.Join(changedRoot, "openapi.yaml"))
		if err != nil {
			t.Fatal(err)
		}
		assertChangeLocation(t, comparison, "/x-opendart/schemaStatus")
	})
}

func TestSemanticComparisonTreatsSetValuedArraysAsUnordered(t *testing.T) {
	tests := []struct {
		name  string
		old   string
		left  string
		right string
	}{
		{name: "required", old: "required: [id]", left: "required: [id, name]", right: "required: [name, id]"},
		{name: "enum", old: "enum: [one]", left: "enum: [one, two]", right: "enum: [two, one]"},
		{name: "operation tags", old: "tags: [things]", left: "tags: [things, other]", right: "tags: [other, things]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			left := strings.Replace(strictLintFixture, test.old, test.left, 1)
			right := strings.Replace(strictLintFixture, test.old, test.right, 1)
			if left == strictLintFixture || right == strictLintFixture {
				t.Fatalf("fixture does not contain %q", test.old)
			}
			comparison := compareDocumentSources(t, left, right)
			if comparison.TotalChanges != 0 {
				t.Fatalf("reorder-only comparison = %+v", comparison)
			}
		})
	}
}

func TestSemanticComparisonPreservesOrderedArrays(t *testing.T) {
	old := "servers:\n  - url: https://api.invalid\n    description: Production"
	left := strings.Replace(strictLintFixture, old, old+"\n  - url: https://backup.invalid\n    description: Backup", 1)
	right := strings.Replace(strictLintFixture, old, "servers:\n  - url: https://backup.invalid\n    description: Backup\n  - url: https://api.invalid\n    description: Production", 1)
	comparison := compareDocumentSources(t, left, right)
	if comparison.TotalChanges == 0 {
		t.Fatal("server reorder was ignored")
	}
}

func TestSemanticComparisonPreservesExtensionArrayOrder(t *testing.T) {
	left := strictLintFixture + "x-ordering:\n  enum: [one, two]\n"
	right := strictLintFixture + "x-ordering:\n  enum: [two, one]\n"
	comparison := compareDocumentSources(t, left, right)
	if comparison.TotalChanges == 0 {
		t.Fatal("extension array reorder was ignored")
	}
}

func TestSemanticComparisonPreservesSchemaInstanceValueArrayOrder(t *testing.T) {
	for _, keyword := range []string{"example", "default", "const"} {
		t.Run(keyword, func(t *testing.T) {
			marker := "      required: [id]\n"
			left := strings.Replace(strictLintFixture, marker, "      "+keyword+":\n        enum: [one, two]\n"+marker, 1)
			right := strings.Replace(strictLintFixture, marker, "      "+keyword+":\n        enum: [two, one]\n"+marker, 1)
			if left == strictLintFixture || right == strictLintFixture {
				t.Fatalf("fixture does not contain %q", marker)
			}
			comparison := compareDocumentSources(t, left, right)
			if comparison.TotalChanges == 0 {
				t.Fatalf("%s array reorder was ignored", keyword)
			}
		})
	}
}

func TestSemanticComparisonNormalizesSchemaPropertyNamedLikeExtension(t *testing.T) {
	marker := "      properties:\n        id:"
	left := strings.Replace(strictLintFixture, marker, "      properties:\n        x-kind:\n          type: string\n          enum: [one, two]\n        id:", 1)
	right := strings.Replace(strictLintFixture, marker, "      properties:\n        x-kind:\n          type: string\n          enum: [two, one]\n        id:", 1)
	if left == strictLintFixture || right == strictLintFixture {
		t.Fatalf("fixture does not contain %q", marker)
	}
	comparison := compareDocumentSources(t, left, right)
	if comparison.TotalChanges != 0 {
		t.Fatalf("schema property enum reorder comparison = %+v", comparison)
	}
}

func TestSemanticComparisonNormalizesXPrefixedComponentEntries(t *testing.T) {
	tests := []struct {
		name  string
		left  map[string]any
		right map[string]any
	}{
		{
			name: "path item",
			left: map[string]any{"components": map[string]any{"pathItems": map[string]any{
				"x-shared": map[string]any{"get": map[string]any{"tags": []any{"one", "two"}}},
			}}},
			right: map[string]any{"components": map[string]any{"pathItems": map[string]any{
				"x-shared": map[string]any{"get": map[string]any{"tags": []any{"two", "one"}}},
			}}},
		},
		{
			name: "callback",
			left: map[string]any{"components": map[string]any{"callbacks": map[string]any{
				"x-shared": map[string]any{"{$request.body#/callbackUrl}": map[string]any{
					"post": map[string]any{"tags": []any{"one", "two"}},
				}},
			}}},
			right: map[string]any{"components": map[string]any{"callbacks": map[string]any{
				"x-shared": map[string]any{"{$request.body#/callbackUrl}": map[string]any{
					"post": map[string]any{"tags": []any{"two", "one"}},
				}},
			}}},
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			changes := semanticChanges(normalizeOpenAPIValue(test.left), normalizeOpenAPIValue(test.right), "#")
			if len(changes) != 0 {
				t.Fatalf("x-prefixed component entry reorder = %+v", changes)
			}
		})
	}
}

func compareDocumentSources(t *testing.T, left, right string) Comparison {
	t.Helper()
	directory := t.TempDir()
	leftRoot := filepath.Join(directory, "left.yaml")
	rightRoot := filepath.Join(directory, "right.yaml")
	if err := os.WriteFile(leftRoot, []byte(left), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(rightRoot, []byte(right), 0o600); err != nil {
		t.Fatal(err)
	}
	comparison, err := Compare(leftRoot, rightRoot)
	if err != nil {
		t.Fatal(err)
	}
	return comparison
}

func TestRepresentativeResponseValidation(t *testing.T) {
	document, err := Load(fixturePath(t, "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	tests := []struct {
		name        string
		path        string
		contentType string
		fixture     string
		wantError   bool
	}{
		{name: "valid JSON", path: "/company.json", contentType: "application/json", fixture: "company.valid.json"},
		{name: "JSON status pattern", path: "/company.json", contentType: "application/json", fixture: "company.invalid.json", wantError: true},
		{name: "JSON required property", path: "/company.json", contentType: "application/json", fixture: "company.missing.json", wantError: true},
		{name: "JSON additional property", path: "/company.json", contentType: "application/json", fixture: "company.additional.json", wantError: true},
		{name: "valid XML", path: "/company.xml", contentType: "application/xml", fixture: "company.valid.xml"},
		{name: "XML status pattern", path: "/company.xml", contentType: "application/xml", fixture: "company.invalid.xml", wantError: true},
		{name: "XML required property", path: "/company.xml", contentType: "application/xml", fixture: "company.missing.xml", wantError: true},
		{name: "XML additional property", path: "/company.xml", contentType: "application/xml", fixture: "company.additional.xml", wantError: true},
		{name: "valid XML API error", path: "/document.xml", contentType: "application/xml", fixture: "error.valid.xml"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			body, err := os.ReadFile(fixturePath(t, "responses", test.fixture))
			if err != nil {
				t.Fatal(err)
			}
			err = document.ValidateResponse("GET", test.path, test.contentType, 200, body)
			if (err != nil) != test.wantError {
				t.Fatalf("ValidateResponse() error = %v, wantError %v", err, test.wantError)
			}
			if err != nil && (strings.Contains(err.Error(), "not-a-status") || strings.Contains(err.Error(), "sentinel-property") || strings.Contains(err.Error(), "private")) {
				t.Fatalf("validation error exposed response data: %v", err)
			}
		})
	}
	wrongRoot := []byte("<sentinel-root><status>000</status><corp_name>company</corp_name></sentinel-root>")
	if err := document.ValidateResponse("GET", "/company.xml", "application/xml; charset=UTF-8", 200, wrongRoot); err == nil || !strings.Contains(err.Error(), "does not match") || strings.Contains(err.Error(), "sentinel-root") {
		t.Fatalf("wrong XML root error = %v", err)
	}
	if err := document.ValidateResponse("HEAD", "/company.xml", "application/xml", 200, wrongRoot); err == nil || !strings.Contains(err.Error(), "does not match") {
		t.Fatalf("HEAD-to-GET XML root error = %v", err)
	}
	validTemplatedXML := []byte("<result><status>000</status><corp_name>company</corp_name></result>")
	for _, method := range []string{"PUT", "PATCH"} {
		if err := document.ValidateResponse(method, "/companies/00126380.xml", "application/xml; charset=UTF-8", 200, validTemplatedXML); err != nil {
			t.Fatalf("valid templated %s XML: %v", method, err)
		}
	}

	validZIP := zipFixture(t, "document.xml", []byte("<document><rcept_no>1</rcept_no></document>"))
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, validZIP); err != nil {
		t.Fatalf("valid ZIP: %v", err)
	}
	if err := document.ValidateResponse("HEAD", "/document.xml", "application/zip", 200, validZIP); err != nil {
		t.Fatalf("HEAD-to-GET ZIP: %v", err)
	}
	xbrlZIP := zipFixture(t, "report.xbrl", []byte("<xbrl/>"))
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip; charset=binary", 200, xbrlZIP); err != nil {
		t.Fatalf("valid XBRL ZIP with parameterized media type: %v", err)
	}
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip; charset=binary", 200, []byte("not a zip")); err == nil {
		t.Fatal("parameterized ZIP media type bypassed archive validation")
	}
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, []byte("not a zip")); err == nil {
		t.Fatal("corrupt ZIP passed validation")
	}
	unsafeZIP := zipFixture(t, "../document.xml", []byte("<document/>"))
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, unsafeZIP); err == nil {
		t.Fatal("unsafe ZIP entry passed validation")
	}
	windowsUnsafeZIP := zipFixture(t, `..\sentinel-document.xml`, []byte("<document/>"))
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, windowsUnsafeZIP); err == nil || strings.Contains(err.Error(), "sentinel-document") {
		t.Fatalf("Windows-style unsafe ZIP error = %v", err)
	}
	noXMLZIP := zipFixture(t, "readme.txt", []byte("not XML"))
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, noXMLZIP); err == nil {
		t.Fatal("ZIP without XML passed validation")
	}
	malformedXMLZIP := zipFixture(t, "document.xml", []byte("<document>"))
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, malformedXMLZIP); err == nil {
		t.Fatal("ZIP with malformed XML passed validation")
	}
	malformedEncoding := append([]byte(`<document>`), 0xff)
	malformedEncoding = append(malformedEncoding, []byte(`</document>`)...)
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, zipFixture(t, "document.xml", malformedEncoding)); err == nil {
		t.Fatal("ZIP with replacement-rune XML fallback passed validation")
	}
	manyEntries := make(map[string][]byte)
	for index := 0; index <= maxArchiveEntries; index++ {
		manyEntries[fmt.Sprintf("%d.xml", index)] = []byte("<document/>")
	}
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, zipEntries(t, manyEntries)); err == nil {
		t.Fatal("ZIP entry limit was not enforced")
	}
	oversizedZIP := zipFixture(t, "document.xml", bytes.Repeat([]byte(" "), maxArchiveBytes+1))
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, oversizedZIP); err == nil {
		t.Fatal("ZIP expansion limit was not enforced")
	}
}

func TestLocalReferencePolicyRejectsRemoteAndEscapingReferences(t *testing.T) {
	for _, reference := range []string{
		"https://example.com/path.yaml",
		"/absolute/path.yaml",
		"../outside.yaml",
	} {
		t.Run(reference, func(t *testing.T) {
			directory := t.TempDir()
			root := filepath.Join(directory, "openapi.yaml")
			source := "openapi: 3.2.0\ninfo:\n  title: unsafe\n  version: 1.0.0\npaths:\n  /unsafe:\n    $ref: " + reference + "\n"
			if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := Load(root); err == nil {
				t.Fatalf("reference %q passed validation", reference)
			}
		})
	}
}

func TestLocalReferencePolicyRejectsPhysicalEscape(t *testing.T) {
	directory := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.yaml")
	if err := os.WriteFile(outside, []byte("get: {}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	linked := filepath.Join(directory, "linked.yaml")
	if err := os.Symlink(outside, linked); err != nil {
		t.Fatal(err)
	}
	root := filepath.Join(directory, "openapi.yaml")
	source := "openapi: 3.2.0\ninfo:\n  title: unsafe\n  version: 1.0.0\npaths:\n  /unsafe:\n    $ref: ./linked.yaml\n"
	if err := os.WriteFile(root, []byte(source), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(root); err == nil || !strings.Contains(err.Error(), "resolves outside") {
		t.Fatalf("Load() error = %v", err)
	}
}

func TestLoadRejectsUnresolvedReferenceFragment(t *testing.T) {
	rootDirectory := copyFixture(t)
	pathFile := filepath.Join(rootDirectory, "paths", "company.json.yaml")
	source, err := os.ReadFile(pathFile)
	if err != nil {
		t.Fatal(err)
	}
	source = bytes.Replace(source,
		[]byte("../schemas/company.yaml"),
		[]byte("../schemas/company.yaml#/properties/missing"), 1)
	if err := os.WriteFile(pathFile, source, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Load(filepath.Join(rootDirectory, "openapi.yaml")); err == nil {
		t.Fatal("unresolved reference fragment passed validation")
	}
}

func fixturePath(t *testing.T, elements ...string) string {
	t.Helper()
	parts := append([]string{"testdata", "document"}, elements...)
	return filepath.Join(parts...)
}

func copyFixture(t *testing.T) string {
	t.Helper()
	sourceRoot := fixturePath(t)
	targetRoot := t.TempDir()
	err := filepath.WalkDir(sourceRoot, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		relative, err := filepath.Rel(sourceRoot, path)
		if err != nil {
			return err
		}
		target := filepath.Join(targetRoot, relative)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o700)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o600)
	})
	if err != nil {
		t.Fatal(err)
	}
	return targetRoot
}

func zipFixture(t *testing.T, name string, data []byte) []byte {
	t.Helper()
	return zipEntries(t, map[string][]byte{name: data})
}

func zipEntries(t *testing.T, entries map[string][]byte) []byte {
	t.Helper()
	var buffer bytes.Buffer
	writer := zip.NewWriter(&buffer)
	names := make([]string, 0, len(entries))
	for name := range entries {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		entry, err := writer.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := entry.Write(entries[name]); err != nil {
			t.Fatal(err)
		}
	}
	if err := writer.Close(); err != nil {
		t.Fatal(err)
	}
	return buffer.Bytes()
}

func replaceInFile(t *testing.T, path, old, replacement string) {
	t.Helper()
	source, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	changed := bytes.Replace(source, []byte(old), []byte(replacement), 1)
	if bytes.Equal(source, changed) {
		t.Fatalf("%q not found in %s", old, path)
	}
	if err := os.WriteFile(path, changed, 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertChangeLocation(t *testing.T, comparison Comparison, suffix string) {
	t.Helper()
	for _, detail := range comparison.Details {
		if strings.HasSuffix(detail.Location, suffix) {
			return
		}
	}
	t.Fatalf("comparison does not contain location suffix %q: %+v", suffix, comparison)
}

func nestedMap(t *testing.T, root map[string]any, keys ...string) map[string]any {
	t.Helper()
	current := root
	for _, key := range keys {
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("%q in %v is %T, want map", key, keys, current[key])
		}
		current = next
	}
	return current
}

func nestedResolvedMap(t *testing.T, document, root map[string]any, keys ...string) map[string]any {
	t.Helper()
	current := root
	for _, key := range keys {
		current = resolveInternalMap(t, document, current)
		next, ok := current[key].(map[string]any)
		if !ok {
			t.Fatalf("%q in %v is %T, want map", key, keys, current[key])
		}
		current = next
	}
	return resolveInternalMap(t, document, current)
}

func resolveInternalMap(t *testing.T, document, value map[string]any) map[string]any {
	t.Helper()
	for range 32 {
		reference, ok := value["$ref"].(string)
		if !ok {
			return value
		}
		if !strings.HasPrefix(reference, "#/") {
			t.Fatalf("reference %q is not internal", reference)
		}
		current := document
		for _, segment := range strings.Split(strings.TrimPrefix(reference, "#/"), "/") {
			segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
			next, ok := current[segment].(map[string]any)
			if !ok {
				t.Fatalf("reference %q segment %q is %T, want map", reference, segment, current[segment])
			}
			current = next
		}
		value = current
	}
	t.Fatal("internal reference chain exceeds limit")
	return nil
}
