// Package rustconformance validates the reviewed Rust and CLI product-name
// manifests and the independently authored OpenAPI case obligations.
package rustconformance

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"sort"
	"strings"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/pelletier/go-toml/v2"
)

const (
	manifestSchema  = 1
	maxManifestSize = 1 << 20
	maxGuardFiles   = 512
)

var (
	rustModuleName = regexp.MustCompile(`^[a-z][a-z0-9]*(?:_[a-z0-9]+)*$`)
	rustItemName   = regexp.MustCompile(`^[A-Z][A-Za-z0-9]*$`)
	cliName        = regexp.MustCompile(`^[a-z0-9]+(?:-[a-z0-9]+)*$`)
	rustKeywords   = map[string]struct{}{
		"as": {}, "async": {}, "await": {}, "break": {}, "const": {}, "continue": {},
		"crate": {}, "dyn": {}, "else": {}, "enum": {}, "extern": {}, "false": {},
		"fn": {}, "for": {}, "if": {}, "impl": {}, "in": {}, "let": {}, "loop": {},
		"match": {}, "mod": {}, "move": {}, "mut": {}, "pub": {}, "ref": {},
		"return": {}, "self": {}, "Self": {}, "static": {}, "struct": {}, "super": {},
		"trait": {}, "true": {}, "type": {}, "union": {}, "unsafe": {}, "use": {},
		"where": {}, "while": {}, "yield": {},
	}
	reviewedConsistentResponseAccessors = map[string]struct{}{
		"adt_a_atn": {}, "od_a_at_b": {}, "od_a_at_t": {},
	}
)

// Report contains only bounded inventory totals suitable for verification output.
type Report struct {
	Batches            int `json:"batches"`
	LogicalOperations  int `json:"logicalOperations"`
	PhysicalOperations int `json:"physicalOperations"`
	ResponseViews      int `json:"responseViews"`
	ResponseAccessors  int `json:"responseAccessors"`
	ExecutableCases    int `json:"executableCases"`
}

// Error identifies one stable rejected conformance rule without echoing manifest bodies.
type Error struct {
	Rule      string
	Artifact  string
	Operation string
	cause     error
}

func (e *Error) Error() string {
	parts := []string{"Rust conformance contract rejected", "rule=" + e.Rule, "artifact=" + e.Artifact}
	if e.Operation != "" {
		parts = append(parts, "operation="+e.Operation)
	}
	return strings.Join(parts, " ")
}

func (e *Error) Unwrap() error { return e.cause }

type accessors struct {
	Source  string `toml:"source"`
	Status  string `toml:"status"`
	Message string `toml:"message"`
	Items   string `toml:"items"`
	Field   string `toml:"field"`
}

type batchManifest struct {
	SchemaVersion int                `toml:"schema_version"`
	Family        string             `toml:"family"`
	Accessors     accessors          `toml:"accessors"`
	Operations    []productOperation `toml:"operation"`
}

type productOperation struct {
	LogicalID     string             `toml:"logical_id"`
	RustModule    string             `toml:"rust_module"`
	RustInput     string             `toml:"rust_input"`
	CLICommand    string             `toml:"cli_command"`
	CLIAlias      string             `toml:"cli_alias"`
	Physical      []physicalProduct  `toml:"physical"`
	Parameters    []parameterProduct `toml:"parameter"`
	ResponseViews []responseView     `toml:"response_view"`
}

type responseView struct {
	Path      string            `toml:"path"`
	RustType  string            `toml:"rust_type"`
	Accessors map[string]string `toml:"accessors"`
}

type physicalProduct struct {
	OperationID  string `toml:"operation_id"`
	RustMethod   string `toml:"rust_method"`
	RustResponse string `toml:"rust_response"`
}

type parameterProduct struct {
	OpenAPIName string `toml:"openapi_name"`
	RustName    string `toml:"rust_name"`
	CLIFlag     string `toml:"cli_flag"`
}

type obligationManifest struct {
	SchemaVersion int              `toml:"schema_version"`
	Cases         []obligationCase `toml:"case"`
}

type obligationCase struct {
	OperationID string   `toml:"operation_id"`
	Obligations []string `toml:"obligations"`
	Executable  bool     `toml:"executable"`
}

type cutoverGuardManifest struct {
	SchemaVersion int            `toml:"schema_version"`
	Activation    string         `toml:"activation"`
	Guards        []cutoverGuard `toml:"guard"`
}

type cutoverGuard struct {
	ID           string               `toml:"id"`
	Kind         string               `toml:"kind"`
	Active       bool                 `toml:"active"`
	Artifacts    []string             `toml:"artifacts"`
	Forbidden    []string             `toml:"forbidden"`
	MustBeAbsent []string             `toml:"must_be_absent"`
	Targets      []cutoverGuardTarget `toml:"target"`
}

type cutoverGuardTarget struct {
	Artifacts []string `toml:"artifacts"`
	Forbidden []string `toml:"forbidden"`
}

type sourceOperation struct {
	logicalID         string
	family            string
	parameters        []string
	parameterContract []openapispec.SDKSurfaceParameter
	security          []openapispec.SDKSurfaceSecurityRequirement
	representation    string
	alternates        []string
	responseSchema    *openapispec.SDKSurfaceSchema
}

// Check validates all six reviewed name batches and the complete physical
// obligation inventory against the canonical OpenAPI document.
func Check(repositoryRoot string) (Report, error) {
	root, err := filepath.Abs(repositoryRoot)
	if err != nil {
		return Report{}, reject("repository-root", "repository", "", err)
	}
	sources, err := loadSourceOperations(root)
	if err != nil {
		return Report{}, err
	}

	interfaceDir := filepath.Join(root, "sdk", "rust", "interface")
	report, err := checkBatches(interfaceDir, sources)
	if err != nil {
		return Report{}, err
	}
	executable, err := checkObligations(filepath.Join(root, "sdk", "rust", "conformance", "obligations.toml"), sources)
	if err != nil {
		return Report{}, err
	}
	if err := checkExecutableCaseHarness(root); err != nil {
		return Report{}, err
	}
	report.ExecutableCases = executable
	if err := checkCutoverGuards(root, filepath.Join(root, "sdk", "rust", "conformance", "cutover-guards.toml")); err != nil {
		return Report{}, err
	}
	return report, nil
}

func loadSourceOperations(root string) (map[string]sourceOperation, error) {
	document, err := openapispec.Load(filepath.Join(root, "openapi", "openapi.yaml"))
	if err != nil {
		return nil, reject("openapi-load", "openapi/openapi.yaml", "", err)
	}
	defer document.Close()
	surface, err := document.InspectSDKSurface()
	if err != nil {
		return nil, reject("openapi-surface", "openapi/openapi.yaml", "", err)
	}
	catalog, err := document.Operations()
	if err != nil {
		return nil, reject("openapi-inventory", "openapi/openapi.yaml", "", err)
	}
	primary := make(map[string]openapispec.Operation, len(catalog.Operations))
	for _, operation := range catalog.Operations {
		primary[operation.OperationID] = operation
	}
	sources := make(map[string]sourceOperation, len(surface.Operations))
	for _, operation := range surface.Operations {
		catalogOperation, ok := primary[operation.OperationID]
		if !ok {
			return nil, reject("inventory-disagreement", "openapi/openapi.yaml", operation.OperationID, nil)
		}
		parameters := make([]string, 0, len(operation.Parameters))
		for _, parameter := range operation.Parameters {
			parameters = append(parameters, parameter.Name)
		}
		sort.Strings(parameters)
		parameterContract := append([]openapispec.SDKSurfaceParameter(nil), operation.Parameters...)
		sort.Slice(parameterContract, func(left, right int) bool {
			return parameterContract[left].Name < parameterContract[right].Name
		})
		sources[operation.OperationID] = sourceOperation{
			logicalID:         operation.LogicalOperationID,
			family:            operation.APIGroupCode,
			parameters:        parameters,
			parameterContract: parameterContract,
			security:          append([]openapispec.SDKSurfaceSecurityRequirement(nil), operation.Security...),
			representation:    catalogOperation.PrimaryRepresentation,
			alternates:        append([]string(nil), catalogOperation.AlternateResponseMedia...),
		}
		if catalogOperation.PrimaryRepresentation != "application/zip" {
			schema, err := responseSchema(operation, catalogOperation.PrimaryRepresentation)
			if err != nil {
				return nil, reject("response-schema", "openapi/openapi.yaml", operation.OperationID, err)
			}
			value := sources[operation.OperationID]
			value.responseSchema = &schema
			sources[operation.OperationID] = value
		}
	}
	return sources, nil
}

func checkBatches(directory string, sources map[string]sourceOperation) (Report, error) {
	seenPhysical := make(map[string]struct{}, len(sources))
	seenLogical := make(map[string]struct{})
	seenRustTypes := make(map[string]string)
	seenCLICommands := make(map[string]string)
	seenResponseAccessors := make(map[responseAccessorIdentity]responseAccessorBinding)
	responseViews := 0
	responseAccessors := 0
	var approvedAccessors accessors
	for number := 1; number <= 6; number++ {
		family := fmt.Sprintf("DS%03d", number)
		artifact := fmt.Sprintf("sdk/rust/interface/ds%03d.toml", number)
		var manifest batchManifest
		if err := decodeManifest(filepath.Join(directory, fmt.Sprintf("ds%03d.toml", number)), &manifest); err != nil {
			return Report{}, reject("interface-manifest", artifact, "", err)
		}
		if manifest.SchemaVersion != manifestSchema || manifest.Family != family || len(manifest.Operations) == 0 {
			return Report{}, reject("interface-header", artifact, "", nil)
		}
		if err := validateAccessors(manifest.Accessors); err != nil {
			return Report{}, reject("accessor-name", artifact, "", err)
		}
		if number == 1 {
			approvedAccessors = manifest.Accessors
		} else if !reflect.DeepEqual(manifest.Accessors, approvedAccessors) {
			return Report{}, reject("accessor-inconsistency", artifact, "", nil)
		}
		for _, operation := range manifest.Operations {
			if err := checkProductOperation(artifact, family, manifest.Accessors, operation, sources, seenPhysical, seenLogical, seenRustTypes, seenCLICommands, seenResponseAccessors); err != nil {
				return Report{}, err
			}
			responseViews += len(operation.ResponseViews)
			for _, view := range operation.ResponseViews {
				responseAccessors += len(view.Accessors)
			}
		}
	}
	if len(seenPhysical) != len(sources) {
		return Report{}, reject("physical-coverage", "sdk/rust/interface", "", fmt.Errorf("mapped %d of %d", len(seenPhysical), len(sources)))
	}
	sourceLogical := make(map[string]struct{})
	for _, source := range sources {
		sourceLogical[source.logicalID] = struct{}{}
	}
	if len(seenLogical) != len(sourceLogical) {
		return Report{}, reject("logical-coverage", "sdk/rust/interface", "", fmt.Errorf("mapped %d of %d", len(seenLogical), len(sourceLogical)))
	}
	return Report{
		Batches:            6,
		LogicalOperations:  len(seenLogical),
		PhysicalOperations: len(seenPhysical),
		ResponseViews:      responseViews,
		ResponseAccessors:  responseAccessors,
	}, nil
}

func responseSchema(operation openapispec.SDKSurfaceOperation, representation string) (openapispec.SDKSurfaceSchema, error) {
	var schemas []openapispec.SDKSurfaceSchema
	for _, response := range operation.Responses {
		for _, media := range response.MediaTypes {
			if media.Name == representation {
				schemas = append(schemas, media.Schema)
			}
		}
	}
	if len(schemas) == 0 {
		return openapispec.SDKSurfaceSchema{}, fmt.Errorf("no %s response schema", representation)
	}
	for _, schema := range schemas[1:] {
		if !reflect.DeepEqual(schemas[0], schema) {
			return openapispec.SDKSurfaceSchema{}, fmt.Errorf("multiple %s response schemas disagree", representation)
		}
	}
	if len(schemas[0].Properties) == 0 {
		return openapispec.SDKSurfaceSchema{}, errors.New("structured response schema has no root object properties")
	}
	return normalizeResponseSchema(schemas[0]), nil
}

func normalizeResponseSchema(schema openapispec.SDKSurfaceSchema) openapispec.SDKSurfaceSchema {
	normalized := schema
	normalized.Reference = ""
	normalized.XMLName = ""
	normalized.XMLNodeType = ""
	normalized.Types = append([]string(nil), schema.Types...)
	normalized.Required = append([]string(nil), schema.Required...)
	normalized.StatusValues = append([]string(nil), schema.StatusValues...)
	normalized.Properties = make([]openapispec.SDKSurfaceProperty, len(schema.Properties))
	for index, property := range schema.Properties {
		normalized.Properties[index] = openapispec.SDKSurfaceProperty{
			Name:   property.Name,
			Schema: normalizeResponseSchema(property.Schema),
		}
	}
	if schema.Items != nil {
		item := normalizeResponseSchema(*schema.Items)
		normalized.Items = &item
	}
	if schema.AdditionalProperties != nil {
		additional := *schema.AdditionalProperties
		normalized.AdditionalProperties = &additional
	}
	return normalized
}

type responseViewContract struct {
	Path       string
	Properties []openapispec.SDKSurfaceProperty
}

type responseAccessorIdentity struct {
	Family      string
	RustModule  string
	Path        string
	OpenAPIName string
	Schema      string
}

type responseAccessorBinding struct {
	RustName  string
	Operation string
}

func responseViewContracts(schema openapispec.SDKSurfaceSchema) ([]responseViewContract, error) {
	return collectResponseViews(schema, "$", nil)
}

func collectResponseViews(schema openapispec.SDKSurfaceSchema, path string, views []responseViewContract) ([]responseViewContract, error) {
	if len(schema.Properties) == 0 {
		return views, errors.New("response view must be an object")
	}
	view := responseViewContract{Path: path}
	var arrayProperty *openapispec.SDKSurfaceProperty
	for _, property := range schema.Properties {
		if property.Schema.Items != nil {
			if arrayProperty != nil {
				return nil, fmt.Errorf("view %s has multiple array properties", path)
			}
			candidate := property
			arrayProperty = &candidate
			continue
		}
		if len(property.Schema.Properties) > 0 {
			return nil, fmt.Errorf("view %s has direct object property %s", path, property.Name)
		}
		if path != "$" || (property.Name != "status" && property.Name != "message") {
			view.Properties = append(view.Properties, property)
		}
	}
	views = append(views, view)
	if arrayProperty == nil {
		return views, nil
	}
	child := *arrayProperty.Schema.Items
	if child.Items != nil || len(child.Properties) == 0 {
		return nil, fmt.Errorf("view %s array %s must contain objects directly", path, arrayProperty.Name)
	}
	return collectResponseViews(child, path+"."+arrayProperty.Name+"[]", views)
}

func checkOperationResponseViews(
	artifact string,
	operation productOperation,
	family string,
	generic accessors,
	schema *openapispec.SDKSurfaceSchema,
	seenRustTypes map[string]string,
	seenResponseAccessors map[responseAccessorIdentity]responseAccessorBinding,
) error {
	if schema == nil {
		if len(operation.ResponseViews) != 0 {
			return reject("binary-response-view", artifact, operation.LogicalID, nil)
		}
		return nil
	}
	expected, err := responseViewContracts(*schema)
	if err != nil {
		return reject("response-view-shape", artifact, operation.LogicalID, err)
	}
	want := make(map[string]responseViewContract, len(expected))
	for _, view := range expected {
		want[view.Path] = view
	}
	seenPaths := make(map[string]struct{}, len(operation.ResponseViews))
	reserved := map[string]struct{}{
		generic.Source: {}, generic.Status: {}, generic.Message: {}, generic.Items: {}, generic.Field: {},
	}
	for _, reviewed := range operation.ResponseViews {
		contract, ok := want[reviewed.Path]
		if !ok {
			return reject("orphan-response-view", artifact, operation.LogicalID, fmt.Errorf("path %s", reviewed.Path))
		}
		if _, duplicate := seenPaths[reviewed.Path]; duplicate {
			return reject("duplicate-response-view", artifact, operation.LogicalID, fmt.Errorf("path %s", reviewed.Path))
		}
		if reviewed.Path == "$" {
			if reviewed.RustType != "" {
				return reject("root-response-view-type", artifact, operation.LogicalID, nil)
			}
		} else {
			if !validRustItem(reviewed.RustType) {
				return reject("response-view-type", artifact, operation.LogicalID, fmt.Errorf("path %s", reviewed.Path))
			}
			key := operation.RustModule + "::" + reviewed.RustType
			if prior, duplicate := seenRustTypes[key]; duplicate {
				return reject("duplicate-rust-type", artifact, operation.LogicalID, fmt.Errorf("also used by %s", prior))
			}
			seenRustTypes[key] = operation.LogicalID + " " + reviewed.Path
		}
		expectedProperties := make(map[string]openapispec.SDKSurfaceSchema, len(contract.Properties))
		for _, property := range contract.Properties {
			expectedProperties[property.Name] = property.Schema
		}
		seenMethods := make(map[string]string, len(reviewed.Accessors))
		for openAPIName, rustName := range reviewed.Accessors {
			propertySchema, ok := expectedProperties[openAPIName]
			if !ok {
				return reject("orphan-response-accessor", artifact, operation.LogicalID, fmt.Errorf("path %s property %s", reviewed.Path, openAPIName))
			}
			if !validRustMember(rustName) {
				return reject("response-accessor-name", artifact, operation.LogicalID, fmt.Errorf("path %s property %s", reviewed.Path, openAPIName))
			}
			if _, conflict := reserved[rustName]; conflict {
				return reject("response-accessor-namespace", artifact, operation.LogicalID, fmt.Errorf("path %s method %s conflicts with generic accessor", reviewed.Path, rustName))
			}
			if prior, duplicate := seenMethods[rustName]; duplicate {
				return reject("response-accessor-namespace", artifact, operation.LogicalID, fmt.Errorf("path %s properties %s and %s map to %s", reviewed.Path, prior, openAPIName, rustName))
			}
			if _, reviewedForConsistency := reviewedConsistentResponseAccessors[openAPIName]; reviewedForConsistency {
				schema, err := json.Marshal(normalizeResponseSchema(propertySchema))
				if err != nil {
					return reject("response-accessor-schema", artifact, operation.LogicalID, err)
				}
				identity := responseAccessorIdentity{
					Family: family, RustModule: operation.RustModule, Path: reviewed.Path,
					OpenAPIName: openAPIName, Schema: string(schema),
				}
				if prior, exists := seenResponseAccessors[identity]; exists && prior.RustName != rustName {
					return reject("response-accessor-inconsistency", artifact, operation.LogicalID,
						fmt.Errorf("path %s property %s maps to %s but %s uses %s", reviewed.Path, openAPIName, rustName, prior.Operation, prior.RustName))
				}
				seenResponseAccessors[identity] = responseAccessorBinding{RustName: rustName, Operation: operation.LogicalID}
			}
			seenMethods[rustName] = openAPIName
		}
		if len(reviewed.Accessors) != len(expectedProperties) {
			return reject("response-accessor-coverage", artifact, operation.LogicalID, fmt.Errorf("path %s mapped %d of %d", reviewed.Path, len(reviewed.Accessors), len(expectedProperties)))
		}
		seenPaths[reviewed.Path] = struct{}{}
	}
	if len(seenPaths) != len(want) {
		return reject("response-view-coverage", artifact, operation.LogicalID, fmt.Errorf("mapped %d of %d", len(seenPaths), len(want)))
	}
	return nil
}

func checkProductOperation(
	artifact, family string,
	generic accessors,
	operation productOperation,
	sources map[string]sourceOperation,
	seenPhysical, seenLogical map[string]struct{},
	seenRustTypes, seenCLICommands map[string]string,
	seenResponseAccessors map[responseAccessorIdentity]responseAccessorBinding,
) error {
	if operation.LogicalID == "" || operation.CLIAlias != operation.LogicalID || len(operation.Physical) == 0 {
		return reject("logical-product-identity", artifact, operation.LogicalID, nil)
	}
	if _, duplicate := seenLogical[operation.LogicalID]; duplicate {
		return reject("duplicate-logical-product", artifact, operation.LogicalID, nil)
	}
	if !validRustMember(operation.RustModule) || !validRustItem(operation.RustInput) || !cliName.MatchString(operation.CLICommand) {
		return reject("product-name-grammar", artifact, operation.LogicalID, nil)
	}
	rustKey := operation.RustModule + "::" + operation.RustInput
	if prior, duplicate := seenRustTypes[rustKey]; duplicate {
		return reject("duplicate-rust-type", artifact, operation.LogicalID, fmt.Errorf("also used by %s", prior))
	}
	seenRustTypes[rustKey] = operation.LogicalID
	if prior, duplicate := seenCLICommands[operation.CLICommand]; duplicate {
		return reject("duplicate-cli-command", artifact, operation.LogicalID, fmt.Errorf("also used by %s", prior))
	}

	expectedParameters := make(map[string]struct{})
	seenMethods := make(map[string]struct{})
	var sharedParameters []openapispec.SDKSurfaceParameter
	var sharedSecurity []openapispec.SDKSurfaceSecurityRequirement
	var sharedResponseSchema *openapispec.SDKSurfaceSchema
	haveSharedInput := false
	for _, physical := range operation.Physical {
		source, ok := sources[physical.OperationID]
		if !ok || source.logicalID != operation.LogicalID || source.family != family {
			return reject("physical-logical-binding", artifact, physical.OperationID, nil)
		}
		if _, duplicate := seenPhysical[physical.OperationID]; duplicate {
			return reject("duplicate-physical-product", artifact, physical.OperationID, nil)
		}
		if !validRustMember(physical.RustMethod) || !validRustItem(physical.RustResponse) {
			return reject("representation-name-grammar", artifact, physical.OperationID, nil)
		}
		if err := checkRepresentationNames(source.representation, physical); err != nil {
			return reject("representation-name", artifact, physical.OperationID, err)
		}
		if !haveSharedInput {
			sharedParameters = source.parameterContract
			sharedSecurity = source.security
			haveSharedInput = true
		} else if !reflect.DeepEqual(sharedParameters, source.parameterContract) || !reflect.DeepEqual(sharedSecurity, source.security) {
			return reject("shared-input-contract", artifact, operation.LogicalID, nil)
		}
		if source.responseSchema != nil {
			if sharedResponseSchema == nil {
				schema := *source.responseSchema
				sharedResponseSchema = &schema
			} else if !reflect.DeepEqual(*sharedResponseSchema, *source.responseSchema) {
				return reject("shared-response-contract", artifact, operation.LogicalID, nil)
			}
		}
		if _, duplicate := seenMethods[physical.RustMethod]; duplicate {
			return reject("duplicate-rust-method", artifact, operation.LogicalID, nil)
		}
		responseKey := operation.RustModule + "::" + physical.RustResponse
		if prior, duplicate := seenRustTypes[responseKey]; duplicate {
			return reject("duplicate-rust-type", artifact, operation.LogicalID, fmt.Errorf("also used by %s", prior))
		}
		seenPhysical[physical.OperationID] = struct{}{}
		seenMethods[physical.RustMethod] = struct{}{}
		seenRustTypes[responseKey] = physical.OperationID
		for _, parameter := range source.parameters {
			expectedParameters[parameter] = struct{}{}
		}
	}
	if err := checkOperationResponseViews(artifact, operation, family, generic, sharedResponseSchema, seenRustTypes, seenResponseAccessors); err != nil {
		return err
	}
	seenOpenAPIParameters := make(map[string]struct{})
	seenRustParameters := make(map[string]struct{})
	seenCLIFlags := make(map[string]struct{})
	for _, parameter := range operation.Parameters {
		if _, ok := expectedParameters[parameter.OpenAPIName]; !ok || !validRustMember(parameter.RustName) || !cliName.MatchString(parameter.CLIFlag) {
			return reject("parameter-product-binding", artifact, operation.LogicalID, nil)
		}
		if _, duplicate := seenOpenAPIParameters[parameter.OpenAPIName]; duplicate {
			return reject("duplicate-openapi-parameter", artifact, operation.LogicalID, nil)
		}
		if _, duplicate := seenRustParameters[parameter.RustName]; duplicate {
			return reject("duplicate-rust-parameter", artifact, operation.LogicalID, nil)
		}
		if _, duplicate := seenCLIFlags[parameter.CLIFlag]; duplicate {
			return reject("duplicate-cli-flag", artifact, operation.LogicalID, nil)
		}
		seenOpenAPIParameters[parameter.OpenAPIName] = struct{}{}
		seenRustParameters[parameter.RustName] = struct{}{}
		seenCLIFlags[parameter.CLIFlag] = struct{}{}
	}
	if len(seenOpenAPIParameters) != len(expectedParameters) {
		return reject("parameter-coverage", artifact, operation.LogicalID, fmt.Errorf("mapped %d of %d", len(seenOpenAPIParameters), len(expectedParameters)))
	}
	seenLogical[operation.LogicalID] = struct{}{}
	seenCLICommands[operation.CLICommand] = operation.LogicalID
	return nil
}

func checkObligations(path string, sources map[string]sourceOperation) (int, error) {
	artifact := "sdk/rust/conformance/obligations.toml"
	var manifest obligationManifest
	if err := decodeManifest(path, &manifest); err != nil {
		return 0, reject("obligation-manifest", artifact, "", err)
	}
	if manifest.SchemaVersion != manifestSchema || len(manifest.Cases) == 0 {
		return 0, reject("obligation-header", artifact, "", nil)
	}
	seen := make(map[string]struct{}, len(manifest.Cases))
	wantExecutable := map[string]string{
		"get_company_json": "application/json",
		"get_company_xml":  "application/xml",
		"get_corpCode_xml": "application/zip",
	}
	seenExecutable := make(map[string]struct{}, len(wantExecutable))
	executableCount := 0
	for _, item := range manifest.Cases {
		source, ok := sources[item.OperationID]
		if !ok {
			return 0, reject("unknown-obligation-operation", artifact, item.OperationID, nil)
		}
		if _, duplicate := seen[item.OperationID]; duplicate {
			return 0, reject("duplicate-obligation-operation", artifact, item.OperationID, nil)
		}
		if source.representation == "application/zip" && !slices.Contains(source.alternates, "application/xml") {
			return 0, reject("binary-alternate-status", artifact, item.OperationID, nil)
		}
		got := append([]string(nil), item.Obligations...)
		sort.Strings(got)
		want := expectedObligations(source)
		if !slices.Equal(got, want) {
			return 0, reject("obligation-set", artifact, item.OperationID, fmt.Errorf("got %v want %v", got, want))
		}
		if item.Executable {
			wantRepresentation, reviewed := wantExecutable[item.OperationID]
			if !reviewed || source.representation != wantRepresentation {
				return 0, reject("executable-case", artifact, item.OperationID, nil)
			}
			executableCount++
			seenExecutable[item.OperationID] = struct{}{}
		} else if _, reviewed := wantExecutable[item.OperationID]; reviewed {
			return 0, reject("executable-case", artifact, item.OperationID, nil)
		}
		seen[item.OperationID] = struct{}{}
	}
	if len(seen) != len(sources) {
		return 0, reject("obligation-coverage", artifact, "", fmt.Errorf("mapped %d of %d", len(seen), len(sources)))
	}
	if len(seenExecutable) != len(wantExecutable) {
		return 0, reject("executable-case", artifact, "", fmt.Errorf("mapped %d of %d", len(seenExecutable), len(wantExecutable)))
	}
	return executableCount, nil
}

func checkExecutableCaseHarness(repositoryRoot string) error {
	artifact := "sdk/rust/crates/opendart/src/conformance.rs"
	path := filepath.Join(repositoryRoot, filepath.FromSlash(artifact))
	info, err := os.Lstat(path)
	if err != nil {
		return reject("executable-case-harness", artifact, "", err)
	}
	if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxManifestSize {
		return reject("executable-case-harness", artifact, "", errors.New("harness must be a bounded regular file"))
	}
	body, err := os.ReadFile(path)
	if err != nil {
		return reject("executable-case-harness", artifact, "", err)
	}
	code := stripRustComments(string(body))
	testDeclaration := regexp.MustCompile(`(?s)#\s*\[\s*test\s*\]\s*fn\s+executable_cases_cover_reviewed_operations\s*\(\s*\)`)
	if !testDeclaration.MatchString(code) {
		return reject("executable-case-harness", artifact, "", errors.New("reviewed executable dispatcher is not a Rust test"))
	}
	required := []string{
		`include_str!("../../../conformance/obligations.toml")`,
		"for operation_id in operation_ids",
		`"get_company_json" => executable_get_company_json()`,
		`"get_company_xml" => executable_get_company_xml()`,
		`"get_corpCode_xml" => executable_get_corp_code_xml()`,
	}
	for _, token := range required {
		if !strings.Contains(code, token) {
			return reject("executable-case-harness", artifact, "", fmt.Errorf("missing required harness token %q", token))
		}
	}
	return nil
}

func expectedObligations(source sourceOperation) []string {
	if source.representation == "application/zip" {
		return []string{"alternate-status", "prepare", "streaming"}
	}
	return []string{"decoder", "prepare", "response-binding"}
}

func checkRepresentationNames(representation string, value physicalProduct) error {
	var wantMethod, wantResponsePart string
	switch representation {
	case "application/json":
		wantMethod, wantResponsePart = "prepare_json", "JsonResponse"
	case "application/xml":
		wantMethod, wantResponsePart = "prepare_xml", "XmlResponse"
	case "application/zip":
		wantMethod, wantResponsePart = "prepare_archive", "ArchiveResponse"
	default:
		return fmt.Errorf("unsupported primary representation %q", representation)
	}
	if value.RustMethod != wantMethod || !strings.HasSuffix(value.RustResponse, wantResponsePart) {
		return fmt.Errorf("expected %s and *%s", wantMethod, wantResponsePart)
	}
	return nil
}

func checkCutoverGuards(repositoryRoot, path string) error {
	artifact := "sdk/rust/conformance/cutover-guards.toml"
	var manifest cutoverGuardManifest
	if err := decodeManifest(path, &manifest); err != nil {
		return reject("cutover-guard-manifest", artifact, "", err)
	}
	if manifest.SchemaVersion != manifestSchema || manifest.Activation != "handwritten-sdk-cutover" {
		return reject("cutover-guard-header", artifact, "", nil)
	}
	want := map[string]cutoverGuard{
		"generated-sdk-exports": {
			Kind:         "public-contract",
			Artifacts:    []string{"sdk/rust/crates/opendart/src/lib.rs"},
			Forbidden:    []string{"mod generated;", "pub use generated::operations;", "pub use generated::responses;"},
			MustBeAbsent: []string{"sdk/rust/crates/opendart/src/generated"},
		},
		"generator-provenance": {
			Kind:      "public-contract",
			Artifacts: []string{"sdk/rust/crates/opendart/src"},
			Forbidden: []string{
				"GENERATOR_SCHEMA", "PROJECTION_CHECKSUM", "generator_schema",
				"projection_identity", "projection_sha256",
			},
		},
		"public-implementation-selector": {
			Kind: "compile-time",
			Targets: []cutoverGuardTarget{
				{
					Artifacts: []string{"sdk/rust/crates/opendart/Cargo.toml"},
					Forbidden: []string{"generated =", "handwritten ="},
				},
				{
					Artifacts: []string{"sdk/rust/crates/opendart/src"},
					Forbidden: []string{
						"cfg(feature = \"generated\")", "cfg(feature = \"handwritten\")",
						"pub enum Implementation", "pub struct Implementation",
						"pub fn implementation(", "pub fn with_implementation(",
					},
				},
			},
		},
		"second-structured-execution-result": {
			Kind:      "public-contract",
			Artifacts: []string{"sdk/rust/crates/opendart/src"},
			Forbidden: []string{"pub async fn execute_raw"},
		},
	}
	seen := make(map[string]struct{}, len(manifest.Guards))
	active := 0
	for _, guard := range manifest.Guards {
		expected, ok := want[guard.ID]
		if !ok || guard.Kind != expected.Kind ||
			!slices.Equal(guard.Artifacts, expected.Artifacts) ||
			!slices.Equal(guard.Forbidden, expected.Forbidden) ||
			!slices.Equal(guard.MustBeAbsent, expected.MustBeAbsent) ||
			!reflect.DeepEqual(guard.Targets, expected.Targets) {
			return reject("cutover-guard-definition", artifact, guard.ID, nil)
		}
		if _, duplicate := seen[guard.ID]; duplicate {
			return reject("duplicate-cutover-guard", artifact, guard.ID, nil)
		}
		if err := validateGuardValues(guard.Artifacts, false); err != nil {
			return reject("cutover-guard-artifacts", artifact, guard.ID, err)
		}
		if err := validateGuardValues(guard.MustBeAbsent, false); err != nil {
			return reject("cutover-guard-artifacts", artifact, guard.ID, err)
		}
		if err := validateGuardValues(guard.Forbidden, true); err != nil {
			if len(guard.Targets) == 0 {
				return reject("cutover-guard-targets", artifact, guard.ID, err)
			}
		}
		if len(guard.Targets) > 0 && (len(guard.Artifacts) > 0 || len(guard.Forbidden) > 0) {
			return reject("cutover-guard-targets", artifact, guard.ID, errors.New("guard cannot mix scoped and unscoped targets"))
		}
		for _, target := range guard.Targets {
			if err := validateGuardValues(target.Artifacts, false); err != nil {
				return reject("cutover-guard-artifacts", artifact, guard.ID, err)
			}
			if err := validateGuardValues(target.Forbidden, true); err != nil {
				return reject("cutover-guard-targets", artifact, guard.ID, err)
			}
		}
		if guard.Active {
			active++
		}
		seen[guard.ID] = struct{}{}
	}
	if len(seen) != len(want) {
		return reject("cutover-guard-coverage", artifact, "", fmt.Errorf("defined %d of %d", len(seen), len(want)))
	}
	if active != 0 && active != len(want) {
		return reject("cutover-guard-activation", artifact, "", fmt.Errorf("active %d of %d", active, len(want)))
	}
	if active == len(want) {
		for _, guard := range manifest.Guards {
			if err := enforceCutoverGuard(repositoryRoot, guard); err != nil {
				return reject("cutover-guard-violation", artifact, guard.ID, err)
			}
		}
	}
	return nil
}

func validateGuardValues(values []string, requireValues bool) error {
	if requireValues && len(values) == 0 {
		return errors.New("guard requires at least one target")
	}
	items := append([]string(nil), values...)
	sort.Strings(items)
	if slices.Contains(items, "") || len(slices.Compact(items)) != len(items) {
		return errors.New("guard values must be non-empty and unique")
	}
	for _, value := range items {
		clean := filepath.Clean(filepath.FromSlash(value))
		if !requireValues && (clean == "." || clean == ".." || filepath.IsAbs(clean) || clean != filepath.FromSlash(value) || strings.HasPrefix(clean, ".."+string(filepath.Separator))) {
			return errors.New("guard artifact must be a normalized repository-relative path")
		}
	}
	return nil
}

func enforceCutoverGuard(repositoryRoot string, guard cutoverGuard) error {
	for _, relative := range guard.MustBeAbsent {
		_, err := os.Lstat(filepath.Join(repositoryRoot, filepath.FromSlash(relative)))
		if err == nil {
			return fmt.Errorf("forbidden artifact remains at %s", relative)
		}
		if !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	targets := guard.Targets
	if len(targets) == 0 {
		targets = []cutoverGuardTarget{{Artifacts: guard.Artifacts, Forbidden: guard.Forbidden}}
	}
	for _, target := range targets {
		if err := enforceCutoverGuardTarget(repositoryRoot, guard, target); err != nil {
			return err
		}
	}
	return nil
}

func enforceCutoverGuardTarget(repositoryRoot string, guard cutoverGuard, target cutoverGuardTarget) error {
	files, err := guardArtifactFiles(repositoryRoot, target.Artifacts)
	if err != nil {
		return err
	}
	for _, path := range files {
		// #nosec G304 -- paths descend from fixed, normalized repository artifacts.
		body, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, forbidden := range target.Forbidden {
			if strings.Contains(string(body), forbidden) {
				return fmt.Errorf("forbidden public contract remains in %s", filepath.ToSlash(path))
			}
		}
		if guard.ID == "public-implementation-selector" && filepath.Ext(path) == ".rs" && looksLikePublicImplementationSelector(string(body)) {
			return fmt.Errorf("public implementation selector remains in %s", filepath.ToSlash(path))
		}
		if guard.ID == "second-structured-execution-result" && filepath.Ext(path) == ".rs" && looksLikeSecondStructuredExecutionResult(string(body)) {
			return fmt.Errorf("second structured execution result remains in %s", filepath.ToSlash(path))
		}
	}
	return nil
}

func guardArtifactFiles(repositoryRoot string, artifacts []string) ([]string, error) {
	files := make([]string, 0, len(artifacts))
	for _, relative := range artifacts {
		path := filepath.Join(repositoryRoot, filepath.FromSlash(relative))
		info, err := os.Lstat(path)
		if err != nil {
			return nil, err
		}
		if info.Mode().IsRegular() {
			if info.Size() > maxManifestSize {
				return nil, errors.New("guard artifact must be bounded")
			}
			files = append(files, path)
			continue
		}
		if !info.IsDir() {
			return nil, errors.New("guard artifact must be a regular file or directory")
		}
		err = filepath.WalkDir(path, func(candidate string, entry os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				return nil
			}
			entryInfo, err := entry.Info()
			if err != nil {
				return err
			}
			if !entryInfo.Mode().IsRegular() || entryInfo.Size() > maxManifestSize {
				return errors.New("guard source artifact must be a bounded regular file")
			}
			if filepath.Ext(candidate) == ".rs" {
				files = append(files, candidate)
				if len(files) > maxGuardFiles {
					return errors.New("guard artifact file count exceeds bound")
				}
			}
			return nil
		})
		if err != nil {
			return nil, err
		}
	}
	return files, nil
}

func looksLikePublicImplementationSelector(body string) bool {
	code := stripRustCommentsAndStrings(body)
	selectorType := regexp.MustCompile(`(?s)\bpub(?:\s*\([^)]*\))?\s+(?:enum|struct)\s+[A-Za-z_][A-Za-z0-9_]*\s*\{[^}]*(?:\bGenerated\b[^}]*\bHandwritten\b|\bHandwritten\b[^}]*\bGenerated\b)`)
	selectorCfg := regexp.MustCompile(`(?s)#\s*\[\s*cfg\s*\([^]]*\b(?:generated|handwritten)\b[^]]*\)\s*\]\s*pub(?:\s*\([^)]*\))?\s+(?:async\s+)?(?:fn|struct|enum|type|trait|mod)\b`)
	return selectorType.MatchString(code) || selectorCfg.MatchString(strings.ToLower(stripRustComments(body)))
}

func looksLikeSecondStructuredExecutionResult(body string) bool {
	code := stripRustCommentsAndStrings(body)
	publicAsync := regexp.MustCompile(`(?s)\bpub(?:\s*\([^)]*\))?\s+async\s+fn\s+[A-Za-z_][A-Za-z0-9_]*\s*(?:<[^>{}]*>)?\s*\([^)]*\)\s*->\s*Result\s*<\s*SourceResponse\s*<\s*SourceReply\s*<\s*SourceValue\s*>\s*>`)
	return publicAsync.MatchString(code)
}

func stripRustCommentsAndStrings(body string) string {
	result := []byte(body)
	inLineComment, inBlockComment, inString, escaped := false, false, false, false
	for index := 0; index < len(result); index++ {
		if inLineComment {
			if result[index] == '\n' {
				inLineComment = false
			} else {
				result[index] = ' '
			}
			continue
		}
		if inBlockComment {
			if index+1 < len(result) && result[index] == '*' && result[index+1] == '/' {
				result[index], result[index+1] = ' ', ' '
				index++
				inBlockComment = false
			} else if result[index] != '\n' {
				result[index] = ' '
			}
			continue
		}
		if inString {
			if escaped {
				escaped = false
			} else if result[index] == '\\' {
				escaped = true
			} else if result[index] == '"' {
				inString = false
			}
			if result[index] != '\n' {
				result[index] = ' '
			}
			continue
		}
		if index+1 < len(result) && result[index] == '/' && result[index+1] == '/' {
			result[index], result[index+1] = ' ', ' '
			index++
			inLineComment = true
			continue
		}
		if index+1 < len(result) && result[index] == '/' && result[index+1] == '*' {
			result[index], result[index+1] = ' ', ' '
			index++
			inBlockComment = true
			continue
		}
		if result[index] == '"' {
			result[index] = ' '
			inString = true
		}
	}
	return string(result)
}

func stripRustComments(body string) string {
	result := []byte(body)
	inLineComment, inBlockComment := false, false
	for index := 0; index < len(result); index++ {
		if inLineComment {
			if result[index] == '\n' {
				inLineComment = false
			} else {
				result[index] = ' '
			}
			continue
		}
		if inBlockComment {
			if index+1 < len(result) && result[index] == '*' && result[index+1] == '/' {
				result[index], result[index+1] = ' ', ' '
				index++
				inBlockComment = false
			} else if result[index] != '\n' {
				result[index] = ' '
			}
			continue
		}
		if index+1 < len(result) && result[index] == '/' && result[index+1] == '/' {
			result[index], result[index+1] = ' ', ' '
			index++
			inLineComment = true
		} else if index+1 < len(result) && result[index] == '/' && result[index+1] == '*' {
			result[index], result[index+1] = ' ', ' '
			index++
			inBlockComment = true
		}
	}
	return string(result)
}

func validateAccessors(value accessors) error {
	values := []string{value.Source, value.Status, value.Message, value.Items, value.Field}
	seen := make(map[string]struct{}, len(values))
	for _, name := range values {
		if !validRustMember(name) {
			return errors.New("accessor is not an idiomatic Rust member name")
		}
		if _, duplicate := seen[name]; duplicate {
			return errors.New("accessor names must be unique")
		}
		seen[name] = struct{}{}
	}
	return nil
}

func validRustMember(name string) bool {
	if !rustModuleName.MatchString(name) {
		return false
	}
	_, reserved := rustKeywords[name]
	return !reserved
}

func validRustItem(name string) bool {
	if !rustItemName.MatchString(name) {
		return false
	}
	_, reserved := rustKeywords[name]
	return !reserved
}

func decodeManifest(path string, target any) error {
	info, err := os.Lstat(path)
	if err != nil {
		return err
	}
	if !info.Mode().IsRegular() || info.Size() == 0 || info.Size() > maxManifestSize {
		return errors.New("manifest must be a bounded regular file")
	}
	// #nosec G304 -- callers supply fixed paths below the repository root.
	body, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	decoder := toml.NewDecoder(strings.NewReader(string(body)))
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func reject(rule, artifact, operation string, cause error) *Error {
	return &Error{Rule: rule, Artifact: artifact, Operation: operation, cause: cause}
}
