package rustconformance

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

func TestCheckAcceptsReviewedRepositoryContract(t *testing.T) {
	report, err := Check(repositoryRoot(t))
	if err != nil {
		t.Fatalf("%v: %v", err, errors.Unwrap(err))
	}
	want := Report{Batches: 6, LogicalOperations: 85, PhysicalOperations: 167, ResponseViews: 171, ResponseAccessors: 1900, ExecutableCases: 3}
	if report != want {
		t.Fatalf("report = %#v, want %#v", report, want)
	}
}

func TestOpenAPIResponseSchemasFitReviewedViewShape(t *testing.T) {
	sources, err := loadSourceOperations(repositoryRoot(t))
	if err != nil {
		t.Fatal(err)
	}
	logicalSchemas := make(map[string]openapispec.SDKSurfaceSchema)
	viewCount := 0
	for _, source := range sources {
		if source.responseSchema == nil {
			continue
		}
		if prior, ok := logicalSchemas[source.logicalID]; ok {
			if !reflect.DeepEqual(prior, *source.responseSchema) {
				t.Fatalf("logical operation %s has representation-specific schema drift", source.logicalID)
			}
			continue
		}
		views, err := responseViewContracts(*source.responseSchema)
		if err != nil {
			t.Fatalf("logical operation %s: %v", source.logicalID, err)
		}
		logicalSchemas[source.logicalID] = *source.responseSchema
		viewCount += len(views)
	}
	if len(logicalSchemas) != 82 || viewCount != 171 {
		t.Fatalf("structured logical operations/views = %d/%d, want 82/171", len(logicalSchemas), viewCount)
	}
}

func TestResponseViewContractRejectsUnreviewedNamesAndUnsupportedShapes(t *testing.T) {
	stringSchema := openapispec.SDKSurfaceSchema{Types: []string{"string"}}
	valid := openapispec.SDKSurfaceSchema{
		Types: []string{"object"},
		Properties: []openapispec.SDKSurfaceProperty{
			{Name: "status", Schema: stringSchema},
			{Name: "message", Schema: stringSchema},
			{Name: "corp_name", Schema: stringSchema},
			{Name: "list", Schema: openapispec.SDKSurfaceSchema{Types: []string{"array"}, Items: &openapispec.SDKSurfaceSchema{
				Types:      []string{"object"},
				Properties: []openapispec.SDKSurfaceProperty{{Name: "corp_code", Schema: stringSchema}},
			}}},
		},
	}
	generic := accessors{Source: "source", Status: "status", Message: "message", Items: "items", Field: "field"}
	operation := productOperation{
		LogicalID:  "logical",
		RustModule: "disclosure",
		ResponseViews: []responseView{
			{Path: "$", Accessors: map[string]string{"corp_name": "company_name"}},
			{Path: "$.list[]", RustType: "Company", Accessors: map[string]string{"corp_code": "company_code"}},
		},
	}
	if err := checkOperationResponseViews("manifest", operation, "DS001", generic, &valid, map[string]string{}, map[responseAccessorIdentity]responseAccessorBinding{}); err != nil {
		t.Fatal(err)
	}

	tests := []struct {
		name      string
		operation productOperation
		rule      string
	}{
		{name: "missing view", operation: productOperation{LogicalID: "logical", RustModule: "disclosure", ResponseViews: operation.ResponseViews[:1]}, rule: "response-view-coverage"},
		{name: "orphan accessor", operation: productOperation{LogicalID: "logical", RustModule: "disclosure", ResponseViews: []responseView{{Path: "$", Accessors: map[string]string{"other": "other"}}, operation.ResponseViews[1]}}, rule: "orphan-response-accessor"},
		{name: "generic collision", operation: productOperation{LogicalID: "logical", RustModule: "disclosure", ResponseViews: []responseView{{Path: "$", Accessors: map[string]string{"corp_name": "field"}}, operation.ResponseViews[1]}}, rule: "response-accessor-namespace"},
		{name: "reserved type", operation: productOperation{LogicalID: "logical", RustModule: "disclosure", ResponseViews: []responseView{operation.ResponseViews[0], {Path: "$.list[]", RustType: "Self", Accessors: map[string]string{"corp_code": "company_code"}}}}, rule: "response-view-type"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			assertRule(t, checkOperationResponseViews("manifest", test.operation, "DS001", generic, &valid, map[string]string{}, map[responseAccessorIdentity]responseAccessorBinding{}), test.rule)
		})
	}

	multipleArrays := valid
	multipleArrays.Properties = append(append([]openapispec.SDKSurfaceProperty(nil), valid.Properties...), openapispec.SDKSurfaceProperty{
		Name: "other", Schema: *valid.Properties[3].Schema.Items,
	})
	items := valid.Properties[3].Schema
	multipleArrays.Properties[len(multipleArrays.Properties)-1].Schema = items
	if _, err := responseViewContracts(multipleArrays); err == nil || !strings.Contains(err.Error(), "multiple array") {
		t.Fatalf("error = %v, want multiple-array shape rejection", err)
	}
	directObject := valid
	directObject.Properties = append(append([]openapispec.SDKSurfaceProperty(nil), valid.Properties...), openapispec.SDKSurfaceProperty{
		Name: "nested", Schema: *valid.Properties[3].Schema.Items,
	})
	if _, err := responseViewContracts(directObject); err == nil || !strings.Contains(err.Error(), "direct object") {
		t.Fatalf("error = %v, want direct-object shape rejection", err)
	}
	nonObjectItems := valid
	nonObjectItems.Properties = append([]openapispec.SDKSurfaceProperty(nil), valid.Properties...)
	nonObjectItems.Properties[3].Schema.Items = &stringSchema
	if _, err := responseViewContracts(nonObjectItems); err == nil || !strings.Contains(err.Error(), "must contain objects") {
		t.Fatalf("error = %v, want non-object item shape rejection", err)
	}
}

func TestLogicalResponseViewsRequireEquivalentRepresentationSchemas(t *testing.T) {
	root := repositoryRoot(t)
	sources, err := loadSourceOperations(root)
	if err != nil {
		t.Fatal(err)
	}
	var manifest batchManifest
	if err := decodeManifest(filepath.Join(root, "sdk", "rust", "interface", "ds001.toml"), &manifest); err != nil {
		t.Fatal(err)
	}
	var company productOperation
	for _, operation := range manifest.Operations {
		if operation.LogicalID == "DS001-2019002" {
			company = operation
			break
		}
	}
	xml := sources["get_company_xml"]
	mutated := normalizeResponseSchema(*xml.responseSchema)
	mutated.Properties[0].Name = "representation_drift"
	xml.responseSchema = &mutated
	sources["get_company_xml"] = xml
	err = checkProductOperation(
		"sdk/rust/interface/ds001.toml",
		"DS001",
		manifest.Accessors,
		company,
		sources,
		map[string]struct{}{},
		map[string]struct{}{},
		map[string]string{},
		map[string]string{},
		map[responseAccessorIdentity]responseAccessorBinding{},
	)
	assertRule(t, err, "shared-response-contract")
}

func TestInterfaceMutationsFailTheirReviewedRule(t *testing.T) {
	root := repositoryRoot(t)
	sources, err := loadSourceOperations(root)
	if err != nil {
		t.Fatal(err)
	}
	tests := []struct {
		name        string
		old         string
		replacement string
		rule        string
	}{
		{name: "logical alias", old: `cli_alias = "DS001-2019001"`, replacement: `cli_alias = "DS001-OTHER"`, rule: "logical-product-identity"},
		{name: "CLI namespace", old: `cli_command = "company-overview"`, replacement: `cli_command = "search-disclosures"`, rule: "duplicate-cli-command"},
		{name: "Rust response namespace", old: `rust_response = "CompanyOverviewJsonResponse"`, replacement: `rust_response = "DisclosureSearchJsonResponse"`, rule: "duplicate-rust-type"},
		{name: "Rust shared type namespace", old: `rust_input = "CompanyOverviewInput"`, replacement: `rust_input = "CompanyOverviewJsonResponse"`, rule: "duplicate-rust-type"},
		{name: "Rust reserved item", old: `rust_input = "CompanyOverviewInput"`, replacement: `rust_input = "Self"`, rule: "product-name-grammar"},
		{name: "response view path", old: `path = "$.list[]"`, replacement: `path = "$.items[]"`, rule: "orphan-response-view"},
		{name: "response accessor namespace", old: `page_no = "page_number"`, replacement: `page_no = "field"`, rule: "response-accessor-namespace"},
		{name: "response view type namespace", old: `rust_type = "DisclosureSummary"`, replacement: `rust_type = "CompanyOverviewInput"`, rule: "duplicate-rust-type"},
		{name: "representation method", old: `rust_method = "prepare_json"`, replacement: `rust_method = "prepare_xml"`, rule: "representation-name"},
		{name: "source parameter", old: `openapi_name = "corp_code"`, replacement: `openapi_name = "company_code"`, rule: "parameter-product-binding"},
		{name: "response accessor consistency", old: `od_a_at_t = "outside_directors_present_count"`, replacement: `od_a_at_t = "outside_directors_present"`, rule: "response-accessor-inconsistency"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			directory := copyManifestDirectory(t, filepath.Join(root, "sdk", "rust", "interface"))
			manifest := "ds001.toml"
			if test.name == "response accessor consistency" {
				manifest = "ds005.toml"
			}
			mutateFirst(t, filepath.Join(directory, manifest), test.old, test.replacement)
			_, err := checkBatches(directory, sources)
			assertRule(t, err, test.rule)
		})
	}
}

func TestSharedInputContractUsesExplicitInitialization(t *testing.T) {
	operation := productOperation{
		LogicalID: "logical", RustModule: "disclosure", RustInput: "Input", CLICommand: "logical", CLIAlias: "logical",
		Physical: []physicalProduct{
			{OperationID: "json", RustMethod: "prepare_json", RustResponse: "JsonResponse"},
			{OperationID: "xml", RustMethod: "prepare_xml", RustResponse: "XmlResponse"},
		},
	}
	tests := map[string]sourceOperation{
		"parameters": {
			logicalID: "logical", family: "DS001", representation: "application/xml",
			parameterContract: []openapispec.SDKSurfaceParameter{{Name: "corp_code"}}, parameters: []string{"corp_code"},
		},
		"security": {
			logicalID: "logical", family: "DS001", representation: "application/xml",
			security: []openapispec.SDKSurfaceSecurityRequirement{{Schemes: []openapispec.SDKSurfaceSecurityScheme{{Identifier: "apiKey"}}}},
		},
	}
	for name, second := range tests {
		t.Run(name, func(t *testing.T) {
			sources := map[string]sourceOperation{
				"json": {logicalID: "logical", family: "DS001", representation: "application/json"},
				"xml":  second,
			}
			err := checkProductOperation("manifest", "DS001", accessors{Source: "source", Status: "status", Message: "message", Items: "items", Field: "field"}, operation, sources,
				map[string]struct{}{}, map[string]struct{}{}, map[string]string{}, map[string]string{}, map[responseAccessorIdentity]responseAccessorBinding{})
			assertRule(t, err, "shared-input-contract")
		})
	}
}

func TestObligationAndCutoverGuardMutationsFailTheirReviewedRule(t *testing.T) {
	root := repositoryRoot(t)
	sources, err := loadSourceOperations(root)
	if err != nil {
		t.Fatal(err)
	}

	obligations := copyManifestFile(t, filepath.Join(root, "sdk", "rust", "conformance", "obligations.toml"))
	mutateFirst(t, obligations,
		`obligations = ["decoder", "prepare", "response-binding"]`,
		`obligations = ["prepare", "response-binding"]`)
	_, err = checkObligations(obligations, sources)
	assertRule(t, err, "obligation-set")

	obligations = copyManifestFile(t, filepath.Join(root, "sdk", "rust", "conformance", "obligations.toml"))
	mutateFirst(t, obligations,
		"operation_id = \"get_company_json\"\nobligations = [\"decoder\", \"prepare\", \"response-binding\"]\nexecutable = true",
		"operation_id = \"get_company_json\"\nobligations = [\"decoder\", \"prepare\", \"response-binding\"]\nexecutable = false")
	mutateFirst(t, obligations,
		"operation_id = \"get_list_json\"\nobligations = [\"decoder\", \"prepare\", \"response-binding\"]\nexecutable = false",
		"operation_id = \"get_list_json\"\nobligations = [\"decoder\", \"prepare\", \"response-binding\"]\nexecutable = true")
	_, err = checkObligations(obligations, sources)
	assertRule(t, err, "executable-case")

	guards := copyManifestFile(t, filepath.Join(root, "sdk", "rust", "conformance", "cutover-guards.toml"))
	mutateFirst(t, guards, `active = false`, `active = true`)
	assertRule(t, checkCutoverGuards(root, guards), "cutover-guard-activation")

	guards = copyManifestFile(t, filepath.Join(root, "sdk", "rust", "conformance", "cutover-guards.toml"))
	body, err := os.ReadFile(guards)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(guards, []byte(strings.ReplaceAll(string(body), `active = false`, `active = true`)), 0o600); err != nil {
		t.Fatal(err)
	}
	assertRule(t, checkCutoverGuards(root, guards), "cutover-guard-violation")
}

func TestCompileTimeGuardRejectsPublicRuntimeSelector(t *testing.T) {
	root := t.TempDir()
	sourceDirectory := filepath.Join(root, "sdk", "rust", "crates", "opendart", "src")
	if err := os.MkdirAll(sourceDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	cargo := filepath.Join(root, "sdk", "rust", "crates", "opendart", "Cargo.toml")
	if err := os.WriteFile(cargo, []byte("[features]\ndefault = []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourceDirectory, "lib.rs"), []byte("pub enum Backend { Generated, Handwritten }\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	guard := cutoverGuard{
		ID:        "public-implementation-selector",
		Kind:      "compile-time",
		Artifacts: []string{"sdk/rust/crates/opendart/Cargo.toml", "sdk/rust/crates/opendart/src"},
		Forbidden: []string{"generated =", "handwritten ="},
	}
	if err := enforceCutoverGuard(root, guard); err == nil || !strings.Contains(err.Error(), "public implementation selector") {
		t.Fatalf("error = %v, want public implementation selector rejection", err)
	}
}

func TestCompileTimeGuardIgnoresSelectorWordsInRustProse(t *testing.T) {
	body := `/// Generated and handwritten implementations are discussed here.
pub fn status() -> &'static str { "generated handwritten" }
`
	if looksLikePublicImplementationSelector(body) {
		t.Fatal("ordinary prose and string literals must not look like a public selector")
	}
}

func TestScopedGuardDoesNotApplyCargoFeatureTokensToRustSource(t *testing.T) {
	root := t.TempDir()
	crate := filepath.Join(root, "sdk", "rust", "crates", "opendart")
	if err := os.MkdirAll(filepath.Join(crate, "src"), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crate, "Cargo.toml"), []byte("[features]\ndefault = []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(crate, "src", "lib.rs"), []byte(`pub const NOTE: &str = "generated = is Cargo syntax";`), 0o600); err != nil {
		t.Fatal(err)
	}
	guard := cutoverGuard{
		ID: "public-implementation-selector", Kind: "compile-time",
		Targets: []cutoverGuardTarget{
			{Artifacts: []string{"sdk/rust/crates/opendart/Cargo.toml"}, Forbidden: []string{"generated =", "handwritten ="}},
			{Artifacts: []string{"sdk/rust/crates/opendart/src"}, Forbidden: []string{`cfg(feature = "generated")`, `cfg(feature = "handwritten")`}},
		},
	}
	if err := enforceCutoverGuard(root, guard); err != nil {
		t.Fatalf("scoped guard rejected unrelated Rust text: %v", err)
	}
	if err := os.WriteFile(filepath.Join(crate, "Cargo.toml"), []byte("[features]\ngenerated = []\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := enforceCutoverGuard(root, guard); err == nil || !strings.Contains(err.Error(), "forbidden public contract") {
		t.Fatalf("error = %v, want Cargo target rejection", err)
	}
}

func TestGeneratorProvenanceGuardRejectsPublicRequestSurface(t *testing.T) {
	for _, forbidden := range []string{"generator_schema", "projection_identity"} {
		t.Run(forbidden, func(t *testing.T) {
			root := t.TempDir()
			requestDirectory := filepath.Join(root, "sdk", "rust", "crates", "opendart", "src", "unrelated")
			if err := os.MkdirAll(requestDirectory, 0o700); err != nil {
				t.Fatal(err)
			}
			requestModule := filepath.Join(requestDirectory, "nested.rs")
			body := fmt.Sprintf("pub const fn %s(&self) {}\n", forbidden)
			if err := os.WriteFile(requestModule, []byte(body), 0o600); err != nil {
				t.Fatal(err)
			}
			guard := cutoverGuard{
				Kind:      "public-contract",
				Artifacts: []string{"sdk/rust/crates/opendart/src"},
				Forbidden: []string{forbidden},
			}
			if err := enforceCutoverGuard(root, guard); err == nil || !strings.Contains(err.Error(), "forbidden public contract") {
				t.Fatalf("error = %v, want request-provenance rejection", err)
			}
		})
	}
}

func TestStructuredExecutionGuardRejectsRenamedMovedRawResult(t *testing.T) {
	root := t.TempDir()
	sourceDirectory := filepath.Join(root, "sdk", "rust", "crates", "opendart", "src", "nested")
	if err := os.MkdirAll(sourceDirectory, 0o700); err != nil {
		t.Fatal(err)
	}
	body := `pub async fn inspect_untyped<T>(&self, request: &PreparedRequest<T>) -> Result<SourceResponse<SourceReply<SourceValue>>, ClientError> { todo!() }`
	if err := os.WriteFile(filepath.Join(sourceDirectory, "raw.rs"), []byte(body), 0o600); err != nil {
		t.Fatal(err)
	}
	guard := cutoverGuard{ID: "second-structured-execution-result", Kind: "public-contract", Artifacts: []string{"sdk/rust/crates/opendart/src"}, Forbidden: []string{"pub async fn execute_raw"}}
	if err := enforceCutoverGuard(root, guard); err == nil || !strings.Contains(err.Error(), "second structured execution result") {
		t.Fatalf("error = %v, want semantic structured-result rejection", err)
	}
}

func TestGuardArtifactValidationRejectsParentDirectory(t *testing.T) {
	if err := validateGuardValues([]string{".."}, false); err == nil {
		t.Fatal("bare parent directory must be rejected")
	}
}

func TestDecodeManifestRejectsZeroByteFile(t *testing.T) {
	path := filepath.Join(t.TempDir(), "empty.toml")
	if err := os.WriteFile(path, nil, 0o600); err != nil {
		t.Fatal(err)
	}
	var manifest batchManifest
	if err := decodeManifest(path, &manifest); err == nil {
		t.Fatal("zero-byte manifest must be rejected")
	}
}

func TestExecutableCaseHarnessRejectsMissingDispatcher(t *testing.T) {
	root := t.TempDir()
	relative := filepath.FromSlash("sdk/rust/crates/opendart/src/conformance.rs")
	path := filepath.Join(root, relative)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		t.Fatal(err)
	}
	body, err := os.ReadFile(filepath.Join(repositoryRoot(t), relative))
	if err != nil {
		t.Fatal(err)
	}
	body = []byte(strings.Replace(string(body), "fn executable_cases_cover_reviewed_operations()", "fn removed_dispatcher()", 1))
	if err := os.WriteFile(path, body, 0o600); err != nil {
		t.Fatal(err)
	}
	assertRule(t, checkExecutableCaseHarness(root), "executable-case-harness")
}

func repositoryRoot(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("resolve test path")
	}
	return filepath.Clean(filepath.Join(filepath.Dir(file), "..", ".."))
}

func copyManifestDirectory(t *testing.T, source string) string {
	t.Helper()
	target := t.TempDir()
	entries, err := os.ReadDir(source)
	if err != nil {
		t.Fatal(err)
	}
	for _, entry := range entries {
		if !entry.Type().IsRegular() {
			continue
		}
		body, err := os.ReadFile(filepath.Join(source, entry.Name()))
		if err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(target, entry.Name()), body, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return target
}

func copyManifestFile(t *testing.T, source string) string {
	t.Helper()
	body, err := os.ReadFile(source)
	if err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(t.TempDir(), filepath.Base(source))
	if err := os.WriteFile(target, body, 0o600); err != nil {
		t.Fatal(err)
	}
	return target
}

func mutateFirst(t *testing.T, path, old, replacement string) {
	t.Helper()
	body, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	updated := strings.Replace(string(body), old, replacement, 1)
	if updated == string(body) {
		t.Fatalf("mutation target %q was absent", old)
	}
	if err := os.WriteFile(path, []byte(updated), 0o600); err != nil {
		t.Fatal(err)
	}
}

func assertRule(t *testing.T, err error, want string) {
	t.Helper()
	var rejected *Error
	if !errors.As(err, &rejected) {
		t.Fatalf("error = %v, want rustconformance.Error", err)
	}
	if rejected.Rule != want {
		t.Fatalf("rule = %q, want %q", rejected.Rule, want)
	}
}
