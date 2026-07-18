package openapi

import (
	"fmt"
	"mime"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
	"strings"

	validatorhelpers "github.com/pb33f/libopenapi-validator/helpers"
	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/high/v3"
	liborderedmap "github.com/pb33f/libopenapi/orderedmap"
	"github.com/pb33f/libopenapi/utils"
	"go.yaml.in/yaml/v4"
)

var (
	pathParameterPattern  = regexp.MustCompile(`\{([^{}]+)\}`)
	safeIdentifierPattern = regexp.MustCompile(`^[A-Za-z0-9._~-]+$`)
)

// LintDiagnostic is a stable repository-owned lint result. It deliberately
// excludes source bodies and URLs so callers can safely print diagnostics.
type LintDiagnostic struct {
	Rule      string `json:"rule"`
	Artifact  string `json:"artifact"`
	Operation string `json:"operation,omitempty"`
	Location  string `json:"location"`
	Message   string `json:"message"`
}

// Lint validates an OpenAPI document and applies the repository's strict
// structural policy. Invalid syntax, schemas, or references are returned as an
// error; policy violations are returned as deterministic diagnostics.
func Lint(root string) ([]LintDiagnostic, error) {
	document, err := Load(root)
	if err != nil {
		return nil, err
	}
	defer document.Close()
	return document.Lint()
}

// Lint applies the strict policy to an already loaded document.
func (d *Document) Lint() ([]LintDiagnostic, error) {
	if err := d.ValidateDocument(); err != nil {
		return nil, err
	}
	linter := documentLinter{
		artifact:         d.root,
		document:         &d.model.Model,
		declaredTags:     make(map[string]bool),
		operationIDs:     make(map[string]string),
		normalizedPaths:  make(map[string]string),
		activeSchemas:    make(map[*base.Schema]bool),
		visitedPathItems: make(map[*yaml.Node]bool),
		visitedDetached:  make(map[*v3.PathItem]bool),
		root:             d.root,
		files:            d.files,
	}
	linter.lint()
	sort.Slice(linter.diagnostics, func(i, j int) bool {
		left, right := linter.diagnostics[i], linter.diagnostics[j]
		if left.Location != right.Location {
			return left.Location < right.Location
		}
		if left.Rule != right.Rule {
			return left.Rule < right.Rule
		}
		return left.Message < right.Message
	})
	return linter.diagnostics, nil
}

type documentLinter struct {
	artifact         string
	document         *v3.Document
	diagnostics      []LintDiagnostic
	declaredTags     map[string]bool
	operationIDs     map[string]string
	normalizedPaths  map[string]string
	activeSchemas    map[*base.Schema]bool
	visitedPathItems map[*yaml.Node]bool
	visitedDetached  map[*v3.PathItem]bool
	referenceCounts  map[string]int
	root             string
	files            []string
}

func (l *documentLinter) lint() {
	l.lintTags()
	l.lintRootServers()
	l.lintServers("", "/servers", l.document.Servers)
	securitySchemes := l.securitySchemeNames()
	l.lintSecurity("/security", l.document.Security, securitySchemes)
	if l.document.Paths != nil && l.document.Paths.PathItems != nil {
		for pathName, pathItem := range l.document.Paths.PathItems.FromOldest() {
			l.lintPath(pathName, pathItem, securitySchemes)
		}
	}
	if l.document.Webhooks != nil {
		for name, pathItem := range l.document.Webhooks.FromOldest() {
			location := "/webhooks/" + escapeJSONPointer(name)
			l.lintPathItemOperations("webhook "+name, location, "", pathItem, securitySchemes)
		}
	}
	if l.document.Components != nil {
		if l.document.Components.Callbacks != nil {
			for name, callback := range l.document.Components.Callbacks.FromOldest() {
				location := "/components/callbacks/" + escapeJSONPointer(name)
				l.lintCallback("callback component "+name, location, callback, securitySchemes)
			}
		}
		if l.document.Components.PathItems != nil {
			for name, pathItem := range l.document.Components.PathItems.FromOldest() {
				location := "/components/pathItems/" + escapeJSONPointer(name)
				l.lintPathItemOperations("path item component "+name, location, "", pathItem, securitySchemes)
			}
		}
	}
	l.lintComponentNamesAndSchemas()
	l.lintUnusedComponents()
}

func (l *documentLinter) add(rule, operation, location, message string) {
	l.diagnostics = append(l.diagnostics, LintDiagnostic{
		Rule: rule, Artifact: l.artifact, Operation: operation,
		Location: location, Message: message,
	})
}

func (l *documentLinter) lintTags() {
	for index, tag := range l.document.Tags {
		location := fmt.Sprintf("/tags/%d", index)
		if tag == nil || strings.TrimSpace(tag.Name) == "" {
			l.add("tag-description", "", location, "tag name is required")
			continue
		}
		if l.declaredTags[tag.Name] {
			l.add("no-duplicated-tag-names", "", location, "tag name is duplicated")
		}
		l.declaredTags[tag.Name] = true
		if strings.TrimSpace(tag.Description) == "" {
			l.add("tag-description", "", location, "tag description is required")
		}
	}
	for index, tag := range l.document.Tags {
		if tag != nil && tag.Parent != "" && !l.declaredTags[tag.Parent] {
			l.add("no-invalid-tag-parents", "", fmt.Sprintf("/tags/%d/parent", index), "tag parent is not declared")
		}
	}
	parents := make(map[string]string)
	for _, tag := range l.document.Tags {
		if tag != nil {
			parents[tag.Name] = tag.Parent
		}
	}
	for index, tag := range l.document.Tags {
		if tag == nil {
			continue
		}
		visited := make(map[string]bool)
		for current := tag.Name; current != ""; current = parents[current] {
			if visited[current] {
				l.add("no-invalid-tag-parents", "", fmt.Sprintf("/tags/%d/parent", index), "tag parent hierarchy contains a cycle")
				break
			}
			visited[current] = true
		}
	}
}

func (l *documentLinter) lintRootServers() {
	source, err := os.ReadFile(l.root)
	if err != nil {
		return
	}
	var document yaml.Node
	if yaml.Unmarshal(source, &document) != nil {
		return
	}
	servers := yamlMappingValue(yamlDocumentValue(&document), "servers")
	if servers == nil {
		l.add("no-empty-servers", "", "/openapi", "servers are required")
		return
	}
	if servers.Kind != yaml.SequenceNode || len(servers.Content) == 0 {
		l.add("no-empty-servers", "", "/servers", "servers must be a non-empty array")
	}
}

func (l *documentLinter) lintPath(pathName string, item *v3.PathItem, securitySchemes map[string]bool) {
	location := "/paths/" + escapeJSONPointer(pathName)
	if strings.Contains(pathName, "?") {
		l.add("no-paths-with-query", "", location, "path must not contain a query string")
	}
	if pathName != "/" && strings.HasSuffix(pathName, "/") {
		l.add("no-path-trailing-slash", "", location, "path must not end with a slash")
	}
	normalized := pathParameterPattern.ReplaceAllString(pathName, "{}")
	if previous, exists := l.normalizedPaths[normalized]; exists && previous != pathName {
		l.add("no-ambiguous-paths", "", location, "path is indistinguishable from another templated path")
	} else {
		l.normalizedPaths[normalized] = pathName
	}
	if item == nil {
		return
	}
	l.lintPathItemOperations(pathName, location, pathName, item, securitySchemes)
}

func (l *documentLinter) lintPathItemOperations(identityPrefix, location, pathName string, item *v3.PathItem, securitySchemes map[string]bool) {
	if item == nil || l.markPathItemVisited(item) {
		return
	}
	l.lintServers("", location+"/servers", item.Servers)
	l.lintParameterList("", location+"/parameters", item.Parameters)
	for method, operation := range item.GetOperations().FromOldest() {
		if operation != nil {
			identity := strings.ToUpper(method) + " " + identityPrefix
			l.lintOperation(identity, location+"/"+strings.ToLower(method), pathName, item.Parameters, operation, securitySchemes)
		}
	}
}

func (l *documentLinter) markPathItemVisited(item *v3.PathItem) bool {
	if lowItem := item.GoLow(); lowItem != nil {
		if root := lowItem.GetRootNode(); root != nil {
			if l.visitedPathItems == nil {
				l.visitedPathItems = make(map[*yaml.Node]bool)
			}
			if l.visitedPathItems[root] {
				return true
			}
			l.visitedPathItems[root] = true
			return false
		}
	}
	if l.visitedDetached == nil {
		l.visitedDetached = make(map[*v3.PathItem]bool)
	}
	if l.visitedDetached[item] {
		return true
	}
	l.visitedDetached[item] = true
	return false
}

func (l *documentLinter) lintOperation(identity, location, pathName string, inherited []*v3.Parameter, operation *v3.Operation, securitySchemes map[string]bool) {
	if strings.TrimSpace(operation.OperationId) == "" {
		l.add("operation-operationId", identity, location, "operationId is required")
	} else {
		if !safeIdentifierPattern.MatchString(operation.OperationId) {
			l.add("operation-operationId-url-safe", identity, location+"/operationId", "operationId is not URL-safe")
		}
		if previous, exists := l.operationIDs[operation.OperationId]; exists {
			l.add("operation-operationId-unique", identity, location+"/operationId", "operationId is also used by "+previous)
		} else {
			l.operationIDs[operation.OperationId] = identity
		}
	}
	if strings.TrimSpace(operation.Summary) == "" {
		l.add("operation-summary", identity, location, "operation summary is required")
	}
	for _, tag := range operation.Tags {
		if !l.declaredTags[tag] {
			l.add("operation-tag-defined", identity, location+"/tags", "operation tag is not declared")
		}
	}
	l.lintServers(identity, location+"/servers", operation.Servers)
	l.lintSecurity(location+"/security", operation.Security, securitySchemes)
	l.lintParameterList(identity, location+"/parameters", operation.Parameters)
	l.lintQuerystringParameters(identity, location+"/parameters", inherited, operation.Parameters)
	if pathName != "" {
		l.lintPathParameters(identity, location, pathName, inherited, operation.Parameters)
	}
	l.lintResponses(identity, location+"/responses", operation.Responses)
	if operation.RequestBody != nil {
		l.lintContent(identity, location+"/requestBody/content", operation.RequestBody.Content)
	}
	if operation.Callbacks != nil {
		for name, callback := range operation.Callbacks.FromOldest() {
			callbackLocation := location + "/callbacks/" + escapeJSONPointer(name)
			l.lintCallback(identity+" callback "+name, callbackLocation, callback, securitySchemes)
		}
	}
}

func (l *documentLinter) lintCallback(identityPrefix, location string, callback *v3.Callback, securitySchemes map[string]bool) {
	if callback == nil || callback.Expression == nil {
		return
	}
	for expression, pathItem := range callback.Expression.FromOldest() {
		expressionLocation := location + "/" + escapeJSONPointer(expression)
		l.lintPathItemOperations(identityPrefix+" "+expression, expressionLocation, "", pathItem, securitySchemes)
	}
}

func (l *documentLinter) lintParameterList(operation, location string, parameters []*v3.Parameter) {
	seen := make(map[string]bool)
	for index, parameter := range parameters {
		if parameter == nil {
			continue
		}
		key := parameter.In + "\x00" + parameter.Name
		if seen[key] {
			l.add("parameter-unique", operation, fmt.Sprintf("%s/%d", location, index), "parameter name and location are duplicated")
		}
		seen[key] = true
		if parameter.In == "path" && (parameter.Required == nil || !*parameter.Required) {
			l.add("path-parameters-defined", operation, fmt.Sprintf("%s/%d/required", location, index), "path parameters must be required")
		}
		if parameter.Schema != nil {
			l.lintSchema(operation, fmt.Sprintf("%s/%d/schema", location, index), parameter.Schema)
			l.lintExample(operation, fmt.Sprintf("%s/%d/example", location, index), parameter.Example, parameter.Schema, "no-invalid-parameter-examples")
			if parameter.Examples != nil {
				for name, example := range parameter.Examples.FromOldest() {
					if example != nil {
						l.lintExampleObject(operation, fmt.Sprintf("%s/%d/examples/%s", location, index, escapeJSONPointer(name)), example)
						l.lintExample(operation, fmt.Sprintf("%s/%d/examples/%s", location, index, escapeJSONPointer(name)), example.Value, parameter.Schema, "no-invalid-parameter-examples")
					}
				}
			}
		}
		if parameter.Content != nil {
			l.lintContent(operation, fmt.Sprintf("%s/%d/content", location, index), parameter.Content)
		}
	}
}

func (l *documentLinter) lintQuerystringParameters(operation, location string, inherited, parameters []*v3.Parameter) {
	query, querystring := false, 0
	for _, parameter := range append(append([]*v3.Parameter(nil), inherited...), parameters...) {
		if parameter == nil {
			continue
		}
		switch parameter.In {
		case "query":
			query = true
		case "querystring":
			querystring++
		}
	}
	if querystring > 1 {
		l.add("querystring-parameters", operation, location, "only one querystring parameter is allowed")
	}
	if query && querystring > 0 {
		l.add("querystring-parameters", operation, location, "query and querystring parameters cannot be combined")
	}
}

func (l *documentLinter) lintPathParameters(operation, location, pathName string, inherited, parameters []*v3.Parameter) {
	declared := make(map[string]bool)
	for _, parameter := range append(append([]*v3.Parameter(nil), inherited...), parameters...) {
		if parameter != nil && parameter.In == "path" {
			declared[parameter.Name] = true
		}
	}
	placeholders := make(map[string]bool)
	for _, match := range pathParameterPattern.FindAllStringSubmatch(pathName, -1) {
		placeholders[match[1]] = true
		if !declared[match[1]] {
			l.add("path-parameters-defined", operation, location+"/parameters", "path template parameter is not declared: "+match[1])
		}
	}
	for name := range declared {
		if !placeholders[name] {
			l.add("path-parameters-defined", operation, location+"/parameters", "declared path parameter is absent from the path template: "+name)
		}
	}
}

func (l *documentLinter) lintResponses(operation, location string, responses *v3.Responses) {
	if responses == nil {
		l.add("operation-2xx-response", operation, location, "responses are required")
		return
	}
	// A default response is the repository's deliberate representation when the
	// official guide does not document HTTP status codes.
	hasSuccess := responses.Default != nil
	if responses.Codes != nil {
		for code, response := range responses.Codes.FromOldest() {
			if strings.HasPrefix(code, "2") {
				hasSuccess = true
			}
			responseLocation := location + "/" + escapeJSONPointer(code)
			if response == nil || strings.TrimSpace(response.Description) == "" {
				l.add("response-description", operation, responseLocation, "response description is required")
			}
			if response != nil {
				l.lintContent(operation, responseLocation+"/content", response.Content)
			}
		}
	}
	if responses.Default != nil {
		if strings.TrimSpace(responses.Default.Description) == "" {
			l.add("response-description", operation, location+"/default", "response description is required")
		}
		l.lintContent(operation, location+"/default/content", responses.Default.Content)
	}
	if !hasSuccess {
		l.add("operation-2xx-response", operation, location, "at least one 2xx response is required")
	}
}

func (l *documentLinter) lintContent(operation, location string, content *liborderedmap.Map[string, *v3.MediaType]) {
	if content == nil {
		return
	}
	for name, media := range content.FromOldest() {
		if _, _, err := mime.ParseMediaType(name); err != nil || !strings.Contains(name, "/") {
			l.add("media-type-name", operation, location+"/"+escapeJSONPointer(name), "media type name is invalid")
		}
		if media != nil && media.Schema != nil {
			mediaLocation := location + "/" + escapeJSONPointer(name)
			l.lintSchema(operation, mediaLocation+"/schema", media.Schema)
			l.lintExample(operation, mediaLocation+"/example", media.Example, media.Schema, "no-invalid-media-type-examples")
			if media.Examples != nil {
				for exampleName, example := range media.Examples.FromOldest() {
					if example != nil {
						l.lintExampleObject(operation, mediaLocation+"/examples/"+escapeJSONPointer(exampleName), example)
						l.lintExample(operation, mediaLocation+"/examples/"+escapeJSONPointer(exampleName), example.Value, media.Schema, "no-invalid-media-type-examples")
					}
				}
			}
			if media.Encoding != nil {
				if media.ItemEncoding != nil {
					l.add("invalid-encoding-combinations", operation, mediaLocation+"/encoding", "encoding and itemEncoding cannot be combined")
				}
				schema := media.Schema.Schema()
				for property := range media.Encoding.KeysFromOldest() {
					if schema == nil || schema.Properties == nil {
						l.add("no-invalid-media-type-encoding", operation, mediaLocation+"/encoding/"+escapeJSONPointer(property), "encoding requires an object schema property")
						continue
					}
					if _, exists := schema.Properties.Get(property); !exists {
						l.add("no-invalid-media-type-encoding", operation, mediaLocation+"/encoding/"+escapeJSONPointer(property), "encoding property is not declared by the media schema")
					}
				}
			}
		}
	}
}

func (l *documentLinter) lintExampleObject(operation, location string, example *base.Example) {
	if example == nil {
		return
	}
	value := example.Value != nil
	external := example.ExternalValue != ""
	data := example.DataValue != nil
	serialized := example.SerializedValue != ""
	if value && external || serialized && external || value && data || value && serialized {
		l.add("example-values", operation, location, "example value forms are mutually exclusive")
	}
}

func (l *documentLinter) lintExample(operation, location string, example *yaml.Node, proxy *base.SchemaProxy, rule string) {
	if example == nil || proxy == nil {
		return
	}
	if schema := proxy.Schema(); schema != nil && !schemaExampleValid(example, schema) {
		l.add(rule, operation, location, "example does not match the declared schema")
	}
}

func (l *documentLinter) lintServers(operation, location string, servers []*v3.Server) {
	if servers != nil && len(servers) == 0 {
		l.add("no-empty-servers", operation, location, "servers must not be an empty array")
	}
	for index, server := range servers {
		serverLocation := fmt.Sprintf("%s/%d", location, index)
		if server == nil || strings.TrimSpace(server.URL) == "" {
			l.add("no-empty-servers", "", serverLocation, "server URL is required")
			continue
		}
		if forbiddenServerHost(server.URL) {
			l.add("no-server-example.com", operation, serverLocation+"/url", "example.com and localhost are not allowed servers")
		}
		if server.URL != "/" && strings.HasSuffix(server.URL, "/") {
			l.add("no-server-trailing-slash", "", serverLocation+"/url", "server URL must not end with a slash")
		}
		used := make(map[string]bool)
		for _, match := range pathParameterPattern.FindAllStringSubmatch(server.URL, -1) {
			used[match[1]] = true
			if server.Variables == nil {
				l.add("no-undefined-server-variable", "", serverLocation+"/url", "server variable is not declared: "+match[1])
			} else if _, exists := server.Variables.Get(match[1]); !exists {
				l.add("no-undefined-server-variable", "", serverLocation+"/url", "server variable is not declared: "+match[1])
			}
		}
		if server.Variables != nil {
			for name, variable := range server.Variables.FromOldest() {
				if !used[name] {
					l.add("no-unused-server-variable", "", serverLocation+"/variables/"+escapeJSONPointer(name), "server variable is not used in the URL")
				}
				if variable != nil && variable.Enum != nil && len(variable.Enum) == 0 {
					l.add("server-variables-empty-enum", operation, serverLocation+"/variables/"+escapeJSONPointer(name)+"/enum", "server variable enum must not be empty")
				}
				if variable != nil && len(variable.Enum) > 0 && !containsString(variable.Enum, variable.Default) {
					l.add("server-variables-empty-enum", operation, serverLocation+"/variables/"+escapeJSONPointer(name)+"/default", "server variable default must be included in its enum")
				}
			}
		}
	}
}

func forbiddenServerHost(serverURL string) bool {
	parsed, err := url.Parse(serverURL)
	if err != nil {
		return false
	}
	hostname := strings.TrimRight(strings.ToLower(parsed.Hostname()), ".")
	return hostname == "localhost" || hostname == "example.com" || strings.HasSuffix(hostname, ".example.com")
}

func (l *documentLinter) securitySchemeNames() map[string]bool {
	result := make(map[string]bool)
	if l.document.Components != nil && l.document.Components.SecuritySchemes != nil {
		for name := range l.document.Components.SecuritySchemes.KeysFromOldest() {
			result[name] = true
		}
	}
	return result
}

func (l *documentLinter) lintSecurity(location string, requirements []*base.SecurityRequirement, schemes map[string]bool) {
	for index, requirement := range requirements {
		if requirement == nil || requirement.Requirements == nil {
			continue
		}
		for name := range requirement.Requirements.KeysFromOldest() {
			if !schemes[name] {
				l.add("security-defined", "", fmt.Sprintf("%s/%d/%s", location, index, escapeJSONPointer(name)), "security scheme is not declared")
			}
		}
	}
}

func (l *documentLinter) lintComponentNamesAndSchemas() {
	components := l.document.Components
	if components == nil {
		return
	}
	if components.Schemas != nil {
		lintComponentNames(l, "schemas", components.Schemas)
		for name, schema := range components.Schemas.FromOldest() {
			l.lintSchema("", "/components/schemas/"+escapeJSONPointer(name), schema)
		}
	}
	if components.Responses != nil {
		lintComponentNames(l, "responses", components.Responses)
	}
	if components.Parameters != nil {
		lintComponentNames(l, "parameters", components.Parameters)
	}
	if components.Examples != nil {
		lintComponentNames(l, "examples", components.Examples)
		for name, example := range components.Examples.FromOldest() {
			l.lintExampleObject("", "/components/examples/"+escapeJSONPointer(name), example)
		}
	}
	if components.RequestBodies != nil {
		lintComponentNames(l, "requestBodies", components.RequestBodies)
	}
	if components.Headers != nil {
		lintComponentNames(l, "headers", components.Headers)
	}
	if components.SecuritySchemes != nil {
		lintComponentNames(l, "securitySchemes", components.SecuritySchemes)
	}
	if components.Links != nil {
		lintComponentNames(l, "links", components.Links)
	}
	if components.Callbacks != nil {
		lintComponentNames(l, "callbacks", components.Callbacks)
	}
	if components.PathItems != nil {
		lintComponentNames(l, "pathItems", components.PathItems)
	}
	if components.MediaTypes != nil {
		lintComponentNames(l, "mediaTypes", components.MediaTypes)
	}
}

func lintComponentNames[V any](l *documentLinter, category string, entries *liborderedmap.Map[string, V]) {
	for name := range entries.KeysFromOldest() {
		if !safeIdentifierPattern.MatchString(name) {
			l.add("component-name-unique", "", "/components/"+category+"/"+escapeJSONPointer(name), "component name contains unsafe characters")
		}
	}
}

func (l *documentLinter) lintUnusedComponents() {
	source, err := os.ReadFile(l.root)
	if err != nil {
		return
	}
	var root yaml.Node
	if yaml.Unmarshal(source, &root) != nil {
		return
	}
	components := yamlMappingValue(yamlDocumentValue(&root), "components")
	if components == nil || components.Kind != yaml.MappingNode {
		return
	}

	counts := l.countReferences()
	securityUsage := l.usedSecuritySchemes()
	for categoryIndex := 0; categoryIndex+1 < len(components.Content); categoryIndex += 2 {
		category := components.Content[categoryIndex].Value
		entries := components.Content[categoryIndex+1]
		if entries.Kind != yaml.MappingNode {
			continue
		}
		for entryIndex := 0; entryIndex+1 < len(entries.Content); entryIndex += 2 {
			name := entries.Content[entryIndex].Value
			value := entries.Content[entryIndex+1]
			alias := referenceIdentity(l.root, "#/components/"+category+"/"+name)
			target := ""
			if referenceNode := yamlMappingValue(value, "$ref"); referenceNode != nil && referenceNode.Kind == yaml.ScalarNode {
				target = referenceIdentity(l.root, referenceNode.Value)
			}
			used := counts[alias] > 0
			if target != "" {
				// A named $ref component is live when its resolved target is used;
				// the alias itself need not appear in another reference.
				used = used || counts[target] > 0
			}
			if category == "securitySchemes" {
				used = securityUsage[name]
			}
			if !used {
				l.add("no-unused-components", "", "/components/"+category+"/"+escapeJSONPointer(name), "component is never used")
			}
		}
	}
}

func (l *documentLinter) countReferences() map[string]int {
	if l.referenceCounts != nil {
		return l.referenceCounts
	}
	l.referenceCounts = make(map[string]int)
	for _, file := range l.sourceFiles() {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		var document yaml.Node
		if yaml.Unmarshal(data, &document) != nil {
			continue
		}
		for _, reference := range references(&document) {
			l.referenceCounts[referenceIdentity(file, reference)]++
		}
	}
	return l.referenceCounts
}

func (l *documentLinter) usedSecuritySchemes() map[string]bool {
	used := make(map[string]bool)
	collect := func(requirements []*base.SecurityRequirement) {
		for _, requirement := range requirements {
			if requirement == nil || requirement.Requirements == nil {
				continue
			}
			for name := range requirement.Requirements.KeysFromOldest() {
				used[name] = true
			}
		}
	}
	collect(l.document.Security)
	visited := make(map[*v3.PathItem]bool)
	var walkPathItem func(*v3.PathItem)
	walkPathItem = func(item *v3.PathItem) {
		if item == nil || visited[item] {
			return
		}
		visited[item] = true
		for _, operation := range item.GetOperations().FromOldest() {
			if operation == nil {
				continue
			}
			collect(operation.Security)
			if operation.Callbacks != nil {
				for _, callback := range operation.Callbacks.FromOldest() {
					walkCallbackPathItems(callback, walkPathItem)
				}
			}
		}
	}
	if l.document.Paths != nil && l.document.Paths.PathItems != nil {
		for _, item := range l.document.Paths.PathItems.FromOldest() {
			walkPathItem(item)
		}
	}
	if l.document.Webhooks != nil {
		for _, item := range l.document.Webhooks.FromOldest() {
			walkPathItem(item)
		}
	}
	if l.document.Components != nil {
		if l.document.Components.PathItems != nil {
			for _, item := range l.document.Components.PathItems.FromOldest() {
				walkPathItem(item)
			}
		}
		if l.document.Components.Callbacks != nil {
			for _, callback := range l.document.Components.Callbacks.FromOldest() {
				walkCallbackPathItems(callback, walkPathItem)
			}
		}
	}
	return used
}

func walkCallbackPathItems(callback *v3.Callback, visit func(*v3.PathItem)) {
	if callback == nil || callback.Expression == nil {
		return
	}
	for _, item := range callback.Expression.FromOldest() {
		visit(item)
	}
}

func (l *documentLinter) sourceFiles() []string {
	files := make([]string, 0, len(l.files))
	for _, file := range l.files {
		if filepath.IsAbs(file) {
			files = append(files, file)
		} else {
			files = append(files, filepath.Join(filepath.Dir(l.root), filepath.FromSlash(file)))
		}
	}
	return files
}

func yamlDocumentValue(node *yaml.Node) *yaml.Node {
	if node != nil && node.Kind == yaml.DocumentNode && len(node.Content) > 0 {
		return node.Content[0]
	}
	return node
}

func yamlMappingValue(node *yaml.Node, key string) *yaml.Node {
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

func referenceIdentity(source, reference string) string {
	parts := strings.SplitN(reference, "#", 2)
	target := source
	if parts[0] != "" {
		target = filepath.Join(filepath.Dir(source), filepath.FromSlash(parts[0]))
	}
	absolute, err := filepath.Abs(target)
	if err == nil {
		target = absolute
	}
	fragment := ""
	if len(parts) == 2 {
		fragment = "#" + parts[1]
	}
	return filepath.Clean(target) + fragment
}

func (l *documentLinter) lintSchema(operation, location string, proxy *base.SchemaProxy) {
	if proxy == nil {
		return
	}
	schema := proxy.Schema()
	if schema == nil || l.activeSchemas[schema] {
		return
	}
	l.activeSchemas[schema] = true
	defer delete(l.activeSchemas, schema)
	for _, required := range schema.Required {
		if schema.Properties == nil {
			l.add("no-required-schema-properties-undefined", operation, location+"/required", "required property is not defined: "+required)
			continue
		}
		if _, exists := schema.Properties.Get(required); !exists {
			l.add("no-required-schema-properties-undefined", operation, location+"/required", "required property is not defined: "+required)
		}
	}
	l.lintRange(operation, location, "length", schema.MinLength, schema.MaxLength)
	l.lintRange(operation, location, "items", schema.MinItems, schema.MaxItems)
	l.lintRange(operation, location, "properties", schema.MinProperties, schema.MaxProperties)
	if schema.Minimum != nil && schema.Maximum != nil && *schema.Minimum > *schema.Maximum {
		l.add("no-schema-type-mismatch", operation, location, "minimum exceeds maximum")
	}
	if schema.Maximum != nil && schema.ExclusiveMaximum != nil && schema.ExclusiveMaximum.IsB() {
		l.add("no-mixed-number-range-constraints", operation, location+"/exclusiveMaximum", "maximum and numeric exclusiveMaximum cannot be combined")
	}
	if schema.Minimum != nil && schema.ExclusiveMinimum != nil && schema.ExclusiveMinimum.IsB() {
		l.add("no-mixed-number-range-constraints", operation, location+"/exclusiveMinimum", "minimum and numeric exclusiveMinimum cannot be combined")
	}
	for index, value := range schema.Enum {
		if !schemaValueMatchesType(value, schema.Type) {
			l.add("no-enum-type-mismatch", operation, fmt.Sprintf("%s/enum/%d", location, index), "enum value does not match the schema type")
		}
	}
	if schema.Default != nil && !schemaValueMatchesType(schema.Default, schema.Type) {
		l.add("no-schema-type-mismatch", operation, location+"/default", "default value does not match the schema type")
	}
	if schema.Example != nil && !schemaExampleValid(schema.Example, schema) {
		l.add("no-invalid-schema-examples", operation, location+"/example", "example does not match the schema")
	}
	for index, example := range schema.Examples {
		if !schemaExampleValid(example, schema) {
			l.add("no-invalid-schema-examples", operation, fmt.Sprintf("%s/examples/%d", location, index), "example does not match the schema")
		}
	}
	if schema.Discriminator != nil {
		if schema.Discriminator.PropertyName == "" {
			l.add("no-invalid-schema-discriminator", operation, location+"/discriminator", "discriminator propertyName is required")
		} else if schema.Properties != nil {
			if _, exists := schema.Properties.Get(schema.Discriminator.PropertyName); !exists {
				l.add("no-invalid-schema-discriminator", operation, location+"/discriminator/propertyName", "discriminator property is not defined")
			}
		}
		if schema.Discriminator.DefaultMapping == "" && !discriminatorPropertyRequired(schema, schema.Discriminator.PropertyName, make(map[*base.Schema]bool)) {
			l.add("discriminator-defaultMapping", operation, location+"/discriminator", "an optional discriminator property requires defaultMapping")
		}
		if mapping := schema.Discriminator.DefaultMapping; mapping != "" && !l.validDiscriminatorMapping(mapping) {
			l.add("discriminator-defaultMapping", operation, location+"/discriminator/defaultMapping", "defaultMapping does not identify a declared schema")
		}
	}
	for _, child := range schemaChildProxies(schema) {
		l.lintSchema(operation, location+child.location, child.proxy)
	}
}

func discriminatorPropertyRequired(schema *base.Schema, property string, visited map[*base.Schema]bool) bool {
	if schema == nil || visited[schema] {
		return true
	}
	visited[schema] = true
	defer delete(visited, schema)
	if containsString(schema.Required, property) {
		return true
	}
	for _, proxy := range schema.AllOf {
		if proxy != nil && discriminatorPropertyRequired(proxy.Schema(), property, visited) {
			return true
		}
	}
	for _, alternatives := range [][]*base.SchemaProxy{schema.OneOf, schema.AnyOf} {
		if len(alternatives) == 0 {
			continue
		}
		allRequired := true
		for _, proxy := range alternatives {
			if proxy == nil || !discriminatorPropertyRequired(proxy.Schema(), property, visited) {
				allRequired = false
				break
			}
		}
		if allRequired {
			return true
		}
	}
	return false
}

type schemaChild struct {
	location string
	proxy    *base.SchemaProxy
}

func schemaChildProxies(schema *base.Schema) []schemaChild {
	children := make([]schemaChild, 0)
	add := func(location string, proxy *base.SchemaProxy) {
		if proxy != nil {
			children = append(children, schemaChild{location: location, proxy: proxy})
		}
	}
	addMap := func(prefix string, entries *liborderedmap.Map[string, *base.SchemaProxy]) {
		if entries == nil {
			return
		}
		for name, proxy := range entries.FromOldest() {
			add(prefix+"/"+escapeJSONPointer(name), proxy)
		}
	}
	addMap("/properties", schema.Properties)
	addMap("/dependentSchemas", schema.DependentSchemas)
	addMap("/patternProperties", schema.PatternProperties)
	addMap("/$defs", schema.Defs)
	for index, proxy := range schema.AllOf {
		add(fmt.Sprintf("/allOf/%d", index), proxy)
	}
	for index, proxy := range schema.OneOf {
		add(fmt.Sprintf("/oneOf/%d", index), proxy)
	}
	for index, proxy := range schema.AnyOf {
		add(fmt.Sprintf("/anyOf/%d", index), proxy)
	}
	for index, proxy := range schema.PrefixItems {
		add(fmt.Sprintf("/prefixItems/%d", index), proxy)
	}
	add("/contains", schema.Contains)
	add("/if", schema.If)
	add("/then", schema.Then)
	add("/else", schema.Else)
	add("/propertyNames", schema.PropertyNames)
	add("/unevaluatedItems", schema.UnevaluatedItems)
	add("/contentSchema", schema.ContentSchema)
	add("/not", schema.Not)
	if schema.Items != nil && schema.Items.IsA() {
		add("/items", schema.Items.A)
	}
	if schema.AdditionalProperties != nil && schema.AdditionalProperties.IsA() {
		add("/additionalProperties", schema.AdditionalProperties.A)
	}
	if schema.UnevaluatedProperties != nil && schema.UnevaluatedProperties.IsA() {
		add("/unevaluatedProperties", schema.UnevaluatedProperties.A)
	}
	return children
}

func (l *documentLinter) validDiscriminatorMapping(mapping string) bool {
	if l.document.Components == nil || l.document.Components.Schemas == nil {
		return false
	}
	if _, exists := l.document.Components.Schemas.Get(mapping); exists {
		return true
	}
	const prefix = "#/components/schemas/"
	if strings.HasPrefix(mapping, prefix) {
		_, exists := l.document.Components.Schemas.Get(strings.TrimPrefix(mapping, prefix))
		return exists
	}
	parts := strings.SplitN(mapping, "#", 2)
	if mapping == "" || hasURIScheme(parts[0]) || strings.HasPrefix(parts[0], "//") || filepath.IsAbs(parts[0]) {
		return false
	}
	target := l.root
	if parts[0] != "" {
		target = filepath.Clean(filepath.Join(filepath.Dir(l.root), filepath.FromSlash(parts[0])))
	}
	if !isWithin(filepath.Dir(l.root), target) {
		return false
	}
	physicalRoot, err := filepath.EvalSymlinks(filepath.Dir(l.root))
	if err != nil {
		return false
	}
	physicalTarget, err := filepath.EvalSymlinks(target)
	if err != nil || !isWithin(physicalRoot, physicalTarget) {
		return false
	}
	source, err := os.ReadFile(physicalTarget)
	if err != nil {
		return false
	}
	if len(parts) == 1 || parts[1] == "" {
		return true
	}
	var document yaml.Node
	if yaml.Unmarshal(source, &document) != nil {
		return false
	}
	return resolveYAMLPointer(yamlDocumentValue(&document), parts[1]) != nil
}

func resolveYAMLPointer(node *yaml.Node, pointer string) *yaml.Node {
	if pointer == "" {
		return node
	}
	if !strings.HasPrefix(pointer, "/") {
		return nil
	}
	current := node
	for _, encoded := range strings.Split(strings.TrimPrefix(pointer, "/"), "/") {
		segment, err := url.PathUnescape(encoded)
		if err != nil {
			return nil
		}
		segment = strings.ReplaceAll(strings.ReplaceAll(segment, "~1", "/"), "~0", "~")
		switch current.Kind {
		case yaml.MappingNode:
			current = yamlMappingValue(current, segment)
		case yaml.SequenceNode:
			index, parseErr := strconv.Atoi(segment)
			if parseErr != nil || index < 0 || index >= len(current.Content) {
				return nil
			}
			current = current.Content[index]
		default:
			return nil
		}
		if current == nil {
			return nil
		}
	}
	return current
}

func (l *documentLinter) lintRange(operation, location, name string, minimum, maximum *int64) {
	if minimum != nil && maximum != nil && *minimum > *maximum {
		l.add("no-schema-type-mismatch", operation, location, "minimum "+name+" exceeds maximum "+name)
	}
}

func schemaValueMatchesType(node *yaml.Node, types []string) bool {
	if node == nil || len(types) == 0 {
		return true
	}
	for node != nil && (node.Kind == yaml.DocumentNode || node.Kind == yaml.AliasNode) {
		if node.Kind == yaml.AliasNode {
			node = node.Alias
		} else if len(node.Content) > 0 {
			node = node.Content[0]
		} else {
			return true
		}
	}
	if node == nil {
		return true
	}
	for _, schemaType := range types {
		switch schemaType {
		case "null":
			if node.Tag == "!!null" || (node.Tag == "" && node.Value == "null") {
				return true
			}
		case "string":
			if node.Tag == "!!str" || (node.Tag == "" && (node.Style != 0 || !looksLikeNonStringScalar(node.Value))) {
				return true
			}
		case "boolean":
			if node.Tag == "!!bool" || (node.Tag == "" && (node.Value == "true" || node.Value == "false")) {
				return true
			}
		case "integer":
			if node.Tag == "!!int" {
				return true
			}
			if node.Tag == "" && node.Style == 0 {
				if _, err := strconv.ParseInt(node.Value, 10, 64); err == nil {
					return true
				}
			}
		case "number":
			if node.Tag == "!!int" || node.Tag == "!!float" {
				if _, err := strconv.ParseFloat(node.Value, 64); err == nil {
					return true
				}
			}
			if node.Tag == "" && node.Style == 0 {
				if _, err := strconv.ParseFloat(node.Value, 64); err == nil {
					return true
				}
			}
		case "array":
			if node.Kind == yaml.SequenceNode {
				return true
			}
		case "object":
			if node.Kind == yaml.MappingNode {
				return true
			}
		}
	}
	return false
}

func looksLikeNonStringScalar(value string) bool {
	if value == "true" || value == "false" || value == "null" {
		return true
	}
	if _, err := strconv.ParseFloat(value, 64); err == nil {
		return true
	}
	return false
}

func schemaExampleValid(node *yaml.Node, schema *base.Schema) bool {
	if node == nil || schema == nil {
		return true
	}
	rendered, err := schema.RenderInlineWithContext(base.NewInlineRenderContextForValidation())
	if err != nil {
		return false
	}
	jsonSchema, err := utils.ConvertYAMLtoJSON(rendered)
	if err != nil {
		return false
	}
	compiled, err := validatorhelpers.NewCompiledSchemaWithVersion("urn:opendart:lint-schema", jsonSchema, nil, 3.2)
	if err != nil {
		return false
	}
	var value any
	if err := node.Decode(&value); err != nil {
		return false
	}
	return compiled.Validate(normalizeOpenAPIValue(value)) == nil
}

func containsString(values []string, wanted string) bool {
	for _, value := range values {
		if value == wanted {
			return true
		}
	}
	return false
}
