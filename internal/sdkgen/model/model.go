// Package model validates and normalizes the language-neutral SDK projection.
package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"
	"unicode"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
)

const SchemaVersion = 1

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

// Model is the complete deterministic SDK input shared by language emitters.
type Model struct {
	SchemaVersion uint32              `json:"schemaVersion"`
	Checksum      string              `json:"checksum,omitempty"`
	Logical       []LogicalOperation  `json:"logicalOperations"`
	Physical      []PhysicalOperation `json:"physicalOperations"`
}

type LogicalOperation struct {
	ID         string              `json:"id"`
	RustName   string              `json:"rustName"`
	Group      string              `json:"group"`
	APIID      string              `json:"apiId"`
	GuideURL   string              `json:"guideUrl"`
	Parameters []Parameter         `json:"parameters"`
	Variants   []PhysicalReference `json:"variants"`
}

type PhysicalReference struct {
	OperationID    string         `json:"operationId"`
	Representation Representation `json:"representation"`
}

type PhysicalOperation struct {
	OperationID             string           `json:"operationId"`
	LogicalID               string           `json:"logicalOperationId"`
	RustConstant            string           `json:"rustConstant"`
	Method                  string           `json:"method"`
	Path                    string           `json:"path"`
	PrimaryRepresentation   Representation   `json:"primaryRepresentation"`
	ExpectedRepresentations []Representation `json:"expectedRepresentations"`
	Responses               []Response       `json:"responses"`
}

type Parameter struct {
	WireName string         `json:"wireName"`
	RustName string         `json:"rustName"`
	Required bool           `json:"required"`
	Shape    ParameterShape `json:"shape"`
	Explode  bool           `json:"explode"`
	MinItems *int64         `json:"minItems,omitempty"`
	MaxItems *int64         `json:"maxItems,omitempty"`
}

type Response struct {
	Selector           string          `json:"selector"`
	HTTPStatusEvidence string          `json:"httpStatusEvidence"`
	Media              []ResponseMedia `json:"media"`
}

type ResponseMedia struct {
	Name              string        `json:"name"`
	ContentTypeStatus string        `json:"contentTypeStatus"`
	Shape             ResponseShape `json:"shape"`
}

type ResponseShape struct {
	Kind                       string             `json:"kind"`
	Description                string             `json:"description,omitempty"`
	Required                   []string           `json:"required,omitempty"`
	Properties                 []ResponseProperty `json:"properties,omitempty"`
	Items                      *ResponseShape     `json:"items,omitempty"`
	AdditionalPropertiesPolicy string             `json:"additionalPropertiesPolicy"`
	XMLName                    string             `json:"xmlName,omitempty"`
	XMLNodeType                string             `json:"xmlNodeType,omitempty"`
	OpenStatus                 bool               `json:"openStatus,omitempty"`
	StatusValues               []string           `json:"statusValues,omitempty"`
}

type ResponseProperty struct {
	Name  string        `json:"name"`
	Shape ResponseShape `json:"shape"`
}

// Error reports a generator-model rule without including source bodies.
type Error struct {
	Rule      string
	Operation string
	Location  string
	Detail    string
}

func (e *Error) Error() string {
	parts := []string{"SDK model rejected input", "rule=" + e.Rule}
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

// Build validates complete coverage and returns a deterministic normalized model.
func Build(surface openapispec.SDKSurface) (Model, error) {
	if len(surface.Operations) == 0 {
		return Model{}, reject("operation-inventory", "", "#/paths", "no physical operations")
	}

	physicalIDs := make(map[string]bool, len(surface.Operations))
	constantNames := make(map[string]string, len(surface.Operations))
	logicalSources := make(map[string][]openapispec.SDKSurfaceOperation)
	physical := make([]PhysicalOperation, 0, len(surface.Operations))
	for _, source := range surface.Operations {
		operation, parameters, err := normalizePhysical(source)
		if err != nil {
			return Model{}, err
		}
		if physicalIDs[operation.OperationID] {
			return Model{}, reject("duplicate-operation-id", operation.OperationID, "operationId", operation.OperationID)
		}
		physicalIDs[operation.OperationID] = true
		if previous := constantNames[operation.RustConstant]; previous != "" {
			return Model{}, reject("rust-name-collision", operation.OperationID, "operationId", previous+" and "+operation.OperationID)
		}
		constantNames[operation.RustConstant] = operation.OperationID
		physical = append(physical, operation)
		entry := source
		entry.Parameters = nil
		for _, parameter := range parameters {
			entry.Parameters = append(entry.Parameters, denormalizeParameter(parameter))
		}
		logicalSources[source.LogicalOperationID] = append(logicalSources[source.LogicalOperationID], entry)
	}
	sort.Slice(physical, func(i, j int) bool { return physical[i].OperationID < physical[j].OperationID })

	logicalIDs := make([]string, 0, len(logicalSources))
	for id := range logicalSources {
		logicalIDs = append(logicalIDs, id)
	}
	sort.Strings(logicalIDs)
	logical := make([]LogicalOperation, 0, len(logicalIDs))
	typeNames := make(map[string]string, len(logicalIDs))
	groupNames := make(map[string]string)
	for _, id := range logicalIDs {
		sources := logicalSources[id]
		sort.Slice(sources, func(i, j int) bool { return sources[i].OperationID < sources[j].OperationID })
		first := sources[0]
		parameters, err := normalizeParameters(first)
		if err != nil {
			return Model{}, err
		}
		name := operationTypeName(first.Path)
		if !validRustIdentifier(name) {
			return Model{}, reject("invalid-rust-name", first.OperationID, "path", name)
		}
		if previous := typeNames[name]; previous != "" && previous != id {
			return Model{}, reject("rust-name-collision", first.OperationID, "path", previous+" and "+id+" normalize to "+name)
		}
		typeNames[name] = id
		group := rustFieldName(first.APIGroupCode)
		if !validRustIdentifier(group) {
			return Model{}, reject("invalid-rust-name", first.OperationID, "x-opendart/apiGroupCode", group)
		}
		if previous := groupNames[group]; previous != "" && previous != first.APIGroupCode {
			return Model{}, reject("rust-name-collision", first.OperationID, "x-opendart/apiGroupCode", previous+" and "+first.APIGroupCode)
		}
		groupNames[group] = first.APIGroupCode
		if err := validateLogicalMetadata(first); err != nil {
			return Model{}, err
		}
		variants := make([]PhysicalReference, 0, len(sources))
		seenRepresentations := make(map[Representation]bool)
		for _, source := range sources {
			if source.APIGroupCode != first.APIGroupCode || source.APIID != first.APIID || source.GuideURL != first.GuideURL {
				return Model{}, reject("incompatible-logical-metadata", source.OperationID, "x-opendart", id)
			}
			candidate, err := normalizeParameters(source)
			if err != nil {
				return Model{}, err
			}
			if !reflect.DeepEqual(candidate, parameters) {
				return Model{}, reject("incompatible-logical-variants", source.OperationID, "parameters", id)
			}
			candidateName := operationTypeName(source.Path)
			if candidateName != name {
				return Model{}, reject("incompatible-logical-names", source.OperationID, "path", name+" and "+candidateName)
			}
			representation, err := primaryRepresentation(source)
			if err != nil {
				return Model{}, err
			}
			if seenRepresentations[representation] {
				return Model{}, reject("duplicate-logical-representation", source.OperationID, "responses", string(representation))
			}
			seenRepresentations[representation] = true
			variants = append(variants, PhysicalReference{OperationID: source.OperationID, Representation: representation})
		}
		sort.Slice(variants, func(i, j int) bool { return variants[i].Representation < variants[j].Representation })
		logical = append(logical, LogicalOperation{
			ID: id, RustName: name, Group: group, APIID: first.APIID,
			GuideURL: first.GuideURL, Parameters: parameters, Variants: variants,
		})
	}

	result := Model{SchemaVersion: SchemaVersion, Logical: logical, Physical: physical}
	encoded, err := json.Marshal(result)
	if err != nil {
		return Model{}, fmt.Errorf("encode SDK projection: %w", err)
	}
	digest := sha256.Sum256(encoded)
	result.Checksum = hex.EncodeToString(digest[:])
	return result, nil
}

func normalizePhysical(source openapispec.SDKSurfaceOperation) (PhysicalOperation, []Parameter, error) {
	if source.OperationID == "" || source.LogicalOperationID == "" {
		return PhysicalOperation{}, nil, reject("operation-identity", source.OperationID, "operationId", "missing physical or logical identity")
	}
	if source.Method != "GET" {
		return PhysicalOperation{}, nil, reject("unsupported-method", source.OperationID, "method", source.Method)
	}
	if !strings.HasPrefix(source.RelativeTarget, "/api/") || strings.ContainsAny(source.RelativeTarget, "?#\\\r\n\t") || strings.Contains(source.RelativeTarget, "..") {
		return PhysicalOperation{}, nil, reject("untrusted-target", source.OperationID, "servers", source.RelativeTarget)
	}
	constant := upperSnake(source.OperationID)
	if !validRustIdentifier(constant) {
		return PhysicalOperation{}, nil, reject("invalid-rust-name", source.OperationID, "operationId", constant)
	}
	if len(source.Security) != 1 || len(source.Security[0].Schemes) != 1 {
		return PhysicalOperation{}, nil, reject("ambiguous-authentication", source.OperationID, "security", "expected one scheme")
	}
	scheme := source.Security[0].Schemes[0]
	if scheme.Type != "apiKey" || scheme.Location != "query" || scheme.Name != "crtfc_key" || len(scheme.Scopes) != 0 {
		return PhysicalOperation{}, nil, reject("unsupported-authentication", source.OperationID, "security", scheme.Identifier)
	}
	parameters, err := normalizeParameters(source)
	if err != nil {
		return PhysicalOperation{}, nil, err
	}
	primary, err := primaryRepresentation(source)
	if err != nil {
		return PhysicalOperation{}, nil, err
	}
	expectedMap := make(map[Representation]bool)
	responses := make([]Response, 0, len(source.Responses))
	for _, response := range source.Responses {
		media := make([]ResponseMedia, 0, len(response.MediaTypes))
		for _, item := range response.MediaTypes {
			representation, err := mediaRepresentation(item.Name)
			if err != nil {
				return PhysicalOperation{}, nil, reject("unsupported-response-media", source.OperationID, "responses/"+response.Selector, item.Name)
			}
			expectedMap[representation] = true
			shape, err := normalizeShape(item.Schema, item.Name)
			if err != nil {
				return PhysicalOperation{}, nil, reject("unsupported-response-schema", source.OperationID, "responses/"+response.Selector+"/"+item.Name, err.Error())
			}
			media = append(media, ResponseMedia{Name: item.Name, ContentTypeStatus: item.ContentTypeStatus, Shape: shape})
		}
		responses = append(responses, Response{Selector: response.Selector, HTTPStatusEvidence: response.HTTPStatusEvidence, Media: media})
	}
	if !expectedMap[primary] {
		return PhysicalOperation{}, nil, reject("missing-primary-response", source.OperationID, "responses", string(primary))
	}
	expected := []Representation{primary}
	for _, candidate := range []Representation{RepresentationJSON, RepresentationXML, RepresentationZIP} {
		if candidate != primary && expectedMap[candidate] {
			expected = append(expected, candidate)
		}
	}
	if !supportedRepresentationSet(expected) {
		return PhysicalOperation{}, nil, reject("unsupported-response-routing", source.OperationID, "responses", fmt.Sprint(expected))
	}
	return PhysicalOperation{
		OperationID: source.OperationID, LogicalID: source.LogicalOperationID,
		RustConstant: constant, Method: source.Method, Path: source.RelativeTarget,
		PrimaryRepresentation: primary, ExpectedRepresentations: expected, Responses: responses,
	}, parameters, nil
}

func normalizeParameters(source openapispec.SDKSurfaceOperation) ([]Parameter, error) {
	parameters := make([]Parameter, 0, len(source.Parameters))
	names := make(map[string]string, len(source.Parameters))
	for _, sourceParameter := range source.Parameters {
		if sourceParameter.Location != "query" || !sourceParameter.HasSchema {
			return nil, reject("unsupported-parameter-location", source.OperationID, "parameters/"+sourceParameter.Name, sourceParameter.Location)
		}
		parameter := Parameter{
			WireName: sourceParameter.Name, RustName: rustFieldName(sourceParameter.Name), Required: sourceParameter.Required,
			Explode: sourceParameter.Explode, MinItems: cloneInt(sourceParameter.MinItems), MaxItems: cloneInt(sourceParameter.MaxItems),
		}
		if sourceParameter.AllowEmptyValue || sourceParameter.AllowReserved {
			return nil, reject("unsupported-parameter-serialization", source.OperationID, "parameters/"+sourceParameter.Name, "allowEmptyValue and allowReserved must be false")
		}
		if !validWireName(parameter.WireName) {
			return nil, reject("unsafe-parameter-name", source.OperationID, "parameters/"+sourceParameter.Name, parameter.WireName)
		}
		if !validRustIdentifier(parameter.RustName) {
			return nil, reject("invalid-rust-name", source.OperationID, "parameters/"+sourceParameter.Name, parameter.RustName)
		}
		switch {
		case reflect.DeepEqual(sourceParameter.Types, []string{"string"}) && len(sourceParameter.ItemTypes) == 0:
			parameter.Shape = ScalarString
			if sourceParameter.Style != "" && sourceParameter.Style != "form" {
				return nil, reject("unsupported-parameter-serialization", source.OperationID, "parameters/"+sourceParameter.Name, sourceParameter.Style)
			}
		case reflect.DeepEqual(sourceParameter.Types, []string{"array"}) && reflect.DeepEqual(sourceParameter.ItemTypes, []string{"string"}):
			parameter.Shape = StringArray
			if sourceParameter.Style != "form" || sourceParameter.Explode {
				return nil, reject("unsupported-parameter-serialization", source.OperationID, "parameters/"+sourceParameter.Name, "arrays require form explode=false")
			}
			if sourceParameter.MinItems == nil || sourceParameter.MaxItems == nil || *sourceParameter.MinItems < 0 || *sourceParameter.MaxItems < *sourceParameter.MinItems {
				return nil, reject("invalid-cardinality", source.OperationID, "parameters/"+sourceParameter.Name, "explicit minItems and maxItems required")
			}
		default:
			return nil, reject("unsupported-parameter-schema", source.OperationID, "parameters/"+sourceParameter.Name, strings.Join(sourceParameter.Types, ","))
		}
		if previous := names[parameter.RustName]; previous != "" {
			return nil, reject("rust-name-collision", source.OperationID, "parameters", previous+" and "+parameter.WireName)
		}
		names[parameter.RustName] = parameter.WireName
		parameters = append(parameters, parameter)
	}
	return parameters, nil
}

func normalizeShape(source openapispec.SDKSurfaceSchema, mediaType string) (ResponseShape, error) {
	if mediaType == "application/zip" {
		return ResponseShape{Kind: "binary", Description: source.Description, AdditionalPropertiesPolicy: additionalPropertiesPolicy(source.AdditionalProperties)}, nil
	}
	if len(source.Types) == 0 && len(source.Properties) == 0 && source.Items == nil {
		if len(source.Required) != 0 || source.AdditionalProperties != nil {
			return ResponseShape{}, fmt.Errorf("opaque schema has object constraints")
		}
		return ResponseShape{
			Kind: "opaque", Description: source.Description, AdditionalPropertiesPolicy: additionalPropertiesPolicy(source.AdditionalProperties), XMLName: source.XMLName, XMLNodeType: source.XMLNodeType,
			OpenStatus: source.OpenStatus, StatusValues: append([]string(nil), source.StatusValues...),
		}, nil
	}
	if len(source.Types) != 1 {
		return ResponseShape{}, fmt.Errorf("schema must declare exactly one type, got %q", strings.Join(source.Types, ","))
	}
	shape := ResponseShape{
		Kind: source.Types[0], Description: source.Description, Required: append([]string(nil), source.Required...),
		AdditionalPropertiesPolicy: additionalPropertiesPolicy(source.AdditionalProperties),
		XMLName:                    source.XMLName,
		XMLNodeType:                source.XMLNodeType,
		OpenStatus:                 source.OpenStatus, StatusValues: append([]string(nil), source.StatusValues...),
	}
	if len(shape.Required) != 0 && shape.Kind != "object" {
		return ResponseShape{}, fmt.Errorf("required is only supported for object schemas")
	}
	if source.AdditionalProperties != nil && shape.Kind != "object" {
		return ResponseShape{}, fmt.Errorf("additionalProperties is only supported for object schemas")
	}
	if shape.OpenStatus && shape.Kind != "string" {
		return ResponseShape{}, fmt.Errorf("OpenDartStatus must be a string")
	}
	switch shape.Kind {
	case "object":
		if source.Items != nil {
			return ResponseShape{}, fmt.Errorf("object has array items")
		}
		properties := make(map[string]bool, len(source.Properties))
		for _, property := range source.Properties {
			properties[property.Name] = true
			child, err := normalizeShape(property.Schema, mediaType)
			if err != nil {
				return ResponseShape{}, fmt.Errorf("property %q: %w", property.Name, err)
			}
			shape.Properties = append(shape.Properties, ResponseProperty{Name: property.Name, Shape: child})
		}
		required := make(map[string]bool, len(shape.Required))
		for _, name := range shape.Required {
			if required[name] || !properties[name] {
				return ResponseShape{}, fmt.Errorf("required property %q is missing or duplicated", name)
			}
			required[name] = true
		}
	case "array":
		if len(source.Properties) != 0 {
			return ResponseShape{}, fmt.Errorf("array has object properties")
		}
		if source.Items == nil {
			return ResponseShape{}, fmt.Errorf("array has no item schema")
		}
		items, err := normalizeShape(*source.Items, mediaType)
		if err != nil {
			return ResponseShape{}, err
		}
		shape.Items = &items
	case "string", "opaque":
		if len(source.Properties) != 0 || source.Items != nil {
			return ResponseShape{}, fmt.Errorf("string has object or array children")
		}
	default:
		return ResponseShape{}, fmt.Errorf("type %q is unsupported", shape.Kind)
	}
	return shape, nil
}

func additionalPropertiesPolicy(value *bool) string {
	if value == nil {
		return "unspecified"
	}
	if *value {
		return "allowed"
	}
	return "forbidden"
}

func primaryRepresentation(source openapispec.SDKSurfaceOperation) (Representation, error) {
	primary := make(map[Representation]bool)
	for _, response := range source.Responses {
		if response.HTTPStatusEvidence != "not-documented" {
			return "", reject("unsupported-http-status-evidence", source.OperationID, "responses/"+response.Selector, response.HTTPStatusEvidence)
		}
		for _, media := range response.MediaTypes {
			representation, err := mediaRepresentation(media.Name)
			if err != nil {
				return "", reject("unsupported-response-media", source.OperationID, "responses/"+response.Selector, media.Name)
			}
			switch media.ContentTypeStatus {
			case "inferred-from-documented-output-format":
				primary[representation] = true
			case "empirically-observed-error-response":
				if representation != RepresentationXML {
					return "", reject("unsupported-response-routing", source.OperationID, "responses/"+response.Selector+"/"+media.Name, media.ContentTypeStatus)
				}
			default:
				return "", reject("unsupported-content-type-evidence", source.OperationID, "responses/"+response.Selector+"/"+media.Name, media.ContentTypeStatus)
			}
		}
	}
	if len(primary) != 1 {
		return "", reject("ambiguous-primary-representation", source.OperationID, "responses", fmt.Sprint(sortedRepresentations(primary)))
	}
	var result Representation
	for representation := range primary {
		result = representation
	}
	pathMatches := strings.HasSuffix(source.Path, ".json") && result == RepresentationJSON ||
		strings.HasSuffix(source.Path, ".xml") && (result == RepresentationXML || result == RepresentationZIP)
	if !pathMatches {
		return "", reject("unsupported-representation", source.OperationID, "path", source.Path+" => "+string(result))
	}
	return result, nil
}

func supportedRepresentationSet(representations []Representation) bool {
	return reflect.DeepEqual(representations, []Representation{RepresentationJSON}) ||
		reflect.DeepEqual(representations, []Representation{RepresentationXML}) ||
		reflect.DeepEqual(representations, []Representation{RepresentationZIP, RepresentationXML})
}

func sortedRepresentations(values map[Representation]bool) []Representation {
	var result []Representation
	for _, candidate := range []Representation{RepresentationJSON, RepresentationXML, RepresentationZIP} {
		if values[candidate] {
			result = append(result, candidate)
		}
	}
	return result
}

func mediaRepresentation(mediaType string) (Representation, error) {
	switch mediaType {
	case "application/json":
		return RepresentationJSON, nil
	case "application/xml":
		return RepresentationXML, nil
	case "application/zip":
		return RepresentationZIP, nil
	default:
		return "", fmt.Errorf("unsupported media type %q", mediaType)
	}
}

func operationTypeName(path string) string {
	base := strings.TrimPrefix(path, "/api/")
	if index := strings.LastIndexByte(base, '.'); index >= 0 {
		base = base[:index]
	}
	return upperCamel(base)
}

func upperCamel(value string) string {
	var result []rune
	capitalize := true
	for _, current := range value {
		if !unicode.IsLetter(current) && !unicode.IsDigit(current) {
			capitalize = true
			continue
		}
		if capitalize {
			current = unicode.ToUpper(current)
			capitalize = false
		}
		result = append(result, current)
	}
	if len(result) == 0 || unicode.IsDigit(result[0]) {
		result = append([]rune("Operation"), result...)
	}
	return string(result)
}

func upperSnake(value string) string {
	var result []rune
	for index, current := range value {
		if unicode.IsUpper(current) && index > 0 && len(result) > 0 && result[len(result)-1] != '_' {
			result = append(result, '_')
		}
		if unicode.IsLetter(current) || unicode.IsDigit(current) {
			result = append(result, unicode.ToUpper(current))
		} else if len(result) > 0 && result[len(result)-1] != '_' {
			result = append(result, '_')
		}
	}
	return strings.Trim(string(result), "_")
}

func rustFieldName(value string) string {
	var result []rune
	for _, current := range strings.ToLower(value) {
		if unicode.IsLetter(current) || unicode.IsDigit(current) || current == '_' {
			result = append(result, current)
		} else if len(result) > 0 && result[len(result)-1] != '_' {
			result = append(result, '_')
		}
	}
	name := strings.Trim(string(result), "_")
	if name == "" || unicode.IsDigit([]rune(name)[0]) {
		name = "field_" + name
	}
	if rustKeywords[name] {
		name += "_"
	}
	return name
}

func validRustIdentifier(value string) bool {
	if value == "" || !isASCIIIdentifierStart(value[0]) {
		return false
	}
	for index := 1; index < len(value); index++ {
		if !isASCIIIdentifierContinue(value[index]) {
			return false
		}
	}
	return !rustKeywords[value]
}

func isASCIIIdentifierStart(value byte) bool {
	return value == '_' || value >= 'A' && value <= 'Z' || value >= 'a' && value <= 'z'
}

func isASCIIIdentifierContinue(value byte) bool {
	return isASCIIIdentifierStart(value) || value >= '0' && value <= '9'
}

func validWireName(value string) bool {
	if value == "" {
		return false
	}
	for index := 0; index < len(value); index++ {
		current := value[index]
		if current != '_' && (current < 'A' || current > 'Z') && (current < 'a' || current > 'z') && (current < '0' || current > '9') {
			return false
		}
	}
	return true
}

func validateLogicalMetadata(source openapispec.SDKSurfaceOperation) error {
	metadata := []struct {
		location string
		value    string
	}{
		{location: "x-opendart/logicalOperationId", value: source.LogicalOperationID},
		{location: "x-opendart/apiGroupCode", value: source.APIGroupCode},
		{location: "x-opendart/apiId", value: source.APIID},
	}
	for _, item := range metadata {
		if item.value == "" || strings.ContainsAny(item.value, "`\r\n\t") {
			return reject("unsafe-generated-documentation", source.OperationID, item.location, item.value)
		}
	}
	guide, err := url.Parse(source.GuideURL)
	if err != nil || guide.Scheme != "https" || guide.Hostname() != "opendart.fss.or.kr" || guide.Port() != "" || guide.User != nil || guide.Fragment != "" || strings.ContainsAny(source.GuideURL, "<> \r\n\t") {
		return reject("unsafe-generated-documentation", source.OperationID, "x-opendart/source/guideUrl", source.GuideURL)
	}
	return nil
}

var rustKeywords = map[string]bool{
	"as": true, "break": true, "const": true, "continue": true, "crate": true,
	"else": true, "enum": true, "extern": true, "false": true, "fn": true,
	"for": true, "if": true, "impl": true, "in": true, "let": true, "loop": true,
	"match": true, "mod": true, "move": true, "mut": true, "pub": true, "ref": true,
	"return": true, "self": true, "Self": true, "static": true, "struct": true,
	"super": true, "trait": true, "true": true, "type": true, "unsafe": true,
	"use": true, "where": true, "while": true, "async": true, "await": true,
	"dyn": true, "abstract": true, "become": true, "box": true, "do": true,
	"final": true, "macro": true, "override": true, "priv": true, "typeof": true,
	"unsized": true, "virtual": true, "yield": true, "try": true,
}

func denormalizeParameter(parameter Parameter) openapispec.SDKSurfaceParameter {
	result := openapispec.SDKSurfaceParameter{
		Name: parameter.WireName, Location: "query", Required: parameter.Required,
		Style: "form", Explode: parameter.Explode, HasSchema: true,
		MinItems: cloneInt(parameter.MinItems), MaxItems: cloneInt(parameter.MaxItems),
	}
	if parameter.Shape == StringArray {
		result.Types = []string{"array"}
		result.ItemTypes = []string{"string"}
	} else {
		result.Types = []string{"string"}
	}
	return result
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
