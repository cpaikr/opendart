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

func TestCompatibilityFixturePreservesOpenAPIFeatures(t *testing.T) {
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
	companySchema := nestedMap(t, bundleValue, "paths", "/company.xml", "get", "responses", "200", "content", "application/xml", "schema")
	xmlMetadata := nestedMap(t, companySchema, "xml")
	if xmlMetadata["nodeType"] != "element" || xmlMetadata["name"] != "result" {
		t.Fatalf("XML metadata = %#v", xmlMetadata)
	}
	extension := nestedMap(t, companySchema, "x-opendart")
	if extension["schemaStatus"] != "source-derived-unverified" || extension["sourceRootKey"] != "result" {
		t.Fatalf("x-opendart = %#v", extension)
	}
}

func TestRepresentativeLintFixtureIsRejected(t *testing.T) {
	root := filepath.Join("testdata", "lint", "missing-response-description.yaml")
	document, err := Load(root)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	if err := document.LintCompatibility(); err == nil {
		t.Fatal("invalid lint fixture passed document validation")
	} else if !strings.Contains(err.Error(), root) {
		t.Fatalf("lint diagnostic does not identify artifact: %v", err)
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
		[]byte("info:\n  title: OpenDART compatibility fixture\n  version: 1.0.0"),
		[]byte("info:\n  version: 1.0.0\n  title: OpenDART compatibility fixture"), 1)
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

func TestBundleBaselineComparisonKeepsReferencedAndDeclaredSchemas(t *testing.T) {
	root := fixturePath(t, "openapi.yaml")
	bundlePath := generatedComponentBundle(t, false)
	comparison, err := compareBundleBaseline(root, bundlePath)
	if err != nil {
		t.Fatal(err)
	}
	if comparison.TotalChanges != 0 {
		t.Fatalf("equivalent generated-component bundle = %+v", comparison)
	}

	mutatedBundlePath := generatedComponentBundle(t, true)
	comparison, err = compareBundleBaseline(root, mutatedBundlePath)
	if err != nil {
		t.Fatal(err)
	}
	assertChangeLocation(t, comparison, "/paths/~1company.json/get/responses/200/content/application~1json/schema/properties/corp_name/minLength")

	operationMutation := generatedComponentBundle(t, false)
	replaceInFile(t, operationMutation, "get_company_json_fixture", "get_company_json_changed")
	comparison, err = compareBundleBaseline(root, operationMutation)
	if err != nil {
		t.Fatal(err)
	}
	assertChangeLocation(t, comparison, "/paths/~1company.json/get/operationId")

	sharedMutation := generatedComponentBundle(t, false)
	replaceInFile(t, sharedMutation, "schemaStatus: empirically-observed", "schemaStatus: changed-status")
	comparison, err = compareBundleBaseline(root, sharedMutation)
	if err != nil {
		t.Fatal(err)
	}
	assertChangeLocation(t, comparison, "/components/schemas/OpenDartError/x-opendart/schemaStatus")
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
	manyEntries := make(map[string][]byte)
	for index := 0; index <= maxCompatibilityArchiveEntries; index++ {
		manyEntries[fmt.Sprintf("%d.xml", index)] = []byte("<document/>")
	}
	if err := document.ValidateResponse("GET", "/document.xml", "application/zip", 200, zipEntries(t, manyEntries)); err == nil {
		t.Fatal("ZIP entry limit was not enforced")
	}
	oversizedZIP := zipFixture(t, "document.xml", bytes.Repeat([]byte(" "), maxCompatibilityArchiveBytes+1))
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
	parts := append([]string{"testdata", "compatibility"}, elements...)
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

func generatedComponentBundle(t *testing.T, mutate bool) string {
	t.Helper()
	document, err := Load(fixturePath(t, "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	bundled, err := document.bundle(true)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := yaml.Unmarshal(bundled, &value); err != nil {
		t.Fatal(err)
	}
	content := nestedMap(t, value, "paths", "/company.json", "get", "responses", "200", "content", "application/json")
	schema := nestedMap(t, content, "schema")
	if mutate {
		corpName := nestedMap(t, schema, "properties", "corp_name")
		corpName["minLength"] = 2
	}
	components := nestedMap(t, value, "components")
	schemas := nestedMap(t, components, "schemas")
	schemas["GeneratedCompany"] = schema
	content["schema"] = map[string]any{"$ref": "#/components/schemas/GeneratedCompany"}
	output, err := yaml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(t.TempDir(), "openapi.bundle.yaml")
	if err := os.WriteFile(path, output, 0o600); err != nil {
		t.Fatal(err)
	}
	return path
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
