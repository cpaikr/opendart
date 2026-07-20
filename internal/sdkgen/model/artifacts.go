package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

const (
	// SemanticSchemaVersion identifies the combined normalized artifact model.
	SemanticSchemaVersion uint32 = 1
	// CLIProjectionSchemaVersion identifies the generated CLI contract projection.
	CLIProjectionSchemaVersion uint32 = 1
)

// ArtifactSet is one normalized build with independently identified projections.
type ArtifactSet struct {
	Semantic SemanticModel
	SDK      Model
	CLI      CLIModel
}

// SemanticModel records the complete normalized facts behind both Rust products.
type SemanticModel struct {
	SchemaVersion uint32   `json:"schemaVersion"`
	Checksum      string   `json:"checksum,omitempty"`
	SDK           Model    `json:"sdk"`
	CLI           CLIModel `json:"cli"`
}

// CLIModel is the deterministic projection consumed only by the public CLI renderer.
type CLIModel struct {
	SchemaVersion uint32         `json:"schemaVersion"`
	Checksum      string         `json:"checksum,omitempty"`
	Operations    []CLIOperation `json:"operations"`
}

// CLIOperation contains generated command, discovery, and SDK dispatch facts.
type CLIOperation struct {
	Name            string              `json:"name"`
	LogicalID       string              `json:"logicalId"`
	Group           string              `json:"group"`
	APIID           string              `json:"apiId"`
	GuideURL        string              `json:"guideUrl"`
	Description     string              `json:"description"`
	SDKInputType    string              `json:"sdkInputType"`
	Parameters      []CLIParameter      `json:"parameters"`
	Representations []CLIRepresentation `json:"representations"`
}

// CLIParameter describes one generated operation-specific flag.
type CLIParameter struct {
	Flag        string         `json:"flag"`
	WireName    string         `json:"wireName"`
	SDKField    string         `json:"sdkField"`
	Description string         `json:"description"`
	Required    bool           `json:"required"`
	Shape       ParameterShape `json:"shape"`
	MinItems    *int64         `json:"minItems,omitempty"`
	MaxItems    *int64         `json:"maxItems,omitempty"`
}

// CLIRepresentation binds one public selector to its SDK preparation surface.
type CLIRepresentation struct {
	Name          Representation   `json:"name"`
	PhysicalID    string           `json:"physicalId"`
	PrepareMethod string           `json:"prepareMethod"`
	ResponseType  string           `json:"responseType"`
	Selector      bool             `json:"selector"`
	ResponseShape CLIResponseShape `json:"responseShape"`
}

// CLIResponseShape is the recursive discovery view of an SDK response.
type CLIResponseShape struct {
	Kind             string             `json:"kind"`
	AdditionalFields bool               `json:"additionalFields,omitempty"`
	Fields           []CLIResponseField `json:"fields,omitempty"`
	Items            *CLIResponseShape  `json:"items,omitempty"`
}

// CLIResponseField describes one source-named field in a generated response.
type CLIResponseField struct {
	Name        string           `json:"name"`
	Required    bool             `json:"required"`
	Description string           `json:"description,omitempty"`
	Shape       CLIResponseShape `json:"shape"`
}

// BuildArtifacts validates and builds the semantic, SDK, and CLI identities together.
func BuildArtifacts(surface openapispec.SDKSurface) (ArtifactSet, error) {
	sdk, err := Build(surface)
	if err != nil {
		return ArtifactSet{}, err
	}
	cli, err := buildCLIProjection(surface, sdk)
	if err != nil {
		return ArtifactSet{}, err
	}
	semantic := SemanticModel{SchemaVersion: SemanticSchemaVersion, SDK: sdk, CLI: cli}
	semantic.SDK.Checksum = ""
	semantic.CLI.Checksum = ""
	semantic.Checksum, err = projectionChecksum(semantic, "semantic model")
	if err != nil {
		return ArtifactSet{}, err
	}
	semantic.SDK = sdk
	semantic.CLI = cli
	return ArtifactSet{Semantic: semantic, SDK: sdk, CLI: cli}, nil
}

func buildCLIProjection(surface openapispec.SDKSurface, sdk Model) (CLIModel, error) {
	sources := make(map[string][]openapispec.SDKSurfaceOperation)
	for _, operation := range surface.Operations {
		sources[operation.LogicalOperationID] = append(sources[operation.LogicalOperationID], operation)
	}
	physical := make(map[string]PhysicalOperation, len(sdk.Physical))
	for _, operation := range sdk.Physical {
		physical[operation.OperationID] = operation
	}

	reserved := map[string]bool{
		"representation": true, "output": true,
		"connect-timeout-ms": true, "read-timeout-ms": true,
		"total-timeout-ms": true, "envelope-limit-bytes": true,
		"artifact-limit-bytes": true, "help": true, "version": true,
	}
	aliases := make(map[string]string, len(sdk.Logical)*2)
	operations := make([]CLIOperation, 0, len(sdk.Logical))
	for _, operation := range sdk.Logical {
		name := kebabName(operation.RustName)
		if name == "" {
			return CLIModel{}, reject("invalid-cli-name", operation.ID, "rustName", operation.RustName)
		}
		for _, alias := range []string{name, operation.ID} {
			if previous := aliases[alias]; previous != "" && previous != operation.ID {
				return CLIModel{}, reject("cli-name-collision", operation.ID, "logicalOperationId", previous+" and "+operation.ID+" share "+alias)
			}
			aliases[alias] = operation.ID
		}

		logicalSources := sources[operation.ID]
		if len(logicalSources) == 0 {
			return CLIModel{}, reject("missing-cli-source", operation.ID, "logicalOperationId", operation.ID)
		}
		description := canonicalDescription(logicalSources[0].Description)
		if description == "" {
			return CLIModel{}, reject("missing-cli-description", operation.ID, "description", operation.ID)
		}
		parameterDescriptions := make(map[string]string, len(logicalSources[0].Parameters))
		for _, parameter := range logicalSources[0].Parameters {
			description := canonicalDescription(parameter.Description)
			if description == "" {
				return CLIModel{}, reject("missing-cli-parameter-description", logicalSources[0].OperationID, "parameters/"+parameter.Name+"/description", operation.ID)
			}
			parameterDescriptions[parameter.Name] = description
		}
		for _, source := range logicalSources[1:] {
			if canonicalDescription(source.Description) != description {
				return CLIModel{}, reject("incompatible-cli-description", source.OperationID, "description", operation.ID)
			}
			for _, parameter := range source.Parameters {
				if canonicalDescription(parameter.Description) != parameterDescriptions[parameter.Name] {
					return CLIModel{}, reject("incompatible-cli-parameter-description", source.OperationID, "parameters/"+parameter.Name, operation.ID)
				}
			}
		}

		flags := make(map[string]string, len(operation.Parameters))
		parameters := make([]CLIParameter, 0, len(operation.Parameters))
		for _, parameter := range operation.Parameters {
			flag := strings.ReplaceAll(parameter.RustName, "_", "-")
			if reserved[flag] {
				return CLIModel{}, reject("reserved-cli-flag", operation.ID, "parameters/"+parameter.WireName, flag)
			}
			if previous := flags[flag]; previous != "" {
				return CLIModel{}, reject("cli-flag-collision", operation.ID, "parameters/"+parameter.WireName, previous+" and "+parameter.WireName)
			}
			flags[flag] = parameter.WireName
			parameters = append(parameters, CLIParameter{
				Flag: flag, WireName: parameter.WireName, SDKField: parameter.RustName,
				Description: parameterDescriptions[parameter.WireName], Required: parameter.Required,
				Shape: parameter.Shape, MinItems: parameter.MinItems, MaxItems: parameter.MaxItems,
			})
		}

		structuredCount := 0
		for _, variant := range operation.Variants {
			if variant.Representation != RepresentationZIP {
				structuredCount++
			}
		}
		representations := make([]CLIRepresentation, 0, len(operation.Variants))
		for _, variant := range operation.Variants {
			physicalOperation, exists := physical[variant.OperationID]
			if !exists || physicalOperation.LogicalID != operation.ID || physicalOperation.PrimaryRepresentation != variant.Representation {
				return CLIModel{}, reject("invalid-cli-representation", operation.ID, "variants", variant.OperationID)
			}
			responseType := "opendart::BinaryReply<opendart::BodyStream>"
			shape := CLIResponseShape{Kind: "binary"}
			if variant.Representation != RepresentationZIP {
				responseType = "opendart::responses::" + responseRootName(operation.RustName, variant.Representation)
				primary, err := primaryShape(physicalOperation)
				if err != nil {
					return CLIModel{}, err
				}
				shape, err = projectResponseShape(primary, physicalOperation.OperationID, "responses/"+string(variant.Representation))
				if err != nil {
					return CLIModel{}, err
				}
			}
			representations = append(representations, CLIRepresentation{
				Name: variant.Representation, PhysicalID: variant.OperationID,
				PrepareMethod: "prepare_" + string(variant.Representation), ResponseType: responseType,
				Selector:      variant.Representation != RepresentationZIP && structuredCount > 1,
				ResponseShape: shape,
			})
		}
		sort.Slice(representations, func(i, j int) bool {
			order := map[Representation]int{RepresentationJSON: 0, RepresentationXML: 1, RepresentationZIP: 2}
			return order[representations[i].Name] < order[representations[j].Name]
		})
		operations = append(operations, CLIOperation{
			Name: name, LogicalID: operation.ID, Group: strings.ToUpper(operation.Group),
			APIID: operation.APIID, GuideURL: operation.GuideURL, Description: description,
			SDKInputType: operation.RustName, Parameters: parameters, Representations: representations,
		})
	}
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Name == operations[j].Name {
			return operations[i].LogicalID < operations[j].LogicalID
		}
		return operations[i].Name < operations[j].Name
	})
	projection := CLIModel{SchemaVersion: CLIProjectionSchemaVersion, Operations: operations}
	checksum, err := projectionChecksum(projection, "CLI projection")
	if err != nil {
		return CLIModel{}, err
	}
	projection.Checksum = checksum
	return projection, nil
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

func projectResponseShape(shape ResponseShape, operation, location string) (CLIResponseShape, error) {
	switch shape.Kind {
	case "binary":
		return CLIResponseShape{Kind: "binary"}, nil
	case "opaque":
		return CLIResponseShape{Kind: "source_value"}, nil
	case "string":
		if shape.OpenStatus {
			return CLIResponseShape{Kind: "source_status"}, nil
		}
		return CLIResponseShape{}, reject("unsupported-cli-response-shape", operation, location, "project CLI response shape: ordinary string kind has no public discovery representation")
	case "array":
		if shape.Items == nil {
			return CLIResponseShape{}, reject("invalid-cli-response-shape", operation, location, "project CLI response shape: array has no items")
		}
		items, err := projectResponseShape(*shape.Items, operation, location+"/items")
		if err != nil {
			return CLIResponseShape{}, err
		}
		return CLIResponseShape{Kind: "array", Items: &items}, nil
	case "object":
		fields := make([]CLIResponseField, 0, len(shape.Properties))
		for _, property := range shape.Properties {
			child, err := projectResponseShape(property.Shape, operation, location+"/properties/"+property.Name)
			if err != nil {
				return CLIResponseShape{}, err
			}
			fields = append(fields, CLIResponseField{
				Name: property.Name, Required: responsePropertyRequired(shape, property.Name),
				Description: strings.TrimSpace(property.Shape.Description), Shape: child,
			})
		}
		return CLIResponseShape{Kind: "object", AdditionalFields: shape.AdditionalPropertiesPolicy == "allowed", Fields: fields}, nil
	default:
		return CLIResponseShape{}, reject("unsupported-cli-response-shape", operation, location, fmt.Sprintf("project CLI response shape: unsupported kind %q", shape.Kind))
	}
}

func responsePropertyRequired(shape ResponseShape, name string) bool {
	for _, required := range shape.Required {
		if required == name {
			return true
		}
	}
	return false
}

func responseRootName(input string, representation Representation) string {
	suffix := map[Representation]string{RepresentationJSON: "Json", RepresentationXML: "Xml"}[representation]
	return input + suffix + "Response"
}

func kebabName(value string) string {
	runes := []rune(value)
	var output strings.Builder
	for index, character := range runes {
		if unicode.IsUpper(character) && index > 0 {
			previous := runes[index-1]
			nextLower := index+1 < len(runes) && unicode.IsLower(runes[index+1])
			if unicode.IsLower(previous) || unicode.IsDigit(previous) || (unicode.IsUpper(previous) && nextLower) {
				output.WriteByte('-')
			}
		}
		output.WriteRune(unicode.ToLower(character))
	}
	return output.String()
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
