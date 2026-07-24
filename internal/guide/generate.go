package guide

import (
	"errors"
	"fmt"
	"net/url"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"regexp"
	"sort"
	"strconv"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

const (
	apiServer      = guideOrigin + "/api"
	openAPIVersion = "3.2.0"
)

var (
	endpointGroupPattern   = regexp.MustCompile(`^DS\d{3}$`)
	endpointIDPattern      = regexp.MustCompile(`^\d+$`)
	operationIDPattern     = regexp.MustCompile(`[^A-Za-z0-9]+`)
	maximumItemsPattern    = regexp.MustCompile(`최대\s*(\d+)건`)
	multiCompanyOperations = map[string]bool{
		"DS003-2019017": true,
		"DS003-2022002": true,
	}
)

var zipErrorObservation = map[string]any{
	"observedAt":        "2026-07-17",
	"requestCondition":  "invalid-40-character-api-key",
	"httpStatus":        200,
	"contentTypeHeader": "application/xml;charset=UTF-8",
	"apiStatus":         "010",
}

type GenerateOptions struct {
	OutputDir string
	CheckedAt string
}

type GenerationResult struct {
	PhysicalPaths int
	SchemaFiles   int
}

// Generate renders guide-derived OpenAPI source artifacts into an existing,
// empty staging directory. Publication and staged validation belong to the
// caller so generation cannot replace accepted repository output directly.
func Generate(endpoints []Endpoint, options GenerateOptions) (GenerationResult, error) {
	if err := validateGenerationTarget(options.OutputDir); err != nil {
		return GenerationResult{}, err
	}
	if _, err := time.Parse("2006-01-02", options.CheckedAt); err != nil {
		return GenerationResult{}, fmt.Errorf("generate OpenAPI: checked date %q is not YYYY-MM-DD", options.CheckedAt)
	}

	files, result, err := buildGeneration(endpoints, options)
	if err != nil {
		return GenerationResult{}, err
	}
	fileNames := make([]string, 0, len(files))
	for name := range files {
		fileNames = append(fileNames, name)
	}
	sort.Strings(fileNames)
	for _, name := range fileNames {
		if err := writeYAML(filepath.Join(options.OutputDir, name), files[name]); err != nil {
			return GenerationResult{}, fmt.Errorf("generate OpenAPI artifact %q: %w", name, err)
		}
	}
	if err := os.WriteFile(filepath.Join(options.OutputDir, OutputMarker), []byte(OutputMarkerContent), 0o644); err != nil {
		return GenerationResult{}, fmt.Errorf("generate OpenAPI marker: %w", err)
	}
	return result, nil
}

func validateGenerationTarget(output string) error {
	if output == "" {
		return errors.New("generate OpenAPI: output directory is required")
	}
	info, err := os.Lstat(output)
	if err != nil {
		return fmt.Errorf("generate OpenAPI: inspect output directory: %w", err)
	}
	if info.Mode()&os.ModeSymlink != 0 || !info.IsDir() {
		return errors.New("generate OpenAPI: output must be a physical directory")
	}
	entries, err := os.ReadDir(output)
	if err != nil {
		return fmt.Errorf("generate OpenAPI: inspect output contents: %w", err)
	}
	if len(entries) != 0 {
		return errors.New("generate OpenAPI: output directory must be empty")
	}
	return nil
}

func buildGeneration(endpoints []Endpoint, options GenerateOptions) (map[string]any, GenerationResult, error) {
	files := make(map[string]any)
	paths := make(map[string]any)
	seenPaths := make(map[string]bool)
	seenOperations := make(map[string]bool)
	seenEndpoints := make(map[string]bool)
	schemaFiles := make(map[string]string)
	rootFile := filepath.Join(options.OutputDir, "openapi.yaml")
	componentsFile := filepath.Join(options.OutputDir, "components", "schemas.yaml")

	messageCodes := []MessageCode(nil)
	if len(endpoints) > 0 {
		messageCodes = endpoints[0].MessageCodes
	}
	for _, endpoint := range endpoints {
		if !reflect.DeepEqual(messageCodes, endpoint.MessageCodes) {
			return nil, GenerationResult{}, endpointError(endpoint, "message-code tables differ")
		}
		if err := validateEndpointIdentity(endpoint); err != nil {
			return nil, GenerationResult{}, err
		}
		if seenEndpoints[endpoint.LogicalOperationID] {
			return nil, GenerationResult{}, endpointError(endpoint, "duplicate logical endpoint identity")
		}
		seenEndpoints[endpoint.LogicalOperationID] = true

		groupDir := strings.ToLower(endpoint.APIGroupCode)
		schemaName := filepath.Join("schemas", groupDir, endpoint.APIID+".yaml")
		schemaFile := filepath.Join(options.OutputDir, schemaName)
		schemaFiles[endpoint.LogicalOperationID] = schemaFile
		files[schemaName] = normalizedResponseSchema(endpoint)

		for _, basicRow := range endpoint.BasicInfo {
			if basicRow.Method != "GET" || basicRow.Encoding != "UTF-8" {
				return nil, GenerationResult{}, endpointError(endpoint, "unexpected method %q or encoding %q", basicRow.Method, basicRow.Encoding)
			}
			requestURL, pathKey, err := documentedAPIURL(basicRow.RequestURL)
			if err != nil {
				return nil, GenerationResult{}, endpointError(endpoint, "%v", err)
			}
			if seenPaths[pathKey] {
				return nil, GenerationResult{}, endpointError(endpoint, "duplicate physical API path %q", pathKey)
			}
			seenPaths[pathKey] = true

			operationID := operationIDFor(requestURL)
			if seenOperations[operationID] {
				return nil, GenerationResult{}, endpointError(endpoint, "duplicate operationId %q", operationID)
			}
			seenOperations[operationID] = true

			pathName := filepath.Join("paths", groupDir, path.Base(requestURL.Path)+".yaml")
			pathFile := filepath.Join(options.OutputDir, pathName)
			fragment, err := pathFragment(endpoint, basicRow, pathFile, schemaFile, componentsFile, options.CheckedAt)
			if err != nil {
				return nil, GenerationResult{}, err
			}
			files[pathName] = fragment
			pathRef, err := relativeReference(rootFile, pathFile)
			if err != nil {
				return nil, GenerationResult{}, endpointError(endpoint, "resolve path artifact reference: %v", err)
			}
			paths[pathKey] = map[string]any{"$ref": pathRef}
		}
	}

	files[filepath.Join("components", "schemas.yaml")] = commonSchemas(messageCodes)
	root, err := rootDocument(endpoints, paths, seenPaths, schemaFiles, rootFile, componentsFile, options.CheckedAt)
	if err != nil {
		return nil, GenerationResult{}, err
	}
	files["openapi.yaml"] = root
	return files, GenerationResult{PhysicalPaths: len(seenPaths), SchemaFiles: len(schemaFiles)}, nil
}

func validateEndpointIdentity(endpoint Endpoint) error {
	wantLogical := endpoint.APIGroupCode + "-" + endpoint.APIID
	if !endpointGroupPattern.MatchString(endpoint.APIGroupCode) ||
		!endpointIDPattern.MatchString(endpoint.APIID) ||
		endpoint.LogicalOperationID != wantLogical {
		return endpointError(endpoint, "invalid endpoint identity (group %q, API ID %q, logical operation %q)", endpoint.APIGroupCode, endpoint.APIID, endpoint.LogicalOperationID)
	}
	source, err := trustedGuideURL(endpoint.SourceURL, "/guide/detail.do")
	if err != nil ||
		!reflect.DeepEqual(source.Query()["apiGrpCd"], []string{endpoint.APIGroupCode}) ||
		!reflect.DeepEqual(source.Query()["apiId"], []string{endpoint.APIID}) {
		return endpointError(endpoint, "guide source identity does not match the endpoint")
	}
	groupSource, err := trustedGuideURL(endpoint.GroupSourceURL, "/guide/main.do")
	if err != nil || len(groupSource.Query()) != 1 ||
		!reflect.DeepEqual(groupSource.Query()["apiGrpCd"], []string{endpoint.APIGroupCode}) {
		return endpointError(endpoint, "guide group source identity does not match the endpoint")
	}
	return nil
}

func endpointError(endpoint Endpoint, format string, args ...any) error {
	source := guideOrigin + "/guide/detail.do"
	if endpointGroupPattern.MatchString(endpoint.APIGroupCode) && endpointIDPattern.MatchString(endpoint.APIID) {
		source += "?apiGrpCd=" + url.QueryEscape(endpoint.APIGroupCode) + "&apiId=" + url.QueryEscape(endpoint.APIID)
	}
	return fmt.Errorf("generate OpenAPI operation %q from %q: %s", endpoint.LogicalOperationID, source, fmt.Sprintf(format, args...))
}

func documentedAPIURL(value string) (*url.URL, string, error) {
	u, err := url.Parse(value)
	if err != nil || u.Scheme != "https" || u.Host != "opendart.fss.or.kr" || u.User != nil || u.Fragment != "" || u.RawQuery != "" {
		return nil, "", errors.New("request URL is outside the documented API server")
	}
	if u.RawPath != "" || u.EscapedPath() != u.Path || path.Clean(u.Path) != u.Path ||
		!strings.HasPrefix(u.Path, "/api/") || u.Path == "/api/" {
		return nil, "", errors.New("request URL is outside the documented API server")
	}
	return u, strings.TrimPrefix(u.Path, "/api"), nil
}

func sourceDescription(row ResponseField) string {
	parts := make([]string, 0, 2)
	if row.Name != "" {
		parts = append(parts, row.Name)
	}
	if row.Description != "" && row.Description != row.Name {
		parts = append(parts, row.Description)
	}
	return strings.Join(parts, "\n\n")
}

func parameterSourceDiagnostics(endpoint Endpoint, argument RequestArgument) []any {
	diagnostics := make([]any, 0)
	if argument.Key == "bsns_year" && argument.DocumentedType == "STRING(1)" && strings.Contains(argument.Description, "4자리") {
		diagnostics = append(diagnostics, map[string]any{
			"code": "documented-length-conflict", "severity": "warning",
			"message":  "공식 가이드의 타입 길이와 값 설명의 자리수가 서로 다릅니다.",
			"evidence": map[string]any{"documentedType": argument.DocumentedType, "description": argument.Description},
		})
	}
	if endpoint.LogicalOperationID == "DS003-2019019" && argument.Key == "rcept_no" {
		diagnostics = append(diagnostics, map[string]any{
			"code": "inconsistent-length-across-endpoints", "severity": "warning",
			"message": "동일한 접수번호 요청키의 공식 가이드 타입 길이가 엔드포인트마다 다릅니다.",
			"evidence": []any{
				map[string]any{"logicalOperationId": endpoint.LogicalOperationID, "documentedType": argument.DocumentedType},
				map[string]any{"logicalOperationId": "DS001-2019003", "documentedType": "STRING(14)"},
			},
		})
	}
	if multiCompanyOperations[endpoint.LogicalOperationID] && argument.Key == "corp_code" {
		diagnostics = append(diagnostics, map[string]any{
			"code": "request-cardinality-conflict", "severity": "warning",
			"message":  "공식 가이드는 복수회사 조회를 설명하지만 요청 인자는 단일 STRING(8)로 표기합니다.",
			"evidence": map[string]any{"documentedType": argument.DocumentedType, "endpointDescription": endpoint.Description},
			"handling": "modeled-from-guide-test-example",
		})
	}
	return diagnostics
}

var responseFieldConflicts = map[string]struct {
	Name        string
	Description string
}{
	"DS001-2019001:total_count":            {Name: "총 건수", Description: "총 페이지 수"},
	"DS002-2019011:rgllbr_co":              {Name: "정규직 수", Description: "상근, 비상근"},
	"DS002-2019011:rgllbr_abacpt_labrr_co": {Name: "정규직 단시간 근로자 수", Description: "대표이사, 이사, 사외이사 등"},
}

func responseFieldSourceDiagnostics(endpoint Endpoint, rows []ResponseField) []any {
	diagnostics := make([]any, 0)
	for _, row := range rows {
		evidence, ok := responseFieldConflicts[endpoint.LogicalOperationID+":"+row.Key]
		if !ok || row.Name != evidence.Name || row.Description != evidence.Description {
			continue
		}
		diagnostics = append(diagnostics, map[string]any{
			"code": "field-name-description-conflict", "severity": "warning",
			"message":  "공식 가이드의 필드 명칭과 출력 설명이 서로 다른 의미를 가리킵니다.",
			"evidence": map[string]any{"name": evidence.Name, "description": evidence.Description},
		})
	}
	return diagnostics
}

type responseNode struct {
	key       string
	depth     float64
	container bool
	rows      []ResponseField
	children  []*responseNode
}

func normalizedResponseSchema(endpoint Endpoint) map[string]any {
	diagnostics := make([]any, 0)
	root := &responseNode{key: "$root", depth: -1, container: true}
	stack := []*responseNode{root}

	findOrAdd := func(parent *responseNode, row ResponseField, container bool) *responseNode {
		for _, candidate := range parent.children {
			if candidate.key == row.Key {
				if candidate.container != container {
					diagnostics = append(diagnostics, map[string]any{"code": "conflicting-source-kind", "key": row.Key, "sourceIndex": row.SourceIndex})
					candidate.container = candidate.container || container
				}
				candidate.rows = append(candidate.rows, row)
				return candidate
			}
		}
		depth := 0.0
		if row.Depth != nil {
			depth = *row.Depth
		}
		node := &responseNode{key: row.Key, depth: depth, container: container, rows: []ResponseField{row}}
		parent.children = append(parent.children, node)
		return node
	}

	for _, row := range endpoint.ResponseFields {
		depth := 0.0
		if row.Depth != nil {
			depth = *row.Depth
		}
		normalizedContainer := row.Key == "result" || row.Key == "list" || row.Key == "group"
		if row.Key == "result" && row.SourceKind != "container" {
			diagnostics = append(diagnostics, map[string]any{"code": "result-source-icon-is-not-container", "sourceIndex": row.SourceIndex, "sourceIconClass": row.SourceIconClass})
		}
		top := stack[len(stack)-1]
		if !normalizedContainer && top.key == "list" && depth <= top.depth {
			diagnostics = append(diagnostics, map[string]any{"code": "list-child-shares-container-depth", "key": row.Key, "sourceIndex": row.SourceIndex})
			findOrAdd(top, row, false)
			continue
		}
		for len(stack) > 1 && stack[len(stack)-1].depth >= depth {
			stack = stack[:len(stack)-1]
		}
		parent := stack[len(stack)-1]
		node := findOrAdd(parent, row, normalizedContainer || row.SourceKind == "container")
		if node.container {
			stack = append(stack, node)
		}
	}

	effectiveRoot := root
	var resultNode *responseNode
	for _, child := range root.children {
		if child.key == "result" {
			resultNode = child
			effectiveRoot = child
			break
		}
	}
	schema := objectSchema(endpoint, effectiveRoot)
	if resultNode != nil {
		schema["xml"] = map[string]any{"nodeType": "element", "name": resultNode.key}
	}
	schema["description"] = "공식 OpenDART 가이드의 응답 결과 표를 정규화한 보수적 스키마입니다. 가이드가 필드 타입을 제공하지 않으므로 타입을 추정하지 않았습니다."
	var sourceRoot any
	if resultNode != nil {
		sourceRoot = "result"
	}
	schema["x-opendart"] = map[string]any{
		"schemaStatus": "source-derived-unverified", "sourceRootKey": sourceRoot,
		"diagnostics": diagnostics, "responseFields": endpoint.ResponseFields,
	}
	return schema
}

func objectSchema(endpoint Endpoint, node *responseNode) map[string]any {
	properties := make(map[string]any)
	for _, child := range node.children {
		if child.container {
			nested := objectSchema(endpoint, child)
			if child.key == "list" || child.key == "group" {
				properties[child.key] = map[string]any{
					"type": "array", "items": nested,
					"description":              "공식 가이드의 계층 표시를 바탕으로 배열 컨테이너로 정규화했습니다.",
					"x-opendart-normalization": "source-derived-unverified",
				}
			} else {
				properties[child.key] = nested
			}
			continue
		}
		if child.key == "status" {
			properties[child.key] = map[string]any{"$ref": "../../components/schemas.yaml#/OpenDartStatus"}
			continue
		}
		descriptions := make([]string, 0)
		seenDescriptions := make(map[string]bool)
		for _, row := range child.rows {
			description := sourceDescription(row)
			if description != "" && !seenDescriptions[description] {
				descriptions = append(descriptions, description)
				seenDescriptions[description] = true
			}
		}
		property := map[string]any{"x-opendart-documented-type": "not-specified"}
		if len(descriptions) > 0 {
			property["description"] = strings.Join(descriptions, "\n\n")
		}
		if len(child.rows) > 0 && child.rows[0].Name != "" {
			property["x-opendart-korean-name"] = child.rows[0].Name
		}
		if diagnostics := responseFieldSourceDiagnostics(endpoint, child.rows); len(diagnostics) > 0 {
			property["x-opendart-source-diagnostics"] = diagnostics
		}
		properties[child.key] = property
	}
	return map[string]any{"type": "object", "properties": properties, "additionalProperties": true}
}

type parameterSerializationResult struct {
	maximumItems    int
	values          []string
	serializedValue string
	extension       map[string]any
}

func parameterSerialization(endpoint Endpoint, argument RequestArgument) (*parameterSerializationResult, error) {
	if !multiCompanyOperations[endpoint.LogicalOperationID] || argument.Key != "corp_code" {
		return nil, nil
	}
	value := ""
	for _, candidate := range endpoint.GuideTestRequestArguments {
		if candidate.Key == "corp_code" {
			value = candidate.Value
			break
		}
	}
	parts := strings.Split(value, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		if part != "" {
			values = append(values, part)
		}
	}
	if len(values) < 2 || value != strings.Join(values, ",") {
		return nil, endpointError(endpoint, "multi-company guide example is missing or malformed")
	}
	for _, candidate := range values {
		if len(candidate) != 8 || !endpointIDPattern.MatchString(candidate) {
			return nil, endpointError(endpoint, "multi-company guide example is missing or malformed")
		}
	}
	var maximum MessageCode
	for _, row := range endpoint.MessageCodes {
		if row.Code == "021" {
			maximum = row
			break
		}
	}
	match := maximumItemsPattern.FindStringSubmatch(maximum.Description)
	if len(match) != 2 {
		return nil, endpointError(endpoint, "multi-company maximum is missing from message 021")
	}
	maximumItems, err := strconv.Atoi(match[1])
	if err != nil || maximumItems < 1 {
		return nil, endpointError(endpoint, "multi-company maximum is missing from message 021")
	}
	extension := map[string]any{
		"status": "guide-example-supported", "wireFormat": "comma-separated", "delimiter": ",",
		"guideEvidence": map[string]any{
			"source": "official-guide-test-form", "serializedValue": value, "values": values,
			"maximumItems": map[string]any{
				"value": maximumItems, "source": "official-guide-message-code", "messageCode": maximum.Code, "description": maximum.Description,
			},
		},
		"authenticatedVerification": map[string]any{"status": "pending"},
	}
	return &parameterSerializationResult{
		maximumItems: maximumItems, values: values, serializedValue: value, extension: extension,
	}, nil
}

func parameterObjects(endpoint Endpoint) ([]any, error) {
	parameters := make([]any, 0, len(endpoint.RequestArguments))
	for _, argument := range endpoint.RequestArguments {
		if argument.Required != "Y" && argument.Required != "N" {
			return nil, endpointError(endpoint, "request argument %q has unknown requiredness %q", argument.Key, argument.Required)
		}
		if argument.Key == "crtfc_key" {
			continue
		}
		serialization, err := parameterSerialization(endpoint, argument)
		if err != nil {
			return nil, err
		}
		parameter := map[string]any{
			"name": argument.Key, "in": "query", "required": argument.Required == "Y",
			"description": argument.Description, "schema": stringSchema(argument),
			"x-opendart-korean-name":         argument.Name,
			"x-opendart-documented-type":     argument.DocumentedType,
			"x-opendart-documented-required": argument.Required,
		}
		if parameter["description"] == "" {
			parameter["description"] = argument.Name
		}
		if diagnostics := parameterSourceDiagnostics(endpoint, argument); len(diagnostics) > 0 {
			parameter["x-opendart-source-diagnostics"] = diagnostics
		}
		if serialization != nil {
			parameter["style"] = "form"
			parameter["explode"] = false
			parameter["schema"] = map[string]any{"type": "array", "minItems": 1, "maxItems": serialization.maximumItems, "items": stringSchema(argument)}
			parameter["examples"] = map[string]any{"officialGuide": map[string]any{
				"summary": "공식 개발가이드 테스트 예시", "dataValue": serialization.values, "serializedValue": "corp_code=" + serialization.serializedValue,
			}}
			parameter["x-opendart-serialization"] = serialization.extension
		}
		parameters = append(parameters, parameter)
	}
	return parameters, nil
}

func outputMediaType(format string) (string, error) {
	switch format {
	case "JSON":
		return "application/json", nil
	case "XML":
		return "application/xml", nil
	case "Zip FILE (binary)":
		return "application/zip", nil
	default:
		return "", fmt.Errorf("unknown documented output format %q", format)
	}
}

func operationIDFor(u *url.URL) string {
	return "get_" + operationIDPattern.ReplaceAllString(path.Base(u.Path), "_")
}

func relativeReference(fromFile, toFile string) (string, error) {
	value, err := filepath.Rel(filepath.Dir(fromFile), toFile)
	if err != nil {
		return "", err
	}
	value = filepath.ToSlash(value)
	if !strings.HasPrefix(value, ".") {
		value = "./" + value
	}
	return value, nil
}

func pathFragment(endpoint Endpoint, basic BasicInfo, pathFile, schemaFile, componentsFile, checkedAt string) (map[string]any, error) {
	mediaType, err := outputMediaType(basic.OutputFormat)
	if err != nil {
		return nil, endpointError(endpoint, "%v", err)
	}
	parameters, err := parameterObjects(endpoint)
	if err != nil {
		return nil, err
	}
	schemaRef, err := relativeReference(pathFile, schemaFile)
	if err != nil {
		return nil, endpointError(endpoint, "resolve response schema reference: %v", err)
	}
	componentsRef, err := relativeReference(pathFile, componentsFile)
	if err != nil {
		return nil, endpointError(endpoint, "resolve common schema reference: %v", err)
	}
	binary := basic.OutputFormat == "Zip FILE (binary)"
	schemaComponent := endpoint.APIGroupCode + "_" + endpoint.APIID + "_Response"
	var content map[string]any
	if binary {
		content = map[string]any{
			mediaType: map[string]any{"schema": map[string]any{}, "x-opendart-content-type-status": "inferred-from-documented-output-format"},
			"application/xml": map[string]any{
				"schema":                         map[string]any{"$ref": componentsRef + "#/OpenDartXmlError"},
				"x-opendart-content-type-status": "empirically-observed-error-response",
				"x-opendart-observation":         zipErrorObservation,
			},
		}
	} else {
		content = map[string]any{mediaType: map[string]any{
			"schema":                         map[string]any{"$ref": schemaRef},
			"x-opendart-content-type-status": "inferred-from-documented-output-format",
		}}
	}
	response := map[string]any{
		"description": "공식 가이드는 API 수준 status/message를 설명하지만 HTTP 상태 코드는 별도로 규정하지 않습니다.",
		"content":     content, "x-opendart-http-status": "not-documented",
	}
	if binary {
		response["description"] = "공식 가이드는 성공 시 ZIP binary 출력을 설명합니다. 무효한 40자리 키를 사용한 실측에서는 동일 응답이 HTTP 200의 XML API 오류를 반환했습니다."
		response["x-opendart-documented-response-schema"] = map[string]any{
			"component": schemaComponent,
			"note":      "응답 결과 표 또는 ZIP 내부 XML 필드 설명을 보존합니다. 성공 ZIP 자체의 구조를 뜻하지 않습니다.",
		}
	}
	requestURL, _, _ := documentedAPIURL(basic.RequestURL)
	operation := map[string]any{
		"operationId": operationIDFor(requestURL), "summary": endpoint.Name + " (" + basic.OutputFormat + ")",
		"description": endpoint.Description, "tags": []string{endpoint.APIGroupCode},
		"externalDocs": map[string]any{"description": "OpenDART 공식 개발가이드", "url": endpoint.SourceURL},
		"security":     []any{map[string]any{"crtfcKey": []any{}}}, "parameters": parameters,
		"responses": map[string]any{"default": response},
		"x-opendart": map[string]any{
			"logicalOperationId": endpoint.LogicalOperationID, "apiGroupCode": endpoint.APIGroupCode,
			"apiGroupName": endpoint.APIGroupName, "apiId": endpoint.APIID, "apiName": endpoint.Name,
			"documentedPageHeading": endpoint.PageHeading,
			"source":                map[string]any{"guideUrl": endpoint.SourceURL, "groupUrl": endpoint.GroupSourceURL, "checkedAt": checkedAt},
			"documentedBasicInfo":   basic, "documentedRequestArguments": endpoint.RequestArguments,
			"sourceTableHeaders": endpoint.SourceTableHeaders, "referenceTables": endpoint.ReferenceTables,
			"sectionNotes": endpoint.SectionNotes,
			"coverage": map[string]any{
				"status": "probe-required", "classification": "not-assessed", "acquisitionIdentity": "not-documented",
				"successfulEmptyCoverage": "not-documented", "partitionClosure": "not-documented", "historicalAvailability": "not-documented",
			},
		},
	}
	return map[string]any{"get": operation}, nil
}

func commonSchemas(codes []MessageCode) map[string]any {
	descriptions := make(map[string]any, len(codes))
	enum := make([]string, 0, len(codes))
	for _, row := range codes {
		enum = append(enum, row.Code)
		descriptions[row.Code] = row.Description
	}
	return map[string]any{
		"OpenDartStatus": map[string]any{
			"type": "string", "enum": enum, "description": "OpenDART API 수준 상태 코드입니다.",
			"x-opendart-code-descriptions": descriptions,
		},
		"OpenDartXmlError": map[string]any{
			"type":       "object",
			"properties": map[string]any{"status": map[string]any{"$ref": "#/OpenDartStatus"}, "message": map[string]any{"type": "string"}},
			"required":   []string{"status", "message"}, "additionalProperties": true,
			"xml":         map[string]any{"nodeType": "element", "name": "result"},
			"description": "ZIP 다운로드 API에서 실측된 XML API 오류 응답입니다.",
			"x-opendart":  map[string]any{"schemaStatus": "empirically-observed", "observation": zipErrorObservation},
		},
	}
}

func rootDocument(endpoints []Endpoint, paths map[string]any, seenPaths map[string]bool, schemaFiles map[string]string, rootFile, componentsFile, checkedAt string) (map[string]any, error) {
	groupCounts := make(map[string]any, len(Groups))
	for _, group := range Groups {
		seen := make(map[string]bool)
		for _, endpoint := range endpoints {
			if endpoint.APIGroupCode == group.Code {
				seen[endpoint.LogicalOperationID] = true
			}
		}
		groupCounts[group.Code] = len(seen)
	}
	tags := make([]any, 0, len(Groups))
	for _, group := range Groups {
		tags = append(tags, map[string]any{
			"name": group.Code, "description": group.Name,
			"externalDocs": map[string]any{"description": group.Name + " 개발가이드", "url": guideOrigin + "/guide/main.do?apiGrpCd=" + group.Code},
		})
	}
	componentsRef, err := relativeReference(rootFile, componentsFile)
	if err != nil {
		return nil, fmt.Errorf("generate OpenAPI root: resolve common schema reference: %w", err)
	}
	componentSchemas := map[string]any{
		"OpenDartStatus":   map[string]any{"$ref": componentsRef + "#/OpenDartStatus"},
		"OpenDartXmlError": map[string]any{"$ref": componentsRef + "#/OpenDartXmlError"},
	}
	for _, endpoint := range endpoints {
		binary := false
		for _, basic := range endpoint.BasicInfo {
			if basic.OutputFormat == "Zip FILE (binary)" {
				binary = true
				break
			}
		}
		if binary {
			schemaRef, err := relativeReference(rootFile, schemaFiles[endpoint.LogicalOperationID])
			if err != nil {
				return nil, endpointError(endpoint, "resolve binary response schema reference: %v", err)
			}
			componentSchemas[endpoint.APIGroupCode+"_"+endpoint.APIID+"_Response"] = map[string]any{"$ref": schemaRef}
		}
	}
	return map[string]any{
		"openapi": openAPIVersion,
		"info": map[string]any{
			"title": "OpenDART API", "version": checkedAt,
			"summary":     "금융감독원 OpenDART 공식 개발가이드 기반 API 명세",
			"description": "공식 OpenDART 개발가이드에서 추출한 소스 기반 명세입니다. HTTP 상태, 응답 필드 타입, 완전성 및 수집 의미처럼 가이드가 규정하지 않는 동작은 추정하지 않고 x-opendart에 명시합니다.",
		},
		"externalDocs": map[string]any{"description": "OpenDART 개발가이드", "url": guideOrigin + "/guide/main.do?apiGrpCd=DS001"},
		"servers":      []any{map[string]any{"url": apiServer, "description": "OpenDART production API"}},
		"tags":         tags, "security": []any{map[string]any{"crtfcKey": []any{}}}, "paths": paths,
		"components": map[string]any{
			"securitySchemes": map[string]any{"crtfcKey": map[string]any{
				"type": "apiKey", "in": "query", "name": "crtfc_key", "description": "OpenDART에서 발급받은 40자리 API 인증키",
				"x-opendart-documented-type": "STRING(40)",
			}},
			"schemas": componentSchemas,
		},
		"x-opendart": map[string]any{
			"source":    map[string]any{"origin": guideOrigin, "checkedAt": checkedAt},
			"inventory": map[string]any{"logicalEndpointCount": len(endpoints), "physicalPathCount": len(seenPaths), "groupCounts": groupCounts},
			"extractionPolicy": map[string]any{
				"sourceLanguage": "ko", "excluded": []string{"site chrome", "interactive test controls and transient results", "commented-out API sample sections"},
				"responseSchema": "Source hierarchy is normalized conservatively; raw rows, indentation, icons, and diagnostics are retained on every schema.",
			},
		},
	}, nil
}

func writeYAML(file string, value any) error {
	if err := os.MkdirAll(filepath.Dir(file), 0o755); err != nil {
		return err
	}
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return os.WriteFile(file, encoded, 0o644)
}
