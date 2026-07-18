package guide

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"

	openapitooling "github.com/cpaikr/opendart/internal/openapi"
	"go.yaml.in/yaml/v4"
)

func TestGenerateDeterministicGuideContract(t *testing.T) {
	endpoints := generationFixture(t)
	left := t.TempDir()
	right := t.TempDir()

	wantResult := GenerationResult{PhysicalPaths: 3, SchemaFiles: 2}
	for _, output := range []string{left, right} {
		result, err := Generate(endpoints, GenerateOptions{OutputDir: output, CheckedAt: "2026-07-17"})
		if err != nil {
			t.Fatal(err)
		}
		if result != wantResult {
			t.Fatalf("result = %#v, want %#v", result, wantResult)
		}
	}

	leftFiles := generatedFileContents(t, left)
	rightFiles := generatedFileContents(t, right)
	if !reflect.DeepEqual(leftFiles, rightFiles) {
		t.Fatal("generation differs between identical runs")
	}
	wantFiles := []string{
		".opendart-spec-output",
		"components/schemas.yaml",
		"openapi.yaml",
		"paths/ds001/document.xml.yaml",
		"paths/ds003/fnlttMultiAcnt.json.yaml",
		"paths/ds003/fnlttMultiAcnt.xml.yaml",
		"schemas/ds001/2019003.yaml",
		"schemas/ds003/2019017.yaml",
	}
	gotFiles := make([]string, 0, len(leftFiles))
	for name := range leftFiles {
		gotFiles = append(gotFiles, name)
	}
	sort.Strings(gotFiles)
	if !reflect.DeepEqual(gotFiles, wantFiles) {
		t.Fatalf("files = %#v, want %#v", gotFiles, wantFiles)
	}
	if string(leftFiles[OutputMarker]) != OutputMarkerContent {
		t.Fatalf("marker = %q", leftFiles[OutputMarker])
	}
	document, err := openapitooling.Load(filepath.Join(left, "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	lintDiagnostics, err := document.Lint()
	if err != nil {
		t.Fatal(err)
	}
	if len(lintDiagnostics) != 0 {
		t.Fatalf("generated document lint diagnostics = %#v", lintDiagnostics)
	}

	root := readGeneratedMap(t, filepath.Join(left, "openapi.yaml"))
	if root["openapi"] != "3.2.0" {
		t.Fatalf("openapi = %#v", root["openapi"])
	}
	paths := yamlMap(t, root["paths"])
	if len(paths) != 3 {
		t.Fatalf("paths = %#v", paths)
	}
	components := yamlMap(t, yamlMap(t, root["components"])["schemas"])
	zipSchema := yamlMap(t, components["DS001_2019003_Response"])
	if zipSchema["$ref"] != "./schemas/ds001/2019003.yaml" {
		t.Fatalf("ZIP response component = %#v", zipSchema)
	}

	multi := readGeneratedMap(t, filepath.Join(left, "paths/ds003/fnlttMultiAcnt.json.yaml"))
	operation := yamlMap(t, multi["get"])
	parameters := yamlSlice(t, operation["parameters"])
	if len(parameters) != 3 {
		t.Fatalf("parameters = %#v", parameters)
	}
	corpCode := yamlMap(t, parameters[0])
	if corpCode["style"] != "form" || corpCode["explode"] != false {
		t.Fatalf("corp_code serialization = %#v", corpCode)
	}
	corpSchema := yamlMap(t, corpCode["schema"])
	if corpSchema["type"] != "array" || corpSchema["maxItems"] != 100 {
		t.Fatalf("corp_code schema = %#v", corpSchema)
	}
	serialization := yamlMap(t, corpCode["x-opendart-serialization"])
	if serialization["wireFormat"] != "comma-separated" {
		t.Fatalf("serialization provenance = %#v", serialization)
	}
	if _, ok := corpCode["x-opendart-source-diagnostics"]; !ok {
		t.Fatal("multi-company source diagnostic is missing")
	}
	reportCode := yamlMap(t, parameters[2])
	if reportCode["name"] != "reprt_code" || reportCode["required"] != true {
		t.Fatalf("reprt_code parameter = %#v", reportCode)
	}

	responseSchema := readGeneratedMap(t, filepath.Join(left, "schemas/ds003/2019017.yaml"))
	xOpenDART := yamlMap(t, responseSchema["x-opendart"])
	if !reflect.DeepEqual(xOpenDART["responseFields"], yamlRoundTrip(t, endpoints[0].ResponseFields)) {
		t.Fatalf("raw response metadata changed: %#v", xOpenDART["responseFields"])
	}
	diagnostics := yamlSlice(t, xOpenDART["diagnostics"])
	if len(diagnostics) != 1 || yamlMap(t, diagnostics[0])["code"] != "list-child-shares-container-depth" {
		t.Fatalf("hierarchy diagnostics = %#v", diagnostics)
	}
	list := yamlMap(t, yamlMap(t, responseSchema["properties"])["list"])
	if list["type"] != "array" || list["x-opendart-normalization"] != "source-derived-unverified" {
		t.Fatalf("normalized list = %#v", list)
	}

	zipPath := readGeneratedMap(t, filepath.Join(left, "paths/ds001/document.xml.yaml"))
	zipDefault := yamlMap(t, yamlMap(t, yamlMap(t, zipPath["get"])["responses"])["default"])
	zipContent := yamlMap(t, zipDefault["content"])
	if _, ok := zipContent["application/zip"]; !ok {
		t.Fatal("ZIP success representation is missing")
	}
	xmlError := yamlMap(t, zipContent["application/xml"])
	if xmlError["x-opendart-content-type-status"] != "empirically-observed-error-response" {
		t.Fatalf("ZIP XML error representation = %#v", xmlError)
	}
}

func TestGenerateRejectsUnsafeOrConflictingInput(t *testing.T) {
	base := generationFixture(t)
	tests := []struct {
		name string
		edit func([]Endpoint)
		want string
	}{
		{
			name: "invalid identity",
			edit: func(endpoints []Endpoint) { endpoints[0].LogicalOperationID = "DS003-evil" },
			want: "invalid endpoint identity",
		},
		{
			name: "method",
			edit: func(endpoints []Endpoint) { endpoints[0].BasicInfo[0].Method = "POST" },
			want: "unexpected method",
		},
		{
			name: "encoding",
			edit: func(endpoints []Endpoint) { endpoints[0].BasicInfo[0].Encoding = "EUC-KR" },
			want: "unexpected method",
		},
		{
			name: "origin",
			edit: func(endpoints []Endpoint) {
				endpoints[0].BasicInfo[0].RequestURL = "https://example.com/api/fnlttMultiAcnt.json"
			},
			want: "outside the documented API server",
		},
		{
			name: "encoded path separator",
			edit: func(endpoints []Endpoint) {
				endpoints[0].BasicInfo[0].RequestURL = "https://opendart.fss.or.kr/api/fnltt%2FMultiAcnt.json"
			},
			want: "outside the documented API server",
		},
		{
			name: "dot path segment",
			edit: func(endpoints []Endpoint) {
				endpoints[0].BasicInfo[0].RequestURL = "https://opendart.fss.or.kr/api/nested/../fnlttMultiAcnt.json"
			},
			want: "outside the documented API server",
		},
		{
			name: "duplicate physical path",
			edit: func(endpoints []Endpoint) {
				endpoints[0].BasicInfo[1].RequestURL = endpoints[0].BasicInfo[0].RequestURL
			},
			want: "duplicate physical API path",
		},
		{
			name: "message tables",
			edit: func(endpoints []Endpoint) { endpoints[1].MessageCodes[0].Description = "changed" },
			want: "message-code tables differ",
		},
		{
			name: "request requiredness",
			edit: func(endpoints []Endpoint) { endpoints[0].RequestArguments[0].Required = "maybe" },
			want: "unknown requiredness",
		},
		{
			name: "multi-company example",
			edit: func(endpoints []Endpoint) { endpoints[0].GuideTestRequestArguments[0].Value = "00334624" },
			want: "multi-company guide example is missing or malformed",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			endpoints := cloneEndpoints(t, base)
			test.edit(endpoints)
			_, err := Generate(endpoints, GenerateOptions{OutputDir: t.TempDir(), CheckedAt: "2026-07-17"})
			if err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("error = %v, want substring %q", err, test.want)
			}
		})
	}
}

func TestGenerateRejectsAdditionalGuideQueryParameters(t *testing.T) {
	endpoints := generationFixture(t)
	endpoints[0].SourceURL += "&view=full"
	if _, err := Generate(endpoints, GenerateOptions{OutputDir: t.TempDir(), CheckedAt: "2026-07-17"}); err == nil || !strings.Contains(err.Error(), "guide source identity") {
		t.Fatalf("additional query parameter error = %v", err)
	}
}

func TestGenerateRequiresEmptyPhysicalDirectory(t *testing.T) {
	endpoints := generationFixture(t)
	nonempty := t.TempDir()
	if err := os.WriteFile(filepath.Join(nonempty, "owned-by-user"), []byte("keep"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(endpoints, GenerateOptions{OutputDir: nonempty, CheckedAt: "2026-07-17"}); err == nil || !strings.Contains(err.Error(), "must be empty") {
		t.Fatalf("nonempty output error = %v", err)
	}

	symlink := filepath.Join(t.TempDir(), "stage-link")
	if err := os.Symlink(t.TempDir(), symlink); err != nil {
		t.Fatal(err)
	}
	if _, err := Generate(endpoints, GenerateOptions{OutputDir: symlink, CheckedAt: "2026-07-17"}); err == nil || !strings.Contains(err.Error(), "physical directory") {
		t.Fatalf("symlink output error = %v", err)
	}
}

func generationFixture(t *testing.T) []Endpoint {
	t.Helper()
	data, err := os.ReadFile("testdata/generate/endpoints.yaml")
	if err != nil {
		t.Fatal(err)
	}
	var endpoints []Endpoint
	if err := yaml.Unmarshal(data, &endpoints); err != nil {
		t.Fatal(err)
	}
	if len(endpoints) != 2 || endpoints[0].LogicalOperationID == "" {
		t.Fatalf("invalid generation fixture: %#v", endpoints)
	}
	return endpoints
}

func cloneEndpoints(t *testing.T, endpoints []Endpoint) []Endpoint {
	t.Helper()
	data, err := yaml.Marshal(endpoints)
	if err != nil {
		t.Fatal(err)
	}
	var clone []Endpoint
	if err := yaml.Unmarshal(data, &clone); err != nil {
		t.Fatal(err)
	}
	return clone
}

func generatedFileContents(t *testing.T, root string) map[string][]byte {
	t.Helper()
	files := make(map[string][]byte)
	err := filepath.WalkDir(root, func(file string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() {
			return nil
		}
		data, err := os.ReadFile(file)
		if err != nil {
			return err
		}
		name, err := filepath.Rel(root, file)
		if err != nil {
			return err
		}
		files[filepath.ToSlash(name)] = data
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return files
}

func readGeneratedMap(t *testing.T, file string) map[string]any {
	t.Helper()
	data, err := os.ReadFile(file)
	if err != nil {
		t.Fatal(err)
	}
	var value map[string]any
	if err := yaml.Unmarshal(data, &value); err != nil {
		t.Fatal(err)
	}
	return value
}

func yamlMap(t *testing.T, value any) map[string]any {
	t.Helper()
	result, ok := value.(map[string]any)
	if !ok {
		t.Fatalf("value %#v is not a YAML mapping", value)
	}
	return result
}

func yamlSlice(t *testing.T, value any) []any {
	t.Helper()
	result, ok := value.([]any)
	if !ok {
		t.Fatalf("value %#v is not a YAML sequence", value)
	}
	return result
}

func yamlRoundTrip(t *testing.T, value any) any {
	t.Helper()
	encoded, err := yaml.Marshal(value)
	if err != nil {
		t.Fatal(err)
	}
	var result any
	if err := yaml.Unmarshal(encoded, &result); err != nil {
		t.Fatal(err)
	}
	return result
}

func TestGeneratedYAMLHasNoDocumentSeparatorOrTrailingSpaces(t *testing.T) {
	output := t.TempDir()
	if _, err := Generate(generationFixture(t), GenerateOptions{OutputDir: output, CheckedAt: "2026-07-17"}); err != nil {
		t.Fatal(err)
	}
	for name, content := range generatedFileContents(t, output) {
		if name == OutputMarker {
			continue
		}
		if bytes.HasPrefix(content, []byte("---")) {
			t.Fatalf("%s has a YAML document separator", name)
		}
		for _, line := range bytes.Split(content, []byte("\n")) {
			if len(line) > 0 && (line[len(line)-1] == ' ' || line[len(line)-1] == '\t') {
				t.Fatalf("%s has trailing whitespace", name)
			}
		}
	}
}
