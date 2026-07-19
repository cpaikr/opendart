package openapi

import (
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

// SDKSurface is compatibility evidence that the private OpenAPI boundary can
// expose generator inputs without exporting libopenapi model types.
type SDKSurface struct {
	Operations []SDKSurfaceOperation
}

// SDKSurfaceOperation contains only repository-owned values needed to prove
// that one physical operation is visible to a future normalized SDK model.
type SDKSurfaceOperation struct {
	Method             string
	Path               string
	OperationID        string
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
	Name      string
	Location  string
	Required  bool
	Style     string
	Explode   bool
	Types     []string
	ItemTypes []string
	MinItems  *int64
	MaxItems  *int64
	HasSchema bool
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
					Name:     parameter.Name,
					Location: parameter.In,
					Style:    parameter.Style,
					Explode:  parameter.IsExploded(),
				}
				if parameter.Required != nil {
					evidence.Required = *parameter.Required
				}
				if parameter.Schema != nil {
					schema := parameter.Schema.Schema()
					if schema == nil {
						return SDKSurface{}, fmt.Errorf("%s parameter %q schema cannot be resolved", identity, parameter.Name)
					}
					evidence.HasSchema = true
					evidence.Types = append([]string(nil), schema.Type...)
					if schema.Items != nil {
						if !schema.Items.IsA() || schema.Items.A == nil || schema.Items.A.Schema() == nil {
							return SDKSurface{}, fmt.Errorf("%s parameter %q has unsupported items schema", identity, parameter.Name)
						}
						evidence.ItemTypes = append([]string(nil), schema.Items.A.Schema().Type...)
					}
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
				for scheme := range requirement.Requirements.KeysFromOldest() {
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
				response, err := inspectSDKResponse(identity, "default", operation.Responses.Default)
				if err != nil {
					return SDKSurface{}, err
				}
				responses = append(responses, response)
			}
			if operation.Responses.Codes != nil {
				for selector, source := range operation.Responses.Codes.FromOldest() {
					response, err := inspectSDKResponse(identity, selector, source)
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
				OperationID:        operation.OperationId,
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

func inspectSDKResponse(identity, selector string, response *v3.Response) (SDKSurfaceResponse, error) {
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
		mediaTypes = append(mediaTypes, SDKSurfaceMediaType{Name: name, ContentTypeStatus: contentTypeStatus})
	}
	sort.Slice(mediaTypes, func(i, j int) bool { return mediaTypes[i].Name < mediaTypes[j].Name })
	if len(mediaTypes) == 0 {
		return SDKSurfaceResponse{}, fmt.Errorf("%s response %q has no media types", identity, selector)
	}
	return SDKSurfaceResponse{Selector: selector, HTTPStatusEvidence: httpStatus, MediaTypes: mediaTypes}, nil
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
