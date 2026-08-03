// Package model validates and normalizes the two generated CLI projections.
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strconv"
	"strings"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/rustinterface"
)

const (
	// CLIInterfaceProjectionSchemaVersion identifies public grammar and discovery.
	CLIInterfaceProjectionSchemaVersion uint32 = 1
	// CLIDispatchProjectionSchemaVersion identifies private handwritten-SDK wiring.
	CLIDispatchProjectionSchemaVersion uint32 = 1
)

type Representation string

const (
	RepresentationJSON Representation = "json"
	RepresentationXML  Representation = "xml"
	RepresentationZIP  Representation = "zip"
)

type ParameterShape string

const (
	ScalarString ParameterShape = "string"
	StringArray  ParameterShape = "string-array"
)

// StringConstraints is the public CLI description of accepted string values.
// The SDK remains the runtime validation authority.
type StringConstraints struct {
	Format         string   `json:"format,omitempty"`
	AllowedValues  []string `json:"allowedValues,omitempty"`
	MinLength      *int64   `json:"minLength,omitempty"`
	MaxLength      *int64   `json:"maxLength,omitempty"`
	DecimalMinimum *int64   `json:"decimalMinimum,omitempty"`
	DecimalMaximum *int64   `json:"decimalMaximum,omitempty"`
}

// Error reports a projection rule without including source bodies.
type Error struct {
	Rule      string
	Operation string
	Location  string
	Detail    string
}

func (e *Error) Error() string {
	parts := []string{"CLI projection rejected input", "rule=" + e.Rule}
	if e.Operation != "" {
		parts = append(parts, "operation="+e.Operation)
	}
	if e.Location != "" {
		parts = append(parts, "location="+e.Location)
	}
	if e.Detail != "" {
		parts = append(parts, "detail="+e.Detail)
	}
	return strings.Join(parts, " ")
}

// ArtifactSet contains the independently identified public interface and
// private handwritten dispatch projections.
type ArtifactSet struct {
	CLIInterface CLIInterfaceModel
	CLIDispatch  CLIDispatchModel
}

// CLIInterfaceModel is the Rust-symbol-free public grammar and discovery input.
type CLIInterfaceModel struct {
	SchemaVersion uint32         `json:"schemaVersion"`
	Checksum      string         `json:"checksum,omitempty"`
	Operations    []CLIOperation `json:"operations"`
}

type CLIOperation struct {
	Name            string              `json:"name"`
	LogicalID       string              `json:"logicalId"`
	Group           string              `json:"group"`
	APIID           string              `json:"apiId"`
	GuideURL        string              `json:"guideUrl"`
	Description     string              `json:"description"`
	Parameters      []CLIParameter      `json:"parameters"`
	Representations []CLIRepresentation `json:"representations"`
}

type CLIParameter struct {
	Flag        string            `json:"flag"`
	SourceName  string            `json:"sourceName"`
	Description string            `json:"description"`
	Required    bool              `json:"required"`
	Shape       ParameterShape    `json:"shape"`
	MinItems    *int64            `json:"minItems,omitempty"`
	MaxItems    *int64            `json:"maxItems,omitempty"`
	Constraints StringConstraints `json:"constraints,omitzero"`
}

type CLIRepresentation struct {
	Name          Representation   `json:"name"`
	PhysicalID    string           `json:"physicalId"`
	Selector      bool             `json:"selector"`
	ResponseShape CLIResponseShape `json:"responseShape"`
}

type CLIResponseShape struct {
	Kind string `json:"kind"`
}

// CLIDispatchModel is the separately identified private handwritten adapter.
type CLIDispatchModel struct {
	SchemaVersion uint32                 `json:"schemaVersion"`
	Checksum      string                 `json:"checksum,omitempty"`
	Operations    []CLIDispatchOperation `json:"operations"`
}

type CLIDispatchOperation struct {
	Name            string                      `json:"name"`
	LogicalID       string                      `json:"logicalId"`
	RustModule      string                      `json:"rustModule"`
	RustInput       string                      `json:"rustInput"`
	Parameters      []CLIDispatchParameter      `json:"parameters"`
	Representations []CLIDispatchRepresentation `json:"representations"`
}

type CLIDispatchParameter struct {
	ArgumentID string         `json:"argumentId"`
	RustField  string         `json:"rustField"`
	Required   bool           `json:"required"`
	Shape      ParameterShape `json:"shape"`
}

type CLIDispatchRepresentation struct {
	Name          Representation `json:"name"`
	PhysicalID    string         `json:"physicalId"`
	PrepareMethod string         `json:"prepareMethod"`
	ResponseType  string         `json:"responseType"`
	TestArgv      []string       `json:"testArgv"`
}

type logicalOperation struct {
	id         string
	group      string
	apiID      string
	guideURL   string
	parameters []sourceParameter
	variants   []physicalVariant
	sources    []openapispec.SDKSurfaceOperation
}

type sourceParameter struct {
	name        string
	description string
	required    bool
	shape       ParameterShape
	minItems    *int64
	maxItems    *int64
	constraints StringConstraints
}

type physicalVariant struct {
	operationID    string
	representation Representation
}

// BuildArtifacts builds only the two CLI-owned projections directly from the
// canonical OpenAPI surface and reviewed product manifests.
func BuildArtifacts(surface openapispec.SDKSurface, batches []rustinterface.Batch) (ArtifactSet, error) {
	logical, err := normalizeCLISources(surface)
	if err != nil {
		return ArtifactSet{}, err
	}
	cliInterface, cliDispatch, err := buildCLIProjections(logical, batches)
	if err != nil {
		return ArtifactSet{}, err
	}
	return ArtifactSet{CLIInterface: cliInterface, CLIDispatch: cliDispatch}, nil
}

func normalizeCLISources(surface openapispec.SDKSurface) ([]logicalOperation, error) {
	if len(surface.Operations) == 0 {
		return nil, reject("operation-inventory", "", "#/paths", "no physical operations")
	}
	grouped := make(map[string][]openapispec.SDKSurfaceOperation)
	physical := make(map[string]bool, len(surface.Operations))
	for _, source := range surface.Operations {
		if source.OperationID == "" || source.LogicalOperationID == "" {
			return nil, reject("operation-identity", source.OperationID, "operationId", "missing physical or logical identity")
		}
		if physical[source.OperationID] {
			return nil, reject("duplicate-operation-id", source.OperationID, "operationId", source.OperationID)
		}
		physical[source.OperationID] = true
		grouped[source.LogicalOperationID] = append(grouped[source.LogicalOperationID], source)
	}
	ids := make([]string, 0, len(grouped))
	for id := range grouped {
		ids = append(ids, id)
	}
	sort.Strings(ids)
	result := make([]logicalOperation, 0, len(ids))
	for _, id := range ids {
		sources := grouped[id]
		sort.Slice(sources, func(i, j int) bool { return sources[i].OperationID < sources[j].OperationID })
		first := sources[0]
		parameters, err := normalizeCLIParameters(first)
		if err != nil {
			return nil, err
		}
		operation := logicalOperation{id: id, group: strings.ToUpper(first.APIGroupCode), apiID: first.APIID, guideURL: first.GuideURL, parameters: parameters, sources: sources}
		seenRepresentations := make(map[Representation]bool)
		for _, source := range sources {
			candidate, err := normalizeCLIParameters(source)
			if err != nil {
				return nil, err
			}
			if source.APIGroupCode != first.APIGroupCode || source.APIID != first.APIID || source.GuideURL != first.GuideURL || !equalParameterContracts(candidate, parameters) {
				return nil, reject("incompatible-logical-variants", source.OperationID, "parameters", id)
			}
			representation, err := primaryRepresentation(source)
			if err != nil {
				return nil, err
			}
			if seenRepresentations[representation] {
				return nil, reject("duplicate-logical-representation", source.OperationID, "responses", string(representation))
			}
			seenRepresentations[representation] = true
			operation.variants = append(operation.variants, physicalVariant{operationID: source.OperationID, representation: representation})
		}
		sort.Slice(operation.variants, func(i, j int) bool {
			return operation.variants[i].representation < operation.variants[j].representation
		})
		result = append(result, operation)
	}
	return result, nil
}

func equalParameterContracts(left, right []sourceParameter) bool {
	if len(left) != len(right) {
		return false
	}
	for index := range left {
		leftParameter := left[index]
		rightParameter := right[index]
		leftParameter.description = ""
		rightParameter.description = ""
		if !reflect.DeepEqual(leftParameter, rightParameter) {
			return false
		}
	}
	return true
}

func normalizeCLIParameters(source openapispec.SDKSurfaceOperation) ([]sourceParameter, error) {
	parameters := make([]sourceParameter, 0, len(source.Parameters))
	seen := make(map[string]bool, len(source.Parameters))
	for _, value := range source.Parameters {
		if value.Name == "" || seen[value.Name] {
			return nil, reject("duplicate-source-parameter", source.OperationID, "parameters", value.Name)
		}
		seen[value.Name] = true
		shape := ScalarString
		switch {
		case reflect.DeepEqual(value.Types, []string{"string"}) && len(value.ItemTypes) == 0:
		case reflect.DeepEqual(value.Types, []string{"array"}) && reflect.DeepEqual(value.ItemTypes, []string{"string"}):
			shape = StringArray
		default:
			return nil, reject("unsupported-cli-parameter-schema", source.OperationID, "parameters/"+value.Name, strings.Join(value.Types, ","))
		}
		parameters = append(parameters, sourceParameter{
			name: value.Name, description: canonicalDescription(value.Description), required: value.Required,
			shape: shape, minItems: cloneInt(value.MinItems), maxItems: cloneInt(value.MaxItems),
			constraints: StringConstraints{
				Format: value.StringConstraints.Format, AllowedValues: append([]string(nil), value.StringConstraints.AllowedValues...),
				MinLength: cloneInt(value.StringConstraints.MinLength), MaxLength: cloneInt(value.StringConstraints.MaxLength),
				DecimalMinimum: cloneInt(value.StringConstraints.DecimalMinimum), DecimalMaximum: cloneInt(value.StringConstraints.DecimalMaximum),
			},
		})
	}
	return parameters, nil
}

func primaryRepresentation(source openapispec.SDKSurfaceOperation) (Representation, error) {
	primary := make(map[Representation]bool)
	for _, response := range source.Responses {
		for _, media := range response.MediaTypes {
			if media.ContentTypeStatus != "inferred-from-documented-output-format" {
				continue
			}
			var representation Representation
			switch media.Name {
			case "application/json":
				representation = RepresentationJSON
			case "application/xml":
				representation = RepresentationXML
			case "application/zip":
				representation = RepresentationZIP
			default:
				return "", reject("unsupported-cli-response-media", source.OperationID, "responses", media.Name)
			}
			primary[representation] = true
		}
	}
	if len(primary) != 1 {
		return "", reject("ambiguous-cli-representation", source.OperationID, "responses", fmt.Sprint(primary))
	}
	for representation := range primary {
		return representation, nil
	}
	panic("unreachable")
}

func buildCLIProjections(logical []logicalOperation, batches []rustinterface.Batch) (CLIInterfaceModel, CLIDispatchModel, error) {
	interfaces := make(map[string]rustinterface.Operation, len(logical))
	interfaceFamilies := make(map[string]string, len(logical))
	seenFamilies := make(map[string]bool, len(batches))
	for _, batch := range batches {
		if batch.SchemaVersion != rustinterface.SchemaVersion || batch.Family == "" || len(batch.Operations) == 0 {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("cli-interface-header", "", "interface", batch.Family)
		}
		if seenFamilies[batch.Family] {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("duplicate-cli-interface-family", "", "interface", batch.Family)
		}
		seenFamilies[batch.Family] = true
		for _, operation := range batch.Operations {
			if operation.LogicalID == "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-interface-identity", "", "logical_id", batch.Family)
			}
			if _, exists := interfaces[operation.LogicalID]; exists {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("duplicate-cli-interface-operation", operation.LogicalID, "logical_id", operation.LogicalID)
			}
			interfaces[operation.LogicalID] = operation
			interfaceFamilies[operation.LogicalID] = batch.Family
		}
	}

	reserved := map[string]bool{
		"representation": true, "output": true, "connect-timeout-ms": true, "read-timeout-ms": true,
		"total-timeout-ms": true, "envelope-limit-bytes": true, "artifact-limit-bytes": true, "help": true, "version": true,
	}
	aliases := make(map[string]string, len(logical)*2)
	operations := make([]CLIOperation, 0, len(logical))
	dispatchOperations := make([]CLIDispatchOperation, 0, len(logical))
	seenLogical := make(map[string]bool, len(logical))
	for _, operation := range logical {
		seenLogical[operation.id] = true
		product, exists := interfaces[operation.id]
		if !exists {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-interface-operation", operation.id, "logicalOperationId", operation.id)
		}
		if product.RustModule == "" || product.RustInput == "" {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-dispatch-symbol", operation.id, "rust_module", product.RustModule)
		}
		name := product.CLICommand
		if !validCLIName(name) {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("invalid-cli-name", operation.id, "cli_command", name)
		}
		if product.CLIAlias != operation.id {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("stale-cli-alias", operation.id, "cli_alias", product.CLIAlias)
		}
		if product.LogicalID != operation.id || interfaceFamilies[operation.id] != operation.group {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("stale-cli-interface-identity", operation.id, "logical_id", product.LogicalID)
		}
		for _, alias := range []string{name, operation.id} {
			if previous := aliases[alias]; previous != "" && previous != operation.id {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("cli-name-collision", operation.id, "logicalOperationId", previous+" and "+operation.id+" share "+alias)
			}
			aliases[alias] = operation.id
		}

		description := canonicalDescription(operation.sources[0].Description)
		if description == "" {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-description", operation.id, "description", operation.id)
		}
		parameterDescriptions := make(map[string]string, len(operation.parameters))
		for _, parameter := range operation.parameters {
			if parameter.description == "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-parameter-description", operation.sources[0].OperationID, "parameters/"+parameter.name+"/description", operation.id)
			}
			parameterDescriptions[parameter.name] = parameter.description
		}
		for _, source := range operation.sources[1:] {
			if canonicalDescription(source.Description) != description {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("incompatible-cli-description", source.OperationID, "description", operation.id)
			}
			for _, parameter := range source.Parameters {
				if canonicalDescription(parameter.Description) != parameterDescriptions[parameter.Name] {
					return CLIInterfaceModel{}, CLIDispatchModel{}, reject("incompatible-cli-parameter-description", source.OperationID, "parameters/"+parameter.Name, operation.id)
				}
			}
		}

		expectedParameters := make(map[string]bool, len(operation.parameters))
		for _, parameter := range operation.parameters {
			expectedParameters[parameter.name] = true
		}
		bindings := make(map[string]rustinterface.Parameter, len(product.Parameters))
		for _, parameter := range product.Parameters {
			if parameter.OpenAPIName == "" || bindings[parameter.OpenAPIName].OpenAPIName != "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("duplicate-cli-parameter-mapping", operation.id, "parameters", parameter.OpenAPIName)
			}
			if !expectedParameters[parameter.OpenAPIName] {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("orphan-cli-parameter-mapping", operation.id, "parameters", parameter.OpenAPIName)
			}
			bindings[parameter.OpenAPIName] = parameter
		}
		flags := make(map[string]string, len(operation.parameters))
		parameters := make([]CLIParameter, 0, len(operation.parameters))
		dispatchParameters := make([]CLIDispatchParameter, 0, len(operation.parameters))
		for _, parameter := range operation.parameters {
			binding, exists := bindings[parameter.name]
			if !exists {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-parameter-mapping", operation.id, "parameters", parameter.name)
			}
			if binding.RustName == "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-dispatch-symbol", operation.id, "parameters/"+parameter.name, "rust_name")
			}
			flag := binding.CLIFlag
			if !validCLIName(flag) {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("invalid-cli-flag", operation.id, "parameters/"+parameter.name, flag)
			}
			if reserved[flag] {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("reserved-cli-flag", operation.id, "parameters/"+parameter.name, flag)
			}
			if previous := flags[flag]; previous != "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("cli-flag-collision", operation.id, "parameters/"+parameter.name, previous+" and "+parameter.name)
			}
			flags[flag] = parameter.name
			parameters = append(parameters, CLIParameter{
				Flag: flag, SourceName: parameter.name, Description: parameterDescriptions[parameter.name], Required: parameter.required,
				Shape: parameter.shape, MinItems: cloneInt(parameter.minItems), MaxItems: cloneInt(parameter.maxItems), Constraints: parameter.constraints,
			})
			dispatchParameters = append(dispatchParameters, CLIDispatchParameter{ArgumentID: parameter.name, RustField: binding.RustName, Required: parameter.required, Shape: parameter.shape})
		}

		structuredCount, hasBinary := 0, false
		for _, variant := range operation.variants {
			if variant.representation == RepresentationZIP {
				hasBinary = true
			} else {
				structuredCount++
			}
		}
		if hasBinary && structuredCount != 0 {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("mixed-cli-representation-kinds", operation.id, "variants", "structured and ZIP variants require incompatible CLI output contracts")
		}
		expectedPhysical := make(map[string]physicalVariant, len(operation.variants))
		for _, variant := range operation.variants {
			expectedPhysical[variant.operationID] = variant
		}
		physicalBindings := make(map[string]rustinterface.Physical, len(product.Physical))
		for _, binding := range product.Physical {
			if binding.OperationID == "" || physicalBindings[binding.OperationID].OperationID != "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("duplicate-cli-physical-mapping", operation.id, "physical", binding.OperationID)
			}
			if _, exists := expectedPhysical[binding.OperationID]; !exists {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("orphan-cli-physical-mapping", operation.id, "physical", binding.OperationID)
			}
			if binding.RustMethod == "" || binding.RustResponse == "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-dispatch-symbol", operation.id, "physical/"+binding.OperationID, "rust_method or rust_response")
			}
			physicalBindings[binding.OperationID] = binding
		}
		representations := make([]CLIRepresentation, 0, len(operation.variants))
		dispatchRepresentations := make([]CLIDispatchRepresentation, 0, len(operation.variants))
		for _, variant := range operation.variants {
			binding, exists := physicalBindings[variant.operationID]
			if !exists {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-physical-mapping", operation.id, "physical", variant.operationID)
			}
			shape := CLIResponseShape{Kind: "structured_source"}
			if variant.representation == RepresentationZIP {
				shape = CLIResponseShape{Kind: "binary"}
			}
			representation := CLIRepresentation{
				Name: variant.representation, PhysicalID: variant.operationID,
				Selector: variant.representation != RepresentationZIP && structuredCount > 1, ResponseShape: shape,
			}
			testArgv, err := cliTestArgv(parameters, representation)
			if err != nil {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-test-invocation", operation.id, "variants/"+variant.operationID, err.Error())
			}
			representations = append(representations, representation)
			dispatchRepresentations = append(dispatchRepresentations, CLIDispatchRepresentation{
				Name: variant.representation, PhysicalID: variant.operationID, PrepareMethod: binding.RustMethod,
				ResponseType: "opendart::operations::" + product.RustModule + "::" + binding.RustResponse, TestArgv: testArgv,
			})
		}
		order := map[Representation]int{RepresentationJSON: 0, RepresentationXML: 1, RepresentationZIP: 2}
		sort.Slice(representations, func(i, j int) bool { return order[representations[i].Name] < order[representations[j].Name] })
		sort.Slice(dispatchRepresentations, func(i, j int) bool {
			return order[dispatchRepresentations[i].Name] < order[dispatchRepresentations[j].Name]
		})
		operations = append(operations, CLIOperation{
			Name: name, LogicalID: operation.id, Group: operation.group, APIID: operation.apiID, GuideURL: operation.guideURL,
			Description: description, Parameters: parameters, Representations: representations,
		})
		dispatchOperations = append(dispatchOperations, CLIDispatchOperation{
			Name: name, LogicalID: operation.id, RustModule: product.RustModule, RustInput: product.RustInput,
			Parameters: dispatchParameters, Representations: dispatchRepresentations,
		})
	}
	for logicalID := range interfaces {
		if !seenLogical[logicalID] {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("orphan-cli-interface-operation", logicalID, "logical_id", logicalID)
		}
	}
	if len(interfaces) != len(logical) {
		return CLIInterfaceModel{}, CLIDispatchModel{}, reject("cli-interface-coverage", "", "logical_id", fmt.Sprintf("mapped %d of %d", len(interfaces), len(logical)))
	}
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Name == operations[j].Name {
			return operations[i].LogicalID < operations[j].LogicalID
		}
		return operations[i].Name < operations[j].Name
	})
	sort.Slice(dispatchOperations, func(i, j int) bool {
		if dispatchOperations[i].Name == dispatchOperations[j].Name {
			return dispatchOperations[i].LogicalID < dispatchOperations[j].LogicalID
		}
		return dispatchOperations[i].Name < dispatchOperations[j].Name
	})
	cliInterface := CLIInterfaceModel{SchemaVersion: CLIInterfaceProjectionSchemaVersion, Operations: operations}
	checksum, err := projectionChecksum(cliInterface, "CLI interface projection")
	if err != nil {
		return CLIInterfaceModel{}, CLIDispatchModel{}, err
	}
	cliInterface.Checksum = checksum
	cliDispatch := CLIDispatchModel{SchemaVersion: CLIDispatchProjectionSchemaVersion, Operations: dispatchOperations}
	checksum, err = projectionChecksum(cliDispatch, "CLI dispatch projection")
	if err != nil {
		return CLIInterfaceModel{}, CLIDispatchModel{}, err
	}
	cliDispatch.Checksum = checksum
	return cliInterface, cliDispatch, nil
}

func cliTestArgv(parameters []CLIParameter, representation CLIRepresentation) ([]string, error) {
	arguments := make([]string, 0, len(parameters)*2+4)
	for _, parameter := range parameters {
		if !parameter.Required {
			continue
		}
		value, err := cliTestValue(parameter)
		if err != nil {
			return nil, fmt.Errorf("parameter %s: %w", parameter.SourceName, err)
		}
		count := int64(1)
		if parameter.Shape == StringArray && parameter.MinItems != nil {
			count = *parameter.MinItems
		}
		if count < 1 || count > 4096 {
			return nil, fmt.Errorf("unsupported required item count %d", count)
		}
		for range count {
			arguments = append(arguments, "--"+parameter.Flag, value)
		}
	}
	if representation.Selector {
		arguments = append(arguments, "--representation", string(representation.Name))
	}
	if representation.Name == RepresentationZIP {
		arguments = append(arguments, "--output", "<generated-test-output>")
	}
	return arguments, nil
}

func cliTestValue(parameter CLIParameter) (string, error) {
	constraints := parameter.Constraints
	if len(constraints.AllowedValues) != 0 {
		return constraints.AllowedValues[0], nil
	}
	if constraints.DecimalMinimum != nil {
		return strconv.FormatInt(*constraints.DecimalMinimum, 10), nil
	}
	switch constraints.Format {
	case "opendart-corp-code":
		return "00126380", nil
	case "opendart-date":
		return "20150101", nil
	case "opendart-year":
		return "2015", nil
	}
	length := int64(len("fixture"))
	if constraints.MinLength != nil {
		length = *constraints.MinLength
	} else if constraints.MaxLength != nil && length > *constraints.MaxLength {
		length = *constraints.MaxLength
	}
	if length < 1 || length > 4096 {
		return "", fmt.Errorf("unsupported fixture length %d", length)
	}
	return strings.Repeat("a", int(length)), nil
}

func validCLIName(value string) bool {
	if value == "" {
		return false
	}
	for _, part := range strings.Split(value, "-") {
		if part == "" {
			return false
		}
		for _, character := range part {
			if character < 'a' || character > 'z' {
				if character < '0' || character > '9' {
					return false
				}
			}
		}
	}
	return true
}

func projectionChecksum(value any, label string) (string, error) {
	encoded, err := json.Marshal(value)
	if err != nil {
		return "", fmt.Errorf("encode %s: %w", label, err)
	}
	digest := sha256.Sum256(encoded)
	return hex.EncodeToString(digest[:]), nil
}

func canonicalDescription(value string) string {
	return strings.TrimSpace(strings.ReplaceAll(strings.ReplaceAll(value, "\r\n", "\n"), "\r", "\n"))
}

func cloneInt(value *int64) *int64 {
	if value == nil {
		return nil
	}
	cloned := *value
	return &cloned
}

func reject(rule, operation, location, detail string) *Error {
	return &Error{Rule: rule, Operation: operation, Location: location, Detail: detail}
}
