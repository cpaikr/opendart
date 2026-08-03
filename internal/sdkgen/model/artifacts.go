package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strconv"
	"strings"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/rustinterface"
)

const (
	// SemanticSchemaVersion identifies the combined normalized artifact model.
	SemanticSchemaVersion uint32 = 3
	// CLIInterfaceProjectionSchemaVersion identifies public grammar and discovery.
	CLIInterfaceProjectionSchemaVersion uint32 = 1
	// CLIDispatchProjectionSchemaVersion identifies private generated-SDK wiring.
	CLIDispatchProjectionSchemaVersion uint32 = 1
)

// ArtifactSet is one normalized build with independently identified projections.
type ArtifactSet struct {
	Semantic     SemanticModel
	SDK          Model
	CLIInterface CLIInterfaceModel
	CLIDispatch  CLIDispatchModel
}

// SemanticModel records the complete normalized facts behind both Rust products.
type SemanticModel struct {
	SchemaVersion uint32            `json:"schemaVersion"`
	Checksum      string            `json:"checksum,omitempty"`
	SDK           Model             `json:"sdk"`
	CLIInterface  CLIInterfaceModel `json:"cliInterface"`
	CLIDispatch   CLIDispatchModel  `json:"cliDispatch"`
}

// CLIInterfaceModel is the Rust-symbol-free public grammar and discovery input.
type CLIInterfaceModel struct {
	SchemaVersion uint32         `json:"schemaVersion"`
	Checksum      string         `json:"checksum,omitempty"`
	Operations    []CLIOperation `json:"operations"`
}

// CLIOperation contains only reviewed CLI names and canonical protocol facts.
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

// CLIParameter describes one generated operation-specific flag.
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

// CLIRepresentation describes one canonical physical representation publicly.
type CLIRepresentation struct {
	Name          Representation   `json:"name"`
	PhysicalID    string           `json:"physicalId"`
	Selector      bool             `json:"selector"`
	ResponseShape CLIResponseShape `json:"responseShape"`
}

// CLIDispatchModel is the separately identified private generated-SDK adapter.
type CLIDispatchModel struct {
	SchemaVersion uint32                 `json:"schemaVersion"`
	Checksum      string                 `json:"checksum,omitempty"`
	Operations    []CLIDispatchOperation `json:"operations"`
}

// CLIDispatchOperation binds one reviewed command to current private SDK symbols.
type CLIDispatchOperation struct {
	Name            string                      `json:"name"`
	LogicalID       string                      `json:"logicalId"`
	SDKInputType    string                      `json:"sdkInputType"`
	Parameters      []CLIDispatchParameter      `json:"parameters"`
	Representations []CLIDispatchRepresentation `json:"representations"`
}

// CLIDispatchParameter binds one hidden parser identity to one SDK input field.
type CLIDispatchParameter struct {
	ArgumentID string         `json:"argumentId"`
	SDKField   string         `json:"sdkField"`
	Required   bool           `json:"required"`
	Shape      ParameterShape `json:"shape"`
}

// CLIDispatchRepresentation binds protocol identity to current SDK preparation.
type CLIDispatchRepresentation struct {
	Name          Representation `json:"name"`
	PhysicalID    string         `json:"physicalId"`
	PrepareMethod string         `json:"prepareMethod"`
	ResponseType  string         `json:"responseType"`
	TestArgv      []string       `json:"testArgv"`
}

// CLIResponseShape is the deliberately coarse public output category.
type CLIResponseShape struct {
	Kind string `json:"kind"`
}

// BuildArtifacts validates and builds the semantic, SDK, and CLI identities together.
func BuildArtifacts(surface openapispec.SDKSurface, batches []rustinterface.Batch) (ArtifactSet, error) {
	sdk, err := Build(surface)
	if err != nil {
		return ArtifactSet{}, err
	}
	cliInterface, cliDispatch, err := buildCLIProjections(surface, sdk, batches)
	if err != nil {
		return ArtifactSet{}, err
	}
	semantic := SemanticModel{
		SchemaVersion: SemanticSchemaVersion,
		SDK:           sdk, CLIInterface: cliInterface, CLIDispatch: cliDispatch,
	}
	semantic.SDK.Checksum = ""
	semantic.CLIInterface.Checksum = ""
	semantic.CLIDispatch.Checksum = ""
	semantic.Checksum, err = projectionChecksum(semantic, "semantic model")
	if err != nil {
		return ArtifactSet{}, err
	}
	semantic.SDK = sdk
	semantic.CLIInterface = cliInterface
	semantic.CLIDispatch = cliDispatch
	return ArtifactSet{
		Semantic: semantic, SDK: sdk,
		CLIInterface: cliInterface, CLIDispatch: cliDispatch,
	}, nil
}

func buildCLIProjections(
	surface openapispec.SDKSurface,
	sdk Model,
	batches []rustinterface.Batch,
) (CLIInterfaceModel, CLIDispatchModel, error) {
	sources := make(map[string][]openapispec.SDKSurfaceOperation)
	for _, operation := range surface.Operations {
		sources[operation.LogicalOperationID] = append(sources[operation.LogicalOperationID], operation)
	}
	physical := make(map[string]PhysicalOperation, len(sdk.Physical))
	for _, operation := range sdk.Physical {
		physical[operation.OperationID] = operation
	}
	interfaces := make(map[string]rustinterface.Operation, len(sdk.Logical))
	interfaceFamilies := make(map[string]string, len(sdk.Logical))
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
		"representation": true, "output": true,
		"connect-timeout-ms": true, "read-timeout-ms": true,
		"total-timeout-ms": true, "envelope-limit-bytes": true,
		"artifact-limit-bytes": true, "help": true, "version": true,
	}
	aliases := make(map[string]string, len(sdk.Logical)*2)
	operations := make([]CLIOperation, 0, len(sdk.Logical))
	dispatchOperations := make([]CLIDispatchOperation, 0, len(sdk.Logical))
	for _, operation := range sdk.Logical {
		product, exists := interfaces[operation.ID]
		if !exists {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-interface-operation", operation.ID, "logicalOperationId", operation.ID)
		}
		name := product.CLICommand
		if !validCLIName(name) {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("invalid-cli-name", operation.ID, "cli_command", name)
		}
		if product.CLIAlias != operation.ID {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("stale-cli-alias", operation.ID, "cli_alias", product.CLIAlias)
		}
		if product.LogicalID != operation.ID || interfaceFamilies[operation.ID] != strings.ToUpper(operation.Group) {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("stale-cli-interface-identity", operation.ID, "logical_id", product.LogicalID)
		}
		for _, alias := range []string{name, operation.ID} {
			if previous := aliases[alias]; previous != "" && previous != operation.ID {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("cli-name-collision", operation.ID, "logicalOperationId", previous+" and "+operation.ID+" share "+alias)
			}
			aliases[alias] = operation.ID
		}

		logicalSources := sources[operation.ID]
		if len(logicalSources) == 0 {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-source", operation.ID, "logicalOperationId", operation.ID)
		}
		description := canonicalDescription(logicalSources[0].Description)
		if description == "" {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-description", operation.ID, "description", operation.ID)
		}
		parameterDescriptions := make(map[string]string, len(logicalSources[0].Parameters))
		for _, parameter := range logicalSources[0].Parameters {
			description := canonicalDescription(parameter.Description)
			if description == "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-parameter-description", logicalSources[0].OperationID, "parameters/"+parameter.Name+"/description", operation.ID)
			}
			parameterDescriptions[parameter.Name] = description
		}
		for _, source := range logicalSources[1:] {
			if canonicalDescription(source.Description) != description {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("incompatible-cli-description", source.OperationID, "description", operation.ID)
			}
			for _, parameter := range source.Parameters {
				if canonicalDescription(parameter.Description) != parameterDescriptions[parameter.Name] {
					return CLIInterfaceModel{}, CLIDispatchModel{}, reject("incompatible-cli-parameter-description", source.OperationID, "parameters/"+parameter.Name, operation.ID)
				}
			}
		}

		expectedParameters := make(map[string]bool, len(operation.Parameters))
		for _, parameter := range operation.Parameters {
			expectedParameters[parameter.WireName] = true
		}
		bindings := make(map[string]rustinterface.Parameter, len(product.Parameters))
		for _, parameter := range product.Parameters {
			if parameter.OpenAPIName == "" || bindings[parameter.OpenAPIName].OpenAPIName != "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("duplicate-cli-parameter-mapping", operation.ID, "parameters", parameter.OpenAPIName)
			}
			if !expectedParameters[parameter.OpenAPIName] {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("orphan-cli-parameter-mapping", operation.ID, "parameters", parameter.OpenAPIName)
			}
			bindings[parameter.OpenAPIName] = parameter
		}
		flags := make(map[string]string, len(operation.Parameters))
		parameters := make([]CLIParameter, 0, len(operation.Parameters))
		dispatchParameters := make([]CLIDispatchParameter, 0, len(operation.Parameters))
		for _, parameter := range operation.Parameters {
			binding, exists := bindings[parameter.WireName]
			if !exists {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-parameter-mapping", operation.ID, "parameters", parameter.WireName)
			}
			flag := binding.CLIFlag
			if !validCLIName(flag) {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("invalid-cli-flag", operation.ID, "parameters/"+parameter.WireName, flag)
			}
			if reserved[flag] {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("reserved-cli-flag", operation.ID, "parameters/"+parameter.WireName, flag)
			}
			if previous := flags[flag]; previous != "" {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("cli-flag-collision", operation.ID, "parameters/"+parameter.WireName, previous+" and "+parameter.WireName)
			}
			flags[flag] = parameter.WireName
			parameters = append(parameters, CLIParameter{
				Flag: flag, SourceName: parameter.WireName,
				Description: parameterDescriptions[parameter.WireName], Required: parameter.Required,
				Shape: parameter.Shape, MinItems: parameter.MinItems, MaxItems: parameter.MaxItems,
				Constraints: parameter.Constraints,
			})
			dispatchParameters = append(dispatchParameters, CLIDispatchParameter{
				ArgumentID: parameter.WireName, SDKField: parameter.RustName,
				Required: parameter.Required, Shape: parameter.Shape,
			})
		}
		structuredCount := 0
		hasBinary := false
		for _, variant := range operation.Variants {
			if variant.Representation == RepresentationZIP {
				hasBinary = true
			} else {
				structuredCount++
			}
		}
		if hasBinary && structuredCount != 0 {
			return CLIInterfaceModel{}, CLIDispatchModel{}, reject("mixed-cli-representation-kinds", operation.ID, "variants", "structured and ZIP variants require incompatible CLI output contracts")
		}
		expectedPhysical := make(map[string]bool, len(operation.Variants))
		for _, variant := range operation.Variants {
			expectedPhysical[variant.OperationID] = true
		}
		physicalBindings := make(map[string]bool, len(product.Physical))
		for _, binding := range product.Physical {
			if binding.OperationID == "" || physicalBindings[binding.OperationID] {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("duplicate-cli-physical-mapping", operation.ID, "physical", binding.OperationID)
			}
			if !expectedPhysical[binding.OperationID] {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("orphan-cli-physical-mapping", operation.ID, "physical", binding.OperationID)
			}
			physicalBindings[binding.OperationID] = true
		}
		representations := make([]CLIRepresentation, 0, len(operation.Variants))
		dispatchRepresentations := make([]CLIDispatchRepresentation, 0, len(operation.Variants))
		for _, variant := range operation.Variants {
			if !physicalBindings[variant.OperationID] {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-physical-mapping", operation.ID, "physical", variant.OperationID)
			}
			physicalOperation, exists := physical[variant.OperationID]
			if !exists || physicalOperation.LogicalID != operation.ID || physicalOperation.PrimaryRepresentation != variant.Representation {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("invalid-cli-representation", operation.ID, "variants", variant.OperationID)
			}
			responseType := "opendart::BinaryReply<opendart::BodyStream>"
			shape := CLIResponseShape{Kind: "binary"}
			if variant.Representation != RepresentationZIP {
				responseType = "opendart::responses::" + responseRootName(operation.RustName, variant.Representation)
				if _, err := primaryShape(physicalOperation); err != nil {
					return CLIInterfaceModel{}, CLIDispatchModel{}, err
				}
				shape = CLIResponseShape{Kind: "structured_source"}
			}
			representation := CLIRepresentation{
				Name: variant.Representation, PhysicalID: variant.OperationID,
				Selector:      variant.Representation != RepresentationZIP && structuredCount > 1,
				ResponseShape: shape,
			}
			testArgv, err := cliTestArgv(parameters, representation)
			if err != nil {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("missing-cli-test-invocation", operation.ID, "variants/"+variant.OperationID, err.Error())
			}
			representations = append(representations, representation)
			dispatchRepresentations = append(dispatchRepresentations, CLIDispatchRepresentation{
				Name: variant.Representation, PhysicalID: variant.OperationID,
				PrepareMethod: "prepare_" + string(variant.Representation),
				ResponseType:  responseType, TestArgv: testArgv,
			})
		}
		sort.Slice(representations, func(i, j int) bool {
			order := map[Representation]int{RepresentationJSON: 0, RepresentationXML: 1, RepresentationZIP: 2}
			return order[representations[i].Name] < order[representations[j].Name]
		})
		sort.Slice(dispatchRepresentations, func(i, j int) bool {
			order := map[Representation]int{RepresentationJSON: 0, RepresentationXML: 1, RepresentationZIP: 2}
			return order[dispatchRepresentations[i].Name] < order[dispatchRepresentations[j].Name]
		})
		operations = append(operations, CLIOperation{
			Name: name, LogicalID: operation.ID, Group: strings.ToUpper(operation.Group),
			APIID: operation.APIID, GuideURL: operation.GuideURL, Description: description,
			Parameters: parameters, Representations: representations,
		})
		dispatchOperations = append(dispatchOperations, CLIDispatchOperation{
			Name: name, LogicalID: operation.ID, SDKInputType: operation.RustName,
			Parameters: dispatchParameters, Representations: dispatchRepresentations,
		})
	}
	if len(interfaces) != len(sdk.Logical) {
		for logicalID := range interfaces {
			if _, exists := sources[logicalID]; !exists {
				return CLIInterfaceModel{}, CLIDispatchModel{}, reject("orphan-cli-interface-operation", logicalID, "logical_id", logicalID)
			}
		}
		return CLIInterfaceModel{}, CLIDispatchModel{}, reject("cli-interface-coverage", "", "logical_id", fmt.Sprintf("mapped %d of %d", len(interfaces), len(sdk.Logical)))
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
	interfaceProjection := CLIInterfaceModel{
		SchemaVersion: CLIInterfaceProjectionSchemaVersion,
		Operations:    operations,
	}
	checksum, err := projectionChecksum(interfaceProjection, "CLI interface projection")
	if err != nil {
		return CLIInterfaceModel{}, CLIDispatchModel{}, err
	}
	interfaceProjection.Checksum = checksum
	dispatchProjection := CLIDispatchModel{
		SchemaVersion: CLIDispatchProjectionSchemaVersion,
		Operations:    dispatchOperations,
	}
	checksum, err = projectionChecksum(dispatchProjection, "CLI dispatch projection")
	if err != nil {
		return CLIInterfaceModel{}, CLIDispatchModel{}, err
	}
	dispatchProjection.Checksum = checksum
	return interfaceProjection, dispatchProjection, nil
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

func primaryShape(operation PhysicalOperation) (ResponseShape, error) {
	mediaName := map[Representation]string{RepresentationJSON: "application/json", RepresentationXML: "application/xml"}[operation.PrimaryRepresentation]
	var primary *ResponseShape
	for _, response := range operation.Responses {
		for _, media := range response.Media {
			if media.Name == mediaName && media.ContentTypeStatus == "inferred-from-documented-output-format" {
				if primary != nil {
					return ResponseShape{}, reject("ambiguous-cli-response-shape", operation.OperationID, "responses", mediaName)
				}
				shape := media.Shape
				primary = &shape
			}
		}
	}
	if primary != nil {
		return *primary, nil
	}
	return ResponseShape{}, reject("missing-cli-response-shape", operation.OperationID, "responses", string(operation.PrimaryRepresentation))
}

func responseRootName(input string, representation Representation) string {
	suffix := map[Representation]string{RepresentationJSON: "Json", RepresentationXML: "Xml"}[representation]
	return input + suffix + "Response"
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
