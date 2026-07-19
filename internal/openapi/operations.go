package openapi

import (
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"slices"
	"sort"
	"strings"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/high/v3"
	"go.yaml.in/yaml/v4"
)

// OperationCatalog is the repository-owned projection of callable OpenAPI
// operations. It deliberately keeps libopenapi model types behind this package
// so live conformance and future generators share one stable seam.
type OperationCatalog struct {
	Servers         []string
	SecuritySchemes []SecurityScheme
	Operations      []Operation
}

type SecurityScheme struct {
	Name          string
	Type          string
	Location      string
	ParameterName string
}

type Operation struct {
	Method                 string
	Path                   string
	OperationID            string
	LogicalOperationID     string
	Servers                []string
	SecurityRequirements   []string
	Parameters             []Parameter
	PrimaryRepresentation  string
	AlternateResponseMedia []string
}

func (o Operation) Identity() string {
	return o.Method + " " + o.Path + " " + o.PrimaryRepresentation
}

type Parameter struct {
	Name        string
	Location    string
	Required    bool
	Style       string
	Explode     bool
	SchemaTypes []string
}

// Operations returns a deterministic projection of every callable operation,
// including its primary success representation and alternate response media.
func (d *Document) Operations() (OperationCatalog, error) {
	if d == nil || d.model == nil {
		return OperationCatalog{}, errors.New("enumerate operations: document is not initialized")
	}
	model := &d.model.Model
	catalog := OperationCatalog{Servers: serverURLs(model.Servers)}
	if model.Components != nil && model.Components.SecuritySchemes != nil {
		for name, scheme := range model.Components.SecuritySchemes.FromOldest() {
			catalog.SecuritySchemes = append(catalog.SecuritySchemes, SecurityScheme{
				Name:          name,
				Type:          scheme.Type,
				Location:      scheme.In,
				ParameterName: scheme.Name,
			})
		}
	}
	if model.Paths == nil || model.Paths.PathItems == nil {
		return catalog, nil
	}
	for pathValue, pathItem := range model.Paths.PathItems.FromOldest() {
		for _, candidate := range pathOperations(pathItem) {
			if candidate.operation == nil {
				continue
			}
			projected, err := projectOperation(model, pathValue, pathItem, candidate.method, candidate.operation)
			if err != nil {
				return OperationCatalog{}, err
			}
			catalog.Operations = append(catalog.Operations, projected)
		}
	}
	sort.Slice(catalog.Operations, func(i, j int) bool {
		return catalog.Operations[i].Identity() < catalog.Operations[j].Identity()
	})
	return catalog, nil
}

// ValidateRequest checks query parameters against the selected OpenAPI
// operation without performing network access.
func (d *Document) ValidateRequest(method, pathValue string, query url.Values) error {
	request, err := http.NewRequest(method, "https://opendart.invalid"+pathValue+"?"+query.Encode(), nil)
	if err != nil {
		return fmt.Errorf("validate request: build request: %w", err)
	}
	valid, validationErrors := d.validator.GetParameterValidator().ValidateQueryParams(request)
	if valid {
		return nil
	}
	return validationError(fmt.Sprintf("%s %s request", method, pathValue), validationErrors)
}

type operationCandidate struct {
	method    string
	operation *v3.Operation
}

func pathOperations(item *v3.PathItem) []operationCandidate {
	return []operationCandidate{
		{method: http.MethodGet, operation: item.Get},
		{method: http.MethodPost, operation: item.Post},
		{method: http.MethodPut, operation: item.Put},
		{method: http.MethodDelete, operation: item.Delete},
		{method: http.MethodPatch, operation: item.Patch},
		{method: http.MethodHead, operation: item.Head},
		{method: http.MethodOptions, operation: item.Options},
		{method: http.MethodTrace, operation: item.Trace},
	}
}

func projectOperation(model *v3.Document, pathValue string, pathItem *v3.PathItem, method string, operation *v3.Operation) (Operation, error) {
	primary, alternates, err := responseRepresentations(operation)
	if err != nil {
		return Operation{}, fmt.Errorf("enumerate %s %s: %w", method, pathValue, err)
	}
	logicalOperationID, err := logicalOperationID(operation.Extensions)
	if err != nil {
		return Operation{}, fmt.Errorf("enumerate %s %s: %w", method, pathValue, err)
	}
	servers := operation.Servers
	if len(servers) == 0 {
		servers = pathItem.Servers
	}
	if len(servers) == 0 {
		servers = model.Servers
	}
	parameters := append([]*v3.Parameter(nil), pathItem.Parameters...)
	parameters = append(parameters, operation.Parameters...)
	projectedParameters := make([]Parameter, 0, len(parameters))
	seenParameters := make(map[string]bool, len(parameters))
	for _, parameter := range parameters {
		identity := parameter.In + "\x00" + parameter.Name
		if seenParameters[identity] {
			return Operation{}, fmt.Errorf("parameter %q in %q is duplicated", parameter.Name, parameter.In)
		}
		seenParameters[identity] = true
		var schemaTypes []string
		if parameter.Schema != nil && parameter.Schema.Schema() != nil {
			schemaTypes = append(schemaTypes, parameter.Schema.Schema().Type...)
		}
		projectedParameters = append(projectedParameters, Parameter{
			Name:        parameter.Name,
			Location:    parameter.In,
			Required:    parameter.Required != nil && *parameter.Required,
			Style:       parameter.Style,
			Explode:     parameter.IsExploded(),
			SchemaTypes: schemaTypes,
		})
	}
	sort.Slice(projectedParameters, func(i, j int) bool {
		if projectedParameters[i].Location == projectedParameters[j].Location {
			return projectedParameters[i].Name < projectedParameters[j].Name
		}
		return projectedParameters[i].Location < projectedParameters[j].Location
	})
	security := operation.Security
	if security == nil {
		security = model.Security
	}
	return Operation{
		Method:                 method,
		Path:                   pathValue,
		OperationID:            operation.OperationId,
		LogicalOperationID:     logicalOperationID,
		Servers:                serverURLs(servers),
		SecurityRequirements:   securityRequirementNames(security),
		Parameters:             projectedParameters,
		PrimaryRepresentation:  primary,
		AlternateResponseMedia: alternates,
	}, nil
}

func responseRepresentations(operation *v3.Operation) (string, []string, error) {
	if operation.Responses == nil {
		return "", nil, errors.New("responses are missing")
	}
	media := make(map[string]bool)
	collectResponseMedia(media, operation.Responses.Default)
	if operation.Responses.Codes != nil {
		for _, response := range operation.Responses.Codes.FromOldest() {
			collectResponseMedia(media, response)
		}
	}
	known := []string{"application/json", "application/xml", "application/zip"}
	for candidate := range media {
		if !slices.Contains(known, candidate) {
			return "", nil, fmt.Errorf("response media type %q is unsupported", candidate)
		}
	}
	primary := ""
	if media["application/zip"] {
		primary = "application/zip"
	} else if media["application/json"] != media["application/xml"] {
		if media["application/json"] {
			primary = "application/json"
		} else {
			primary = "application/xml"
		}
	}
	if primary == "" {
		return "", nil, errors.New("one primary JSON, XML, or ZIP representation is required")
	}
	delete(media, primary)
	alternates := make([]string, 0, len(media))
	for candidate := range media {
		alternates = append(alternates, candidate)
	}
	sort.Strings(alternates)
	return primary, alternates, nil
}

func collectResponseMedia(destination map[string]bool, response *v3.Response) {
	if response == nil || response.Content == nil {
		return
	}
	for mediaType := range response.Content.FromOldest() {
		destination[strings.ToLower(mediaType)] = true
	}
}

func logicalOperationID(extensions interface {
	Get(string) (*yaml.Node, bool)
}) (string, error) {
	if extensions == nil {
		return "", errors.New("x-opendart extension is missing")
	}
	node, exists := extensions.Get("x-opendart")
	if !exists || node == nil {
		return "", errors.New("x-opendart extension is missing")
	}
	var extension struct {
		LogicalOperationID string `yaml:"logicalOperationId"`
	}
	if err := node.Decode(&extension); err != nil {
		return "", errors.New("x-opendart extension is malformed")
	}
	if strings.TrimSpace(extension.LogicalOperationID) == "" {
		return "", errors.New("x-opendart logicalOperationId is missing")
	}
	return extension.LogicalOperationID, nil
}

func serverURLs(servers []*v3.Server) []string {
	result := make([]string, 0, len(servers))
	for _, server := range servers {
		result = append(result, server.URL)
	}
	return result
}

func securityRequirementNames(requirements []*base.SecurityRequirement) []string {
	set := make(map[string]bool)
	for _, requirement := range requirements {
		if requirement == nil || requirement.Requirements == nil {
			continue
		}
		for name := range requirement.Requirements.FromOldest() {
			set[name] = true
		}
	}
	result := make([]string, 0, len(set))
	for name := range set {
		result = append(result, name)
	}
	sort.Strings(result)
	return result
}
