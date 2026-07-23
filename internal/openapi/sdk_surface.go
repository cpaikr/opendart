package openapi

import (
	"errors"
	"fmt"
	"net/url"
	"reflect"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// SDKSurface is compatibility evidence that the private OpenAPI boundary can
// expose generator inputs without exporting libopenapi model types.
type SDKSurface struct {
	Operations []SDKSurfaceOperation
}

// SDKSurfaceError reports a generator-boundary rule with stable operation and
// source-pointer context, without exposing third-party model values.
type SDKSurfaceError struct {
	Rule      string
	Operation string
	Location  string
	Detail    string
}

func (e *SDKSurfaceError) Error() string {
	return fmt.Sprintf("SDK surface rejected input rule=%s operation=%s location=%s detail=%s", e.Rule, e.Operation, e.Location, e.Detail)
}

func rejectSDKSurface(rule, operation, location, detail string) *SDKSurfaceError {
	return &SDKSurfaceError{Rule: rule, Operation: operation, Location: location, Detail: detail}
}

// SDKSurfaceOperation contains only repository-owned values needed to prove
// that one physical operation is visible to a future normalized SDK model.
type SDKSurfaceOperation struct {
	Method             string
	Path               string
	RelativeTarget     string
	OperationID        string
	Description        string
	LogicalOperationID string
	APIGroupCode       string
	APIID              string
	GuideURL           string
	SourceCheckedAt    string
	Parameters         []SDKSurfaceParameter
	Security           []SDKSurfaceSecurityRequirement
	Responses          []SDKSurfaceResponse
}

// SDKSurfaceParameter records the request serialization and schema facts that
// must cross the private OpenAPI boundary during SDK generation.
type SDKSurfaceParameter struct {
	Name              string
	Description       string
	Location          string
	Required          bool
	Style             string
	Explode           bool
	AllowEmptyValue   bool
	AllowReserved     bool
	Types             []string
	ItemTypes         []string
	MinItems          *int64
	MaxItems          *int64
	StringConstraints SDKSurfaceStringConstraints
	HasSchema         bool
}

// SDKSurfaceStringConstraints is the closed request-value validation surface
// supported by generators. For array parameters it applies to each item.
type SDKSurfaceStringConstraints struct {
	Format         string
	AllowedValues  []string
	MinLength      *int64
	MaxLength      *int64
	DecimalMinimum *int64
	DecimalMaximum *int64
}

// SDKSurfaceSchema is the conservative, repository-owned response-shape
// projection used by SDK generators. It deliberately excludes examples,
// defaults, formats, and narrative constraints.
type SDKSurfaceSchema struct {
	Reference            string
	Description          string
	Types                []string
	Required             []string
	Properties           []SDKSurfaceProperty
	Items                *SDKSurfaceSchema
	AdditionalProperties *bool
	XMLName              string
	XMLNodeType          string
	OpenStatus           bool
	StatusValues         []string
}

// SDKSurfaceProperty retains one named object property in source order after
// deterministic normalization.
type SDKSurfaceProperty struct {
	Name   string
	Schema SDKSurfaceSchema
}

// SDKSurfaceSecurityRequirement preserves one alternative security
// requirement and the definitions needed to place its credentials.
type SDKSurfaceSecurityRequirement struct {
	Schemes []SDKSurfaceSecurityScheme
}

// SDKSurfaceSecurityScheme contains the repository-owned credential placement
// facts for one referenced OpenAPI security scheme.
type SDKSurfaceSecurityScheme struct {
	Identifier string
	Type       string
	Location   string
	Name       string
	Scopes     []string
}

// SDKSurfaceResponse preserves the response selector and source HTTP-status
// evidence needed to distinguish normal and alternate representations.
type SDKSurfaceResponse struct {
	Selector           string
	HTTPStatusEvidence string
	MediaTypes         []SDKSurfaceMediaType
}

// SDKSurfaceMediaType preserves one response representation and the source
// evidence that determines how generated code routes it.
type SDKSurfaceMediaType struct {
	Name              string
	ContentTypeStatus string
	Schema            SDKSurfaceSchema
}

// InspectSDKSurface walks every physical operation and returns a deterministic
// repository-owned projection. It is intentionally a compatibility probe, not
// the final normalized generator model.
func (d *Document) InspectSDKSurface() (SDKSurface, error) {
	if d == nil || d.model == nil || d.model.Model.Paths == nil || d.model.Model.Paths.PathItems == nil {
		return SDKSurface{}, errors.New("OpenAPI document has no paths")
	}

	operations := make([]SDKSurfaceOperation, 0)
	for pathName, pathItem := range d.model.Model.Paths.PathItems.FromOldest() {
		if pathItem == nil {
			return SDKSurface{}, fmt.Errorf("path %q has no path item", pathName)
		}
		for method, operation := range pathItem.GetOperations().FromOldest() {
			if operation == nil {
				continue
			}
			identity := strings.ToUpper(method) + " " + pathName
			if strings.TrimSpace(operation.OperationId) == "" {
				return SDKSurface{}, fmt.Errorf("%s has no operationId", identity)
			}
			if operation.RequestBody != nil {
				return SDKSurface{}, rejectSDKSurface("unsupported-request-body", operation.OperationId, pathName+"/"+method+"/requestBody", "generated requests are bodyless")
			}
			relativeTarget, err := sdkRelativeTarget(d.model.Model.Servers, pathItem.Servers, operation.Servers, pathName)
			if err != nil {
				return SDKSurface{}, fmt.Errorf("%s has invalid server target: %w", identity, err)
			}
			if operation.Extensions == nil {
				return SDKSurface{}, fmt.Errorf("%s has no x-opendart metadata", identity)
			}
			node, ok := operation.Extensions.Get("x-opendart")
			if !ok || node == nil {
				return SDKSurface{}, fmt.Errorf("%s has no x-opendart metadata", identity)
			}
			source := sdkMappingValue(node, "source")
			logicalOperationID := sdkScalarValue(sdkMappingValue(node, "logicalOperationId"))
			apiGroupCode := sdkScalarValue(sdkMappingValue(node, "apiGroupCode"))
			apiID := sdkScalarValue(sdkMappingValue(node, "apiId"))
			guideURL := sdkScalarValue(sdkMappingValue(source, "guideUrl"))
			sourceCheckedAt := sdkScalarValue(sdkMappingValue(source, "checkedAt"))
			if logicalOperationID == "" || apiGroupCode == "" || apiID == "" || guideURL == "" || sourceCheckedAt == "" {
				return SDKSurface{}, fmt.Errorf("%s has incomplete x-opendart identity or source metadata", identity)
			}

			effectiveParameters, err := effectiveSDKParameters(identity, pathItem.Parameters, operation.Parameters)
			if err != nil {
				return SDKSurface{}, err
			}
			parameters := make([]SDKSurfaceParameter, 0, len(effectiveParameters))
			for _, parameter := range effectiveParameters {
				evidence := SDKSurfaceParameter{
					Name:            parameter.Name,
					Description:     parameter.Description,
					Location:        parameter.In,
					Style:           parameter.Style,
					Explode:         parameter.IsExploded(),
					AllowEmptyValue: parameter.AllowEmptyValue,
					AllowReserved:   parameter.AllowReserved,
				}
				if parameter.Required != nil {
					evidence.Required = *parameter.Required
				}
				if parameter.Schema != nil {
					schema := parameter.Schema.Schema()
					if schema == nil {
						return SDKSurface{}, fmt.Errorf("%s parameter %q schema cannot be resolved", identity, parameter.Name)
					}
					if keyword := unsupportedSDKParameterSchema(schema, false); keyword != "" {
						return SDKSurface{}, rejectSDKSurface("unsupported-request-schema", operation.OperationId, pathName+"/"+method+"/parameters/"+parameter.Name+"/schema", keyword)
					}
					evidence.HasSchema = true
					evidence.Types = append([]string(nil), schema.Type...)
					constraintSchema := schema
					if schema.Items != nil {
						if !schema.Items.IsA() || schema.Items.A == nil || schema.Items.A.Schema() == nil {
							return SDKSurface{}, fmt.Errorf("%s parameter %q has unsupported items schema", identity, parameter.Name)
						}
						itemSchema := schema.Items.A.Schema()
						if keyword := unsupportedSDKParameterSchema(itemSchema, true); keyword != "" {
							return SDKSurface{}, rejectSDKSurface("unsupported-request-schema", operation.OperationId, pathName+"/"+method+"/parameters/"+parameter.Name+"/schema/items", keyword)
						}
						evidence.ItemTypes = append([]string(nil), itemSchema.Type...)
						constraintSchema = itemSchema
					}
					constraints, err := inspectSDKStringConstraints(constraintSchema)
					if err != nil {
						return SDKSurface{}, rejectSDKSurface("invalid-request-constraint", operation.OperationId, pathName+"/"+method+"/parameters/"+parameter.Name+"/schema", err.Error())
					}
					evidence.StringConstraints = constraints
					evidence.MinItems = copyInt64(schema.MinItems)
					evidence.MaxItems = copyInt64(schema.MaxItems)
				}
				parameters = append(parameters, evidence)
			}

			securityRequirements := operation.Security
			if securityRequirements == nil {
				securityRequirements = d.model.Model.Security
			}
			security := make([]SDKSurfaceSecurityRequirement, 0, len(securityRequirements))
			for _, requirement := range securityRequirements {
				if requirement == nil || requirement.Requirements == nil {
					return SDKSurface{}, fmt.Errorf("%s has an invalid security requirement", identity)
				}
				schemes := make([]SDKSurfaceSecurityScheme, 0)
				for scheme, scopes := range requirement.Requirements.FromOldest() {
					if d.model.Model.Components == nil || d.model.Model.Components.SecuritySchemes == nil {
						return SDKSurface{}, fmt.Errorf("%s references security scheme %q without components", identity, scheme)
					}
					definition, ok := d.model.Model.Components.SecuritySchemes.Get(scheme)
					if !ok || definition == nil {
						return SDKSurface{}, fmt.Errorf("%s references unknown security scheme %q", identity, scheme)
					}
					schemes = append(schemes, SDKSurfaceSecurityScheme{
						Identifier: scheme,
						Type:       definition.Type,
						Location:   definition.In,
						Name:       definition.Name,
						Scopes:     append([]string(nil), scopes...),
					})
				}
				sort.Slice(schemes, func(i, j int) bool { return schemes[i].Identifier < schemes[j].Identifier })
				security = append(security, SDKSurfaceSecurityRequirement{Schemes: schemes})
			}

			if operation.Responses == nil {
				return SDKSurface{}, fmt.Errorf("%s has no responses", identity)
			}
			responses := make([]SDKSurfaceResponse, 0)
			if operation.Responses.Default != nil {
				response, err := inspectSDKResponse(identity, operation.OperationId, pathName+"/"+method+"/responses/default", "default", operation.Responses.Default)
				if err != nil {
					return SDKSurface{}, err
				}
				responses = append(responses, response)
			}
			if operation.Responses.Codes != nil {
				for selector, source := range operation.Responses.Codes.FromOldest() {
					response, err := inspectSDKResponse(identity, operation.OperationId, pathName+"/"+method+"/responses/"+selector, selector, source)
					if err != nil {
						return SDKSurface{}, err
					}
					responses = append(responses, response)
				}
			}
			if len(responses) == 0 {
				return SDKSurface{}, fmt.Errorf("%s has no response media types", identity)
			}

			operations = append(operations, SDKSurfaceOperation{
				Method:             strings.ToUpper(method),
				Path:               pathName,
				RelativeTarget:     relativeTarget,
				OperationID:        operation.OperationId,
				Description:        operation.Description,
				LogicalOperationID: logicalOperationID,
				APIGroupCode:       apiGroupCode,
				APIID:              apiID,
				GuideURL:           guideURL,
				SourceCheckedAt:    sourceCheckedAt,
				Parameters:         parameters,
				Security:           security,
				Responses:          responses,
			})
		}
	}
	sort.Slice(operations, func(i, j int) bool {
		if operations[i].Path == operations[j].Path {
			return operations[i].Method < operations[j].Method
		}
		return operations[i].Path < operations[j].Path
	})
	return SDKSurface{Operations: operations}, nil
}

func sdkRelativeTarget(document, pathItem, operation []*v3.Server, operationPath string) (string, error) {
	servers := operation
	if servers == nil {
		servers = pathItem
	}
	if servers == nil {
		servers = document
	}
	if len(servers) != 1 || servers[0] == nil {
		return "", errors.New("exactly one server is required")
	}
	server := servers[0]
	if (server.Variables != nil && !server.Variables.IsZero()) || strings.ContainsAny(server.URL, "{}") {
		return "", errors.New("server variables are unsupported")
	}
	parsed, err := url.Parse(server.URL)
	if err != nil {
		return "", errors.New("server URL is malformed")
	}
	if parsed.Scheme != "https" || parsed.Hostname() != "opendart.fss.or.kr" || parsed.Port() != "" || parsed.User != nil || parsed.RawQuery != "" || parsed.Fragment != "" {
		return "", errors.New("server URL is outside the trusted OpenDART origin")
	}
	base := strings.TrimSuffix(parsed.EscapedPath(), "/")
	if base == "" || !strings.HasPrefix(operationPath, "/") || strings.Contains(operationPath, "..") {
		return "", errors.New("server and operation paths cannot form a trusted relative target")
	}
	return base + operationPath, nil
}

func effectiveSDKParameters(identity string, inherited, operation []*v3.Parameter) ([]*v3.Parameter, error) {
	effective := make([]*v3.Parameter, 0, len(inherited)+len(operation))
	positions := make(map[string]int, len(inherited)+len(operation))
	keyFor := func(parameter *v3.Parameter) (string, error) {
		if parameter == nil {
			return "", fmt.Errorf("%s has a nil parameter", identity)
		}
		return parameter.In + "\x00" + parameter.Name, nil
	}
	for _, parameter := range inherited {
		key, err := keyFor(parameter)
		if err != nil {
			return nil, err
		}
		if _, exists := positions[key]; exists {
			return nil, fmt.Errorf("%s has duplicate inherited parameter %q", identity, parameter.Name)
		}
		positions[key] = len(effective)
		effective = append(effective, parameter)
	}
	operationKeys := make(map[string]bool, len(operation))
	for _, parameter := range operation {
		key, err := keyFor(parameter)
		if err != nil {
			return nil, err
		}
		if operationKeys[key] {
			return nil, fmt.Errorf("%s has duplicate operation parameter %q", identity, parameter.Name)
		}
		operationKeys[key] = true
		if position, exists := positions[key]; exists {
			effective[position] = parameter
			continue
		}
		positions[key] = len(effective)
		effective = append(effective, parameter)
	}
	return effective, nil
}

func inspectSDKResponse(identity, operationID, location, selector string, response *v3.Response) (SDKSurfaceResponse, error) {
	if response == nil || response.Content == nil {
		return SDKSurfaceResponse{}, fmt.Errorf("%s response %q has no content", identity, selector)
	}
	httpStatus := sdkExtensionScalar(response.Extensions, "x-opendart-http-status")
	if httpStatus == "" {
		return SDKSurfaceResponse{}, fmt.Errorf("%s response %q has no x-opendart HTTP status evidence", identity, selector)
	}
	mediaTypes := make([]SDKSurfaceMediaType, 0)
	for name, media := range response.Content.FromOldest() {
		if media == nil || media.Schema == nil || media.Schema.Schema() == nil {
			return SDKSurfaceResponse{}, fmt.Errorf("%s response %q media type %q has no resolvable schema", identity, selector, name)
		}
		contentTypeStatus := sdkExtensionScalar(media.Extensions, "x-opendart-content-type-status")
		if contentTypeStatus == "" {
			return SDKSurfaceResponse{}, fmt.Errorf("%s response %q media type %q has no x-opendart routing evidence", identity, selector, name)
		}
		schema, err := inspectSDKSchema(media.Schema, make(map[string]bool))
		if err != nil {
			return SDKSurfaceResponse{}, rejectSDKSurface("unsupported-response-schema", operationID, location+"/content/"+name+"/schema", err.Error())
		}
		mediaTypes = append(mediaTypes, SDKSurfaceMediaType{Name: name, ContentTypeStatus: contentTypeStatus, Schema: schema})
	}
	sort.Slice(mediaTypes, func(i, j int) bool { return mediaTypes[i].Name < mediaTypes[j].Name })
	if len(mediaTypes) == 0 {
		return SDKSurfaceResponse{}, fmt.Errorf("%s response %q has no media types", identity, selector)
	}
	return SDKSurfaceResponse{Selector: selector, HTTPStatusEvidence: httpStatus, MediaTypes: mediaTypes}, nil
}

func inspectSDKSchema(proxy interface {
	Schema() *base.Schema
	GetReference() string
}, visiting map[string]bool) (SDKSurfaceSchema, error) {
	if proxy == nil || proxy.Schema() == nil {
		return SDKSurfaceSchema{}, errors.New("schema cannot be resolved")
	}
	reference := proxy.GetReference()
	if reference != "" {
		if visiting[reference] {
			return SDKSurfaceSchema{}, fmt.Errorf("recursive schema reference %q is unsupported", reference)
		}
		visiting[reference] = true
		defer delete(visiting, reference)
	}
	source := proxy.Schema()
	if len(source.AllOf) != 0 || len(source.OneOf) != 0 || len(source.AnyOf) != 0 || source.Not != nil ||
		len(source.PrefixItems) != 0 || source.Contains != nil || source.If != nil || source.Then != nil || source.Else != nil ||
		source.PatternProperties != nil || source.UnevaluatedProperties != nil {
		return SDKSurfaceSchema{}, errors.New("unsupported composed or conditional schema construct")
	}
	projected := SDKSurfaceSchema{
		Reference:   reference,
		Description: source.Description,
		Types:       append([]string(nil), source.Type...),
		Required:    append([]string(nil), source.Required...),
		OpenStatus:  strings.HasSuffix(reference, "#/OpenDartStatus"),
	}
	if keyword := unsupportedSDKResponseSchema(source, projected.OpenStatus); keyword != "" {
		return SDKSurfaceSchema{}, fmt.Errorf("unsupported schema keyword %q", keyword)
	}
	if source.XML != nil {
		if source.XML.Namespace != "" || source.XML.Prefix != "" || source.XML.Attribute || source.XML.Wrapped || (source.XML.Extensions != nil && !source.XML.Extensions.IsZero()) ||
			(source.XML.NodeType != "" && source.XML.NodeType != "element") {
			return SDKSurfaceSchema{}, errors.New("unsupported XML schema metadata")
		}
		projected.XMLName = source.XML.Name
		projected.XMLNodeType = source.XML.NodeType
	}
	sort.Strings(projected.Types)
	sort.Strings(projected.Required)
	if source.AdditionalProperties != nil {
		switch {
		case source.AdditionalProperties.IsB():
			value := source.AdditionalProperties.B
			projected.AdditionalProperties = &value
		case source.AdditionalProperties.IsA():
			return SDKSurfaceSchema{}, errors.New("typed additionalProperties are unsupported")
		}
	}
	if projected.OpenStatus {
		for _, value := range source.Enum {
			if value == nil || value.Kind != yaml.ScalarNode {
				return SDKSurfaceSchema{}, errors.New("OpenDartStatus contains a non-scalar value")
			}
			projected.StatusValues = append(projected.StatusValues, value.Value)
		}
		sort.Strings(projected.StatusValues)
	}
	if source.Properties != nil {
		for name, property := range source.Properties.FromOldest() {
			child, err := inspectSDKSchema(property, visiting)
			if err != nil {
				return SDKSurfaceSchema{}, fmt.Errorf("property %q: %w", name, err)
			}
			projected.Properties = append(projected.Properties, SDKSurfaceProperty{Name: name, Schema: child})
		}
		sort.Slice(projected.Properties, func(i, j int) bool { return projected.Properties[i].Name < projected.Properties[j].Name })
	}
	if source.Items != nil {
		if !source.Items.IsA() || source.Items.A == nil {
			return SDKSurfaceSchema{}, errors.New("boolean array items are unsupported")
		}
		items, err := inspectSDKSchema(source.Items.A, visiting)
		if err != nil {
			return SDKSurfaceSchema{}, fmt.Errorf("array items: %w", err)
		}
		projected.Items = &items
	}
	return projected, nil
}

func unsupportedSDKParameterSchema(source *base.Schema, item bool) string {
	if !reflect.DeepEqual(source.Type, []string{"string"}) {
		if keyword := discardedSDKStringConstraint(source); keyword != "" {
			return keyword
		}
	}
	allowed := map[string]bool{
		"Type": true, "Title": true, "Description": true, "Format": true,
		"Enum": true, "MinLength": true, "MaxLength": true,
		"Examples": true, "Example": true, "ExternalDocs": true, "Deprecated": true,
		"Extensions": true, "ParentProxy": true,
	}
	if !item {
		allowed["Items"] = true
		allowed["MinItems"] = true
		allowed["MaxItems"] = true
	}
	return firstUnsupportedSchemaField(source, allowed, func(field string) bool {
		return field == "Extensions" && !supportedSDKParameterSchemaExtensions(source.Extensions)
	})
}

func discardedSDKStringConstraint(source *base.Schema) string {
	switch {
	case source.Format != "":
		return "format"
	case len(source.Enum) > 0:
		return "enum"
	case source.MinLength != nil:
		return "minLength"
	case source.MaxLength != nil:
		return "maxLength"
	case hasSDKExtension(source.Extensions, "x-opendart-decimal-range"):
		return "x-opendart-decimal-range"
	default:
		return ""
	}
}

func hasSDKExtension(extensions *orderedmap.Map[string, *yaml.Node], name string) bool {
	if extensions == nil {
		return false
	}
	_, exists := extensions.Get(name)
	return exists
}

func unsupportedSDKResponseSchema(source *base.Schema, openStatus bool) string {
	allowed := map[string]bool{
		"Type": true, "Properties": true, "Items": true, "Required": true,
		"AdditionalProperties": true, "Title": true, "Description": true, "Format": true,
		"Examples": true, "Example": true, "ExternalDocs": true, "Deprecated": true,
		"XML": true, "Extensions": true, "ParentProxy": true,
	}
	if openStatus {
		allowed["Enum"] = true
	}
	return firstUnsupportedSchemaField(source, allowed, func(field string) bool {
		if field != "Extensions" {
			return false
		}
		return !supportedSDKSchemaExtensions(source.Extensions)
	})
}

func supportedSDKSchemaExtensions(extensions *orderedmap.Map[string, *yaml.Node]) bool {
	return supportedSDKExtensions(extensions, false)
}

func supportedSDKParameterSchemaExtensions(extensions *orderedmap.Map[string, *yaml.Node]) bool {
	return supportedSDKExtensions(extensions, true)
}

func supportedSDKExtensions(extensions *orderedmap.Map[string, *yaml.Node], request bool) bool {
	if extensions == nil {
		return true
	}
	for name := range extensions.KeysFromOldest() {
		switch name {
		case "x-opendart", "x-opendart-code-descriptions", "x-opendart-documented-type", "x-opendart-korean-name", "x-opendart-normalization", "x-opendart-source-diagnostics":
		case "x-opendart-decimal-range":
			if request {
				continue
			}
			return false
		default:
			return false
		}
	}
	return true
}

func inspectSDKStringConstraints(source *base.Schema) (SDKSurfaceStringConstraints, error) {
	if source == nil {
		return SDKSurfaceStringConstraints{}, nil
	}
	if !reflect.DeepEqual(source.Type, []string{"string"}) {
		if keyword := discardedSDKStringConstraint(source); keyword != "" {
			return SDKSurfaceStringConstraints{}, fmt.Errorf("%s requires a string schema", keyword)
		}
		return SDKSurfaceStringConstraints{}, nil
	}
	constraints := SDKSurfaceStringConstraints{
		Format: source.Format, MinLength: copyInt64(source.MinLength), MaxLength: copyInt64(source.MaxLength),
	}
	switch constraints.Format {
	case "", "opendart-corp-code", "opendart-date", "opendart-year":
	default:
		return SDKSurfaceStringConstraints{}, fmt.Errorf("unsupported format %q", constraints.Format)
	}
	if constraints.MinLength != nil && *constraints.MinLength < 1 || constraints.MaxLength != nil && *constraints.MaxLength < 1 || constraints.MinLength != nil && constraints.MaxLength != nil && *constraints.MinLength > *constraints.MaxLength {
		return SDKSurfaceStringConstraints{}, errors.New("invalid string length range")
	}
	seen := make(map[string]bool, len(source.Enum))
	for _, value := range source.Enum {
		if value == nil || value.Kind != yaml.ScalarNode || value.ShortTag() != "!!str" || value.Value == "" || seen[value.Value] {
			return SDKSurfaceStringConstraints{}, errors.New("allowed values must be unique non-empty strings")
		}
		seen[value.Value] = true
		constraints.AllowedValues = append(constraints.AllowedValues, value.Value)
	}
	if source.Extensions != nil {
		if node, ok := source.Extensions.Get("x-opendart-decimal-range"); ok {
			if !validSDKDecimalRangeNode(node) {
				return SDKSurfaceStringConstraints{}, errors.New("invalid decimal range")
			}
			var value struct {
				Minimum *int64 `yaml:"minimum"`
				Maximum *int64 `yaml:"maximum"`
			}
			if node == nil || node.Decode(&value) != nil || value.Minimum == nil || *value.Minimum < 0 || value.Maximum != nil && *value.Maximum < *value.Minimum {
				return SDKSurfaceStringConstraints{}, errors.New("invalid decimal range")
			}
			constraints.DecimalMinimum = copyInt64(value.Minimum)
			constraints.DecimalMaximum = copyInt64(value.Maximum)
		}
	}
	return constraints, nil
}

func validSDKDecimalRangeNode(node *yaml.Node) bool {
	if node == nil || node.Kind != yaml.MappingNode || len(node.Content)%2 != 0 {
		return false
	}
	seen := make(map[string]bool, 2)
	for index := 0; index < len(node.Content); index += 2 {
		key := node.Content[index]
		if key == nil || key.Kind != yaml.ScalarNode || key.ShortTag() != "!!str" || seen[key.Value] {
			return false
		}
		switch key.Value {
		case "minimum", "maximum":
			seen[key.Value] = true
		default:
			return false
		}
	}
	return true
}

func firstUnsupportedSchemaField(source *base.Schema, allowed map[string]bool, unsupportedAllowed func(string) bool) string {
	value := reflect.ValueOf(source).Elem()
	typeOf := value.Type()
	for index := 0; index < value.NumField(); index++ {
		field := typeOf.Field(index)
		if field.PkgPath != "" || value.Field(index).IsZero() {
			continue
		}
		if field.Name == "Extensions" && source.Extensions != nil && source.Extensions.IsZero() {
			continue
		}
		if allowed[field.Name] && (unsupportedAllowed == nil || !unsupportedAllowed(field.Name)) {
			continue
		}
		keyword := strings.Split(field.Tag.Get("json"), ",")[0]
		if keyword == "" || keyword == "-" {
			keyword = field.Name
		}
		return keyword
	}
	return ""
}

func sdkExtensionScalar(extensions *orderedmap.Map[string, *yaml.Node], key string) string {
	if extensions == nil {
		return ""
	}
	node, ok := extensions.Get(key)
	if !ok {
		return ""
	}
	return sdkScalarValue(node)
}

func copyInt64(value *int64) *int64 {
	if value == nil {
		return nil
	}
	copied := *value
	return &copied
}

func sdkMappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for index := 0; index+1 < len(node.Content); index += 2 {
		if node.Content[index].Value == key {
			return node.Content[index+1]
		}
	}
	return nil
}

func sdkScalarValue(node *yaml.Node) string {
	if node == nil || node.Kind != yaml.ScalarNode {
		return ""
	}
	return strings.TrimSpace(node.Value)
}
