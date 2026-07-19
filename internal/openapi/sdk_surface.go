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
	SecuritySchemes    []string
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
	MinItems  *int64
	MaxItems  *int64
	HasSchema bool
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

			parameters := make([]SDKSurfaceParameter, 0, len(pathItem.Parameters)+len(operation.Parameters))
			for _, parameter := range append(append([]*v3.Parameter(nil), pathItem.Parameters...), operation.Parameters...) {
				if parameter == nil {
					return SDKSurface{}, fmt.Errorf("%s has a nil parameter", identity)
				}
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
					evidence.MinItems = copyInt64(schema.MinItems)
					evidence.MaxItems = copyInt64(schema.MaxItems)
				}
				parameters = append(parameters, evidence)
			}

			securitySchemes := make([]string, 0)
			for _, requirement := range operation.Security {
				if requirement == nil || requirement.Requirements == nil {
					continue
				}
				for scheme := range requirement.Requirements.KeysFromOldest() {
					securitySchemes = append(securitySchemes, scheme)
				}
			}
			sort.Strings(securitySchemes)

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
				SecuritySchemes:    securitySchemes,
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
