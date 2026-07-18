package guide

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"go.yaml.in/yaml/v4"
)

const catalogOpenAPIVersion = "3.2.0"

var catalogReferenceTables = map[string]struct {
	Title string
	Rows  int
}{
	"DS001-2019001": {Title: "상세 유형", Rows: 61},
	"DS003-2020001": {Title: "재무제표구분", Rows: 26},
}

var catalogMultiCompanyEvidence = map[string]string{
	"DS003-2019017": "00334624,00126380",
	"DS003-2022002": "00164742,00159023",
}

// CatalogOptions controls validation of one generated OpenDART source tree.
// Structural mode retains all ownership, provenance, normalization, and
// reference checks while allowing an intentionally partial endpoint inventory.
type CatalogOptions struct {
	Root           string
	StructuralOnly bool
}

// CatalogReport is the stable, credential-free inventory emitted by the
// catalog boundary. Volatile details remain in the generated source itself.
type CatalogReport struct {
	Root             string         `json:"root"`
	OpenAPI          string         `json:"openapi"`
	LogicalEndpoints int            `json:"logicalEndpoints"`
	PhysicalPaths    int            `json:"physicalPaths"`
	RequestArguments int            `json:"requestArguments"`
	ResponseFields   int            `json:"responseFields"`
	MessageCodes     int            `json:"messageCodes"`
	GroupCounts      map[string]int `json:"groupCounts"`
}

// CatalogDiagnostic identifies a failed invariant without embedding source
// documents or other unbounded content in automation output.
type CatalogDiagnostic struct {
	Rule      string `json:"rule"`
	Phase     string `json:"phase"`
	Artifact  string `json:"artifact,omitempty"`
	Operation string `json:"operation,omitempty"`
	Location  string `json:"location,omitempty"`
	Message   string `json:"message"`
}

// CatalogError carries one fail-fast catalog diagnostic and preserves an
// underlying filesystem or parse error for programmatic inspection.
type CatalogError struct {
	Diagnostic CatalogDiagnostic `json:"diagnostic"`
	cause      error
}

func (e *CatalogError) Error() string {
	parts := []string{e.Diagnostic.Phase, e.Diagnostic.Rule}
	if e.Diagnostic.Operation != "" {
		parts = append(parts, "operation="+e.Diagnostic.Operation)
	}
	if e.Diagnostic.Artifact != "" {
		parts = append(parts, "artifact="+e.Diagnostic.Artifact)
	}
	if e.Diagnostic.Location != "" {
		parts = append(parts, "location="+e.Diagnostic.Location)
	}
	return strings.Join(parts, ": ") + ": " + e.Diagnostic.Message
}

func (e *CatalogError) Unwrap() error { return e.cause }

type catalogValidator struct {
	rootFile       string
	rootDir        string
	physicalRoot   string
	structuralOnly bool
	parsedFiles    map[string]any
}

type logicalCatalogOperation struct {
	APIGroupCode     string
	APIID            string
	RequestArguments []RequestArgument
	ReferenceTables  []ReferenceTable
	Formats          map[string]bool
	Paths            []string
}

// ValidateCatalog verifies the generated source tree without performing any
// network request. It intentionally matches the legacy catalog check's full
// and structural-only modes so staged synchronization can cut over in place.
func ValidateCatalog(options CatalogOptions) (CatalogReport, error) {
	if options.Root == "" {
		return CatalogReport{}, catalogFailure("options", "catalog-root", "", "", "", "catalog root is required", nil)
	}
	rootFile, err := filepath.Abs(options.Root)
	if err != nil {
		return CatalogReport{}, catalogFailure("options", "catalog-root", options.Root, "", "", "catalog root cannot be resolved", err)
	}
	rootDir := filepath.Dir(rootFile)
	physicalRoot, err := filepath.EvalSymlinks(rootDir)
	if err != nil {
		return CatalogReport{}, catalogFailure("references", "physical-root", rootDir, "", "", "specification directory cannot be resolved", err)
	}
	v := &catalogValidator{
		rootFile: rootFile, rootDir: rootDir, physicalRoot: physicalRoot,
		structuralOnly: options.StructuralOnly, parsedFiles: make(map[string]any),
	}
	return v.validate()
}

func (v *catalogValidator) validate() (CatalogReport, error) {
	if err := v.validateMarker(); err != nil {
		return CatalogReport{}, err
	}
	physicalRootFile, err := filepath.EvalSymlinks(v.rootFile)
	if err != nil {
		return CatalogReport{}, v.fail("references", "root-target", v.rootFile, "", "", "root artifact cannot be resolved", err)
	}
	expectedRootFile := filepath.Join(v.physicalRoot, filepath.Base(v.rootFile))
	if filepath.Clean(physicalRootFile) != filepath.Clean(expectedRootFile) {
		return CatalogReport{}, v.fail("references", "root-symlink", v.rootFile, "", "", "root artifact uses a symlink below the specification directory", nil)
	}
	rootValue, _, err := v.readYAML(v.rootFile)
	if err != nil {
		return CatalogReport{}, err
	}
	root, ok := rootValue.(map[string]any)
	if !ok {
		return CatalogReport{}, v.fail("root", "root-shape", v.rootFile, "", "#", "root document must be a mapping", nil)
	}
	v.parsedFiles[v.rootFile] = root
	if stringValue(root["openapi"]) != catalogOpenAPIVersion {
		return CatalogReport{}, v.fail("root", "openapi-version", v.rootFile, "", "#/openapi", "unexpected OpenAPI version", nil)
	}
	paths, ok := root["paths"].(map[string]any)
	if !ok {
		return CatalogReport{}, v.fail("root", "paths-object", v.rootFile, "", "#/paths", "root paths object is missing", nil)
	}
	catalogMetadata, ok := root["x-opendart"].(map[string]any)
	if !ok {
		return CatalogReport{}, v.fail("root", "catalog-metadata", v.rootFile, "", "#/x-opendart", "catalog metadata is missing", nil)
	}
	source, _ := catalogMetadata["source"].(map[string]any)
	checkedAt := stringValue(source["checkedAt"])
	info, _ := root["info"].(map[string]any)
	if checkedAt == "" || stringValue(info["version"]) != checkedAt {
		return CatalogReport{}, v.fail("root", "catalog-version", v.rootFile, "", "#/info/version", "catalog version and source check date differ", nil)
	}
	inventory, ok := catalogMetadata["inventory"].(map[string]any)
	if !ok {
		return CatalogReport{}, v.fail("root", "inventory", v.rootFile, "", "#/x-opendart/inventory", "catalog inventory is missing", nil)
	}
	if integerValue(inventory["physicalPathCount"]) != len(paths) {
		return CatalogReport{}, v.fail("root", "physical-path-inventory", v.rootFile, "", "#/x-opendart/inventory/physicalPathCount", "root inventory path count differs from the paths object", nil)
	}
	if !v.structuralOnly && len(paths) != ExpectedFullTotals.PhysicalPaths {
		return CatalogReport{}, v.fail("inventory", "physical-path-completeness", v.rootFile, "", "#/paths", "physical path count is incomplete", nil)
	}

	logical := make(map[string]*logicalCatalogOperation)
	operationIDs := make(map[string]bool)
	referencedPathFiles := make(map[string]bool)
	referencedSchemaFiles := make(map[string]bool)
	parameterDiagnosticCount := 0
	multiCompanySerializationCount := 0
	pendingVerificationCount := 0

	pathKeys := sortedMapKeys(paths)
	for _, pathKey := range pathKeys {
		pathReference, ok := paths[pathKey].(map[string]any)
		if !ok || len(pathReference) != 1 || stringValue(pathReference["$ref"]) == "" {
			return CatalogReport{}, v.fail("paths", "path-reference-shape", v.rootFile, "", "#/paths/"+escapePointer(pathKey), "path entry must contain one local $ref", nil)
		}
		pathFile, err := v.resolveReference(v.rootFile, stringValue(pathReference["$ref"]), "", "#/paths/"+escapePointer(pathKey)+"/$ref")
		if err != nil {
			return CatalogReport{}, err
		}
		referencedPathFiles[pathFile] = true
		pathValue, pathText, err := v.readYAML(pathFile)
		if err != nil {
			return CatalogReport{}, err
		}
		pathItem, ok := pathValue.(map[string]any)
		if !ok || len(pathItem) != 1 {
			return CatalogReport{}, v.fail("paths", "path-fragment-shape", pathFile, "", "#", "path fragment must contain one GET operation", nil)
		}
		operation, ok := pathItem["get"].(map[string]any)
		if !ok {
			return CatalogReport{}, v.fail("paths", "path-fragment-method", pathFile, "", "#/get", "path fragment must contain one GET operation", nil)
		}
		v.parsedFiles[pathFile] = pathItem
		identity, group, apiID, err := v.validateOperation(pathKey, pathFile, pathText, operation, checkedAt, operationIDs)
		if err != nil {
			return CatalogReport{}, err
		}
		parameters, _ := operation["parameters"].([]any)
		for _, value := range parameters {
			parameter, _ := value.(map[string]any)
			diagnostics, _ := parameter["x-opendart-source-diagnostics"].([]any)
			parameterDiagnosticCount += len(diagnostics)
			serialization, _ := parameter["x-opendart-serialization"].(map[string]any)
			if stringValue(serialization["status"]) == "guide-example-supported" {
				multiCompanySerializationCount++
			}
			verification, _ := serialization["authenticatedVerification"].(map[string]any)
			if stringValue(verification["status"]) == "pending" {
				pendingVerificationCount++
			}
		}
		xOpenDART := operation["x-opendart"].(map[string]any)
		requestArguments, err := decodeRequestArguments(xOpenDART["documentedRequestArguments"])
		if err != nil {
			return CatalogReport{}, v.fail("operations", "documented-request-arguments", pathFile, identity, "#/get/x-opendart/documentedRequestArguments", "documented request arguments are malformed", err)
		}
		referenceTableValue, referenceTablesPresent := xOpenDART["referenceTables"]
		_, referenceTablesSequence := referenceTableValue.([]any)
		referenceTables, err := decodeReferenceTables(referenceTableValue)
		if !referenceTablesPresent || !referenceTablesSequence || err != nil {
			return CatalogReport{}, v.fail("operations", "reference-tables", pathFile, identity, "#/get/x-opendart/referenceTables", "reference tables are malformed", err)
		}
		basic, err := decodeBasicInfo(xOpenDART["documentedBasicInfo"])
		if err != nil {
			return CatalogReport{}, v.fail("operations", "documented-basic-info", pathFile, identity, "#/get/x-opendart/documentedBasicInfo", "documented basic info is malformed", err)
		}
		current := logical[identity]
		if current == nil {
			logical[identity] = &logicalCatalogOperation{APIGroupCode: group, APIID: apiID, RequestArguments: requestArguments, ReferenceTables: referenceTables, Formats: map[string]bool{basic.OutputFormat: true}, Paths: []string{pathKey}}
		} else {
			if current.APIGroupCode != group || current.APIID != apiID || !reflect.DeepEqual(current.RequestArguments, requestArguments) || !reflect.DeepEqual(current.ReferenceTables, referenceTables) {
				return CatalogReport{}, v.fail("operations", "cross-format-metadata", pathFile, identity, "#/get/x-opendart", "logical endpoint metadata differs by format", nil)
			}
			current.Formats[basic.OutputFormat] = true
			current.Paths = append(current.Paths, pathKey)
		}
		if err := v.collectReferences(pathFile, pathItem, identity, referencedSchemaFiles); err != nil {
			return CatalogReport{}, err
		}
	}

	if integerValue(inventory["logicalEndpointCount"]) != len(logical) {
		return CatalogReport{}, v.fail("inventory", "logical-endpoint-inventory", v.rootFile, "", "#/x-opendart/inventory/logicalEndpointCount", "root inventory logical count differs from operations", nil)
	}
	if !v.structuralOnly {
		if len(logical) != ExpectedFullTotals.LogicalEndpoints {
			return CatalogReport{}, v.fail("inventory", "logical-endpoint-completeness", v.rootFile, "", "#/paths", "logical endpoint count is incomplete", nil)
		}
		if parameterDiagnosticCount != 37 || multiCompanySerializationCount != 4 || pendingVerificationCount != 4 {
			return CatalogReport{}, v.fail("inventory", "parameter-policy-counts", v.rootFile, "", "#/paths", "request parameter policy counts changed", nil)
		}
	}

	groupCounts := make(map[string]int, len(Groups))
	for _, group := range Groups {
		groupCounts[group.Code] = 0
	}
	requestArgumentCount := 0
	logicalIDs := sortedLogicalKeys(logical)
	for _, identity := range logicalIDs {
		operation := logical[identity]
		if _, known := groupCounts[operation.APIGroupCode]; !known {
			return CatalogReport{}, v.fail("inventory", "api-group", v.rootFile, identity, "#/paths", "operation uses an unknown API group", nil)
		}
		groupCounts[operation.APIGroupCode]++
		requestArgumentCount += len(operation.RequestArguments)
		formats := boolMapKeys(operation.Formats)
		wantFormats := []string{"JSON", "XML"}
		if operation.Formats["Zip FILE (binary)"] {
			wantFormats = []string{"Zip FILE (binary)"}
		}
		if !reflect.DeepEqual(formats, wantFormats) {
			return CatalogReport{}, v.fail("inventory", "representation-set", v.rootFile, identity, "#/paths", "logical endpoint representation set is incomplete", nil)
		}
		expectedReference, expected := catalogReferenceTables[identity]
		if expected {
			if len(operation.ReferenceTables) != 1 || operation.ReferenceTables[0].Title != expectedReference.Title || len(operation.ReferenceTables[0].Rows) != expectedReference.Rows {
				return CatalogReport{}, v.fail("inventory", "reference-table", v.rootFile, identity, "#/paths", "expected endpoint reference table changed", nil)
			}
		} else if len(operation.ReferenceTables) != 0 {
			return CatalogReport{}, v.fail("inventory", "reference-table", v.rootFile, identity, "#/paths", "unexpected endpoint reference table appeared", nil)
		}
	}
	rootGroupCounts, ok := intMap(inventory["groupCounts"])
	if !ok || !reflect.DeepEqual(groupCounts, rootGroupCounts) {
		return CatalogReport{}, v.fail("inventory", "group-inventory", v.rootFile, "", "#/x-opendart/inventory/groupCounts", "root inventory group counts differ from operations", nil)
	}
	if !v.structuralOnly {
		expectedGroups := make(map[string]int, len(Groups))
		for _, group := range Groups {
			expectedGroups[group.Code] = group.ExpectedCount
		}
		if !reflect.DeepEqual(groupCounts, expectedGroups) || requestArgumentCount != ExpectedFullTotals.RequestArguments {
			return CatalogReport{}, v.fail("inventory", "accepted-inventory", v.rootFile, "", "#/x-opendart/inventory", "accepted group or request-argument inventory changed", nil)
		}
	}

	messageCodes, responseFields, err := v.validateSchemas(root, logical, referencedSchemaFiles)
	if err != nil {
		return CatalogReport{}, err
	}
	if err := v.validateArtifactClosure(referencedPathFiles, referencedSchemaFiles); err != nil {
		return CatalogReport{}, err
	}
	for file, value := range v.parsedFiles {
		if err := v.validateAllReferences(file, value); err != nil {
			return CatalogReport{}, err
		}
	}
	return CatalogReport{Root: v.rootFile, OpenAPI: catalogOpenAPIVersion, LogicalEndpoints: len(logical), PhysicalPaths: len(paths), RequestArguments: requestArgumentCount, ResponseFields: responseFields, MessageCodes: messageCodes, GroupCounts: groupCounts}, nil
}

func (v *catalogValidator) validateMarker() error {
	marker := filepath.Join(v.rootDir, OutputMarker)
	info, err := os.Lstat(marker)
	if err != nil {
		return v.fail("ownership", "output-marker", marker, "", "", "generated-output ownership marker is missing", err)
	}
	if !info.Mode().IsRegular() || info.Mode()&os.ModeSymlink != 0 {
		return v.fail("ownership", "output-marker", marker, "", "", "generated-output marker must be a regular non-symlink file", nil)
	}
	content, err := os.ReadFile(marker)
	if err != nil {
		return v.fail("ownership", "output-marker", marker, "", "", "generated-output ownership marker cannot be read", err)
	}
	if string(content) != OutputMarkerContent {
		return v.fail("ownership", "output-marker", marker, "", "", "generated-output marker changed", nil)
	}
	return nil
}

func (v *catalogValidator) validateOperation(pathKey, pathFile, pathText string, operation map[string]any, checkedAt string, operationIDs map[string]bool) (string, string, string, error) {
	xOpenDART, ok := operation["x-opendart"].(map[string]any)
	if !ok {
		return "", "", "", v.fail("operations", "provenance", pathFile, "", "#/get/x-opendart", "operation provenance is missing", nil)
	}
	identity := stringValue(xOpenDART["logicalOperationId"])
	group := stringValue(xOpenDART["apiGroupCode"])
	apiID := stringValue(xOpenDART["apiId"])
	if identity == "" || stringValue(xOpenDART["documentedPageHeading"]) == "" {
		return "", "", "", v.fail("operations", "provenance", pathFile, identity, "#/get/x-opendart", "logical identity or documented page heading is missing", nil)
	}
	source, _ := xOpenDART["source"].(map[string]any)
	if stringValue(source["guideUrl"]) == "" || stringValue(source["checkedAt"]) == "" {
		return "", "", "", v.fail("operations", "source-provenance", pathFile, identity, "#/get/x-opendart/source", "source URL or checked date is missing", nil)
	}
	if stringValue(source["checkedAt"]) != checkedAt {
		return "", "", "", v.fail("operations", "source-date", pathFile, identity, "#/get/x-opendart/source/checkedAt", "operation source check date differs from the catalog", nil)
	}
	if identity != group+"-"+apiID {
		return "", "", "", v.fail("operations", "logical-identity", pathFile, identity, "#/get/x-opendart/logicalOperationId", "logical operation identity differs from its group and API ID", nil)
	}
	operationID := stringValue(operation["operationId"])
	if operationID == "" || operationIDs[operationID] {
		return "", "", "", v.fail("operations", "operation-id", pathFile, identity, "#/get/operationId", "operationId is missing or duplicated", nil)
	}
	operationIDs[operationID] = true

	headers, err := decodeSourceTableHeaders(xOpenDART["sourceTableHeaders"])
	if err != nil || !reflect.DeepEqual(headers, SourceTableHeaders{
		BasicInfo:        []string{"메서드", "요청URL", "인코딩", "출력포멧"},
		RequestArguments: []string{"요청키", "명칭", "타입", "필수여부", "값설명"},
		ResponseFields:   []string{"응답키", "명칭", "출력설명"},
	}) {
		return "", "", "", v.fail("operations", "source-table-headers", pathFile, identity, "#/get/x-opendart/sourceTableHeaders", "documented source-table headers changed", err)
	}
	basic, err := decodeBasicInfo(xOpenDART["documentedBasicInfo"])
	if err != nil || basic.Method != "GET" || basic.Encoding != "UTF-8" {
		return "", "", "", v.fail("operations", "documented-basic-info", pathFile, identity, "#/get/x-opendart/documentedBasicInfo", "documented method or encoding changed", err)
	}
	arguments, err := decodeRequestArguments(xOpenDART["documentedRequestArguments"])
	if err != nil {
		return "", "", "", v.fail("operations", "documented-request-arguments", pathFile, identity, "#/get/x-opendart/documentedRequestArguments", "documented request arguments are malformed", err)
	}
	endpoint := Endpoint{EndpointSummary: EndpointSummary{APIGroupCode: group, APIID: apiID, LogicalOperationID: identity, Description: stringValue(operation["description"])}, RequestArguments: arguments}
	if example, multi := catalogMultiCompanyEvidence[identity]; multi {
		endpoint.GuideTestRequestArguments = []GuideTestArgument{{Key: "corp_code", Value: example}}
		endpoint.MessageCodes = []MessageCode{{Code: "021", Description: "조회 가능한 회사 개수가 초과하였습니다.(최대 100건)"}}
	}
	expectedParameters, expectedErr := parameterObjects(endpoint)
	if expectedErr == nil {
		expectedParameters, expectedErr = genericCatalogSlice(expectedParameters)
	}
	if expectedErr != nil || !reflect.DeepEqual(operation["parameters"], expectedParameters) {
		location := "#/get/parameters"
		if expectedErr == nil {
			location += firstCatalogDifference(operation["parameters"], expectedParameters)
		}
		return "", "", "", v.fail("operations", "parameter-normalization", pathFile, identity, location, "OpenAPI parameters differ from the documented request arguments", expectedErr)
	}
	if !reflect.DeepEqual(operation["security"], []any{map[string]any{"crtfcKey": []any{}}}) {
		return "", "", "", v.fail("operations", "operation-security", pathFile, identity, "#/get/security", "operation security does not use the documented query authentication key", nil)
	}
	documentedURL, err := url.Parse(basic.RequestURL)
	if err != nil || documentedURL.Path != "/api"+pathKey {
		return "", "", "", v.fail("operations", "documented-path", pathFile, identity, "#/get/x-opendart/documentedBasicInfo/requestUrl", "OpenAPI path does not match the documented URL", err)
	}
	if !((strings.HasSuffix(pathKey, ".json") && basic.OutputFormat == "JSON") || (strings.HasSuffix(pathKey, ".xml") && (basic.OutputFormat == "XML" || basic.OutputFormat == "Zip FILE (binary)"))) {
		return "", "", "", v.fail("operations", "output-format-path", pathFile, identity, "#/get/responses/default", "path suffix and documented output format disagree", nil)
	}
	mediaType, err := outputMediaType(basic.OutputFormat)
	if err != nil {
		return "", "", "", v.fail("operations", "output-format", pathFile, identity, "#/get/x-opendart/documentedBasicInfo/outputFormat", "unknown documented output format", err)
	}
	responses, _ := operation["responses"].(map[string]any)
	response, _ := responses["default"].(map[string]any)
	content, _ := response["content"].(map[string]any)
	wantMedia := []string{mediaType}
	if basic.OutputFormat == "Zip FILE (binary)" {
		wantMedia = []string{"application/xml", "application/zip"}
	}
	actualMedia := sortedMapKeys(content)
	if response == nil || !reflect.DeepEqual(actualMedia, wantMedia) {
		return "", "", "", v.fail("operations", "response-media-types", pathFile, identity, "#/get/responses/default/content", "response media types differ from the documented output format", nil)
	}
	media, _ := content[mediaType].(map[string]any)
	responseSchema, responseSchemaPresent := media["schema"].(map[string]any)
	expectedSchemaFile := filepath.Join(v.rootDir, "schemas", strings.ToLower(group), apiID+".yaml")
	if basic.OutputFormat == "Zip FILE (binary)" {
		if !responseSchemaPresent || len(responseSchema) != 0 {
			return "", "", "", v.fail("operations", "zip-schema", pathFile, identity, "#/get/responses/default/content/application~1zip/schema", "ZIP response does not use the canonical raw-binary schema", nil)
		}
		xmlError, _ := content["application/xml"].(map[string]any)
		xmlSchema, _ := xmlError["schema"].(map[string]any)
		xmlRef := stringValue(xmlSchema["$ref"])
		target, resolveErr := v.resolveReference(pathFile, xmlRef, identity, "#/get/responses/default/content/application~1xml/schema/$ref")
		if resolveErr != nil || !strings.HasSuffix(xmlRef, "#/OpenDartXmlError") || target != filepath.Join(v.rootDir, "components", "schemas.yaml") {
			if resolveErr != nil {
				return "", "", "", resolveErr
			}
			return "", "", "", v.fail("operations", "zip-xml-error-schema", pathFile, identity, "#/get/responses/default/content/application~1xml", "ZIP XML error response does not use the shared empirical schema", nil)
		}
		if stringValue(xmlError["x-opendart-content-type-status"]) != "empirically-observed-error-response" || !reflect.DeepEqual(xmlError["x-opendart-observation"], zipErrorObservation) {
			return "", "", "", v.fail("operations", "zip-observation", pathFile, identity, "#/get/responses/default/content/application~1xml", "ZIP XML error observation is missing or changed", nil)
		}
		documentedSchema, _ := response["x-opendart-documented-response-schema"].(map[string]any)
		if stringValue(documentedSchema["component"]) != group+"_"+apiID+"_Response" {
			return "", "", "", v.fail("operations", "zip-documented-schema", pathFile, identity, "#/get/responses/default/x-opendart-documented-response-schema", "ZIP operation points to the wrong documented response-table component", nil)
		}
	} else {
		ref := stringValue(responseSchema["$ref"])
		if ref == "" {
			return "", "", "", v.fail("operations", "response-schema", pathFile, identity, "#/get/responses/default/content/"+escapePointer(mediaType)+"/schema", "structured response schema reference is missing", nil)
		}
		target, resolveErr := v.resolveReference(pathFile, ref, identity, "#/get/responses/default/content/"+escapePointer(mediaType)+"/schema/$ref")
		if resolveErr != nil {
			return "", "", "", resolveErr
		}
		if target != expectedSchemaFile {
			return "", "", "", v.fail("operations", "response-schema", pathFile, identity, "#/get/responses/default/content/"+escapePointer(mediaType)+"/schema/$ref", "operation points to another endpoint response schema", nil)
		}
	}
	if strings.Contains(pathText, "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx") {
		return "", "", "", v.fail("operations", "credential-placeholder", pathFile, identity, "#", "test-form credential placeholder leaked into a path file", nil)
	}
	return identity, group, apiID, nil
}

func (v *catalogValidator) validateSchemas(root map[string]any, logical map[string]*logicalCatalogOperation, referencedSchemas map[string]bool) (int, int, error) {
	components, _ := root["components"].(map[string]any)
	schemas, _ := components["schemas"].(map[string]any)
	statusReference, statusOK := schemas["OpenDartStatus"].(map[string]any)
	xmlReference, xmlOK := schemas["OpenDartXmlError"].(map[string]any)
	if !statusOK || stringValue(statusReference["$ref"]) == "" || !xmlOK || stringValue(xmlReference["$ref"]) == "" {
		return 0, 0, v.fail("schemas", "shared-components", v.rootFile, "", "#/components/schemas", "shared status or XML error schema is missing", nil)
	}
	sharedNames := map[string]bool{"OpenDartStatus": true, "OpenDartXmlError": true}
	endpointNames := make([]string, 0)
	for name := range schemas {
		if !sharedNames[name] {
			endpointNames = append(endpointNames, name)
		}
	}
	sort.Strings(endpointNames)
	binaryCount := 0
	for _, operation := range logical {
		if operation.Formats["Zip FILE (binary)"] {
			binaryCount++
		}
	}
	if len(endpointNames) != binaryCount {
		return 0, 0, v.fail("schemas", "binary-component-count", v.rootFile, "", "#/components/schemas", "binary endpoint schema component count changed", nil)
	}
	for _, name := range endpointNames {
		reference, ok := schemas[name].(map[string]any)
		if !ok || stringValue(reference["$ref"]) == "" {
			return 0, 0, v.fail("schemas", "binary-component-reference", v.rootFile, "", "#/components/schemas/"+escapePointer(name), "endpoint schema component must use a local $ref", nil)
		}
		parts := strings.Split(name, "_")
		if len(parts) != 3 || !endpointGroupPattern.MatchString(parts[0]) || !endpointIDPattern.MatchString(parts[1]) || parts[2] != "Response" {
			return 0, 0, v.fail("schemas", "binary-component-name", v.rootFile, "", "#/components/schemas/"+escapePointer(name), "binary endpoint schema component has an unexpected name", nil)
		}
		target, err := v.resolveReference(v.rootFile, stringValue(reference["$ref"]), parts[0]+"-"+parts[1], "#/components/schemas/"+escapePointer(name)+"/$ref")
		if err != nil {
			return 0, 0, err
		}
		expected := filepath.Join(v.rootDir, "schemas", strings.ToLower(parts[0]), parts[1]+".yaml")
		if target != expected {
			return 0, 0, v.fail("schemas", "binary-component-reference", v.rootFile, parts[0]+"-"+parts[1], "#/components/schemas/"+escapePointer(name)+"/$ref", "binary component points to another endpoint schema", nil)
		}
		referencedSchemas[target] = true
	}
	if len(referencedSchemas) != len(logical) {
		return 0, 0, v.fail("schemas", "schema-reference-coverage", v.rootFile, "", "#/components/schemas", "endpoint schema reference coverage is incomplete", nil)
	}

	responseFieldCount := 0
	responseDiagnosticCount := 0
	expectedDiagnosticCount := 0
	schemaFiles := sortedBoolKeys(referencedSchemas)
	for _, schemaFile := range schemaFiles {
		value, _, err := v.readYAML(schemaFile)
		if err != nil {
			return 0, 0, err
		}
		schema, ok := value.(map[string]any)
		if !ok {
			return 0, 0, v.fail("schemas", "schema-shape", schemaFile, "", "#", "endpoint schema must be a mapping", nil)
		}
		v.parsedFiles[schemaFile] = schema
		xOpenDART, _ := schema["x-opendart"].(map[string]any)
		fields, err := decodeResponseFields(xOpenDART["responseFields"])
		if err != nil || len(fields) == 0 {
			return 0, 0, v.fail("schemas", "raw-response-fields", schemaFile, "", "#/x-opendart/responseFields", "raw response-field rows are missing", err)
		}
		for index, field := range fields {
			if field.SourceIndex != index {
				return 0, 0, v.fail("schemas", "response-source-order", schemaFile, "", fmt.Sprintf("#/x-opendart/responseFields/%d/sourceIndex", index), "response-field source order is not preserved", nil)
			}
		}
		responseFieldCount += len(fields)
		roots := make([]ResponseField, 0, 1)
		for _, field := range fields {
			if field.Depth != nil && *field.Depth == 0 {
				roots = append(roots, field)
			}
		}
		if len(roots) != 1 || roots[0].Key != "result" {
			return 0, 0, v.fail("schemas", "response-root", schemaFile, "", "#/x-opendart/responseFields", "documented response root changed", nil)
		}
		xml, _ := schema["xml"].(map[string]any)
		if stringValue(xOpenDART["sourceRootKey"]) != roots[0].Key || stringValue(xml["name"]) != roots[0].Key || stringValue(xml["nodeType"]) != "element" {
			return 0, 0, v.fail("schemas", "response-xml-root", schemaFile, "", "#/xml", "response schema XML root does not match its documented source root", nil)
		}
		actualDiagnostics := extensionValues(schema, "x-opendart-source-diagnostics")
		expectedDiagnostics := v.expectedResponseDiagnostics(schemaFile)
		if !sameMembers(actualDiagnostics, expectedDiagnostics) {
			return 0, 0, v.fail("schemas", "response-source-diagnostics", schemaFile, "", "#", "response source diagnostics differ from curated contradictions", nil)
		}
		for _, value := range actualDiagnostics {
			if diagnostics, ok := value.([]any); ok {
				responseDiagnosticCount += len(diagnostics)
			}
		}
		for _, value := range expectedDiagnostics {
			if diagnostics, ok := value.([]any); ok {
				expectedDiagnosticCount += len(diagnostics)
			}
		}
	}
	if !v.structuralOnly && responseFieldCount != ExpectedFullTotals.ResponseFields {
		return 0, 0, v.fail("inventory", "response-field-completeness", v.rootFile, "", "#/paths", "response field total changed", nil)
	}
	if responseDiagnosticCount != expectedDiagnosticCount {
		return 0, 0, v.fail("schemas", "response-source-diagnostic-count", v.rootFile, "", "#/paths", "response source-diagnostic count changed", nil)
	}

	statusFile, err := v.resolveReference(v.rootFile, stringValue(statusReference["$ref"]), "", "#/components/schemas/OpenDartStatus/$ref")
	if err != nil {
		return 0, 0, err
	}
	xmlRef := stringValue(xmlReference["$ref"])
	xmlFile, err := v.resolveReference(v.rootFile, xmlRef, "", "#/components/schemas/OpenDartXmlError/$ref")
	if err != nil {
		return 0, 0, err
	}
	if !strings.HasSuffix(xmlRef, "#/OpenDartXmlError") || xmlFile != statusFile {
		return 0, 0, v.fail("schemas", "shared-xml-component", v.rootFile, "", "#/components/schemas/OpenDartXmlError", "shared XML error component points to an unexpected schema", nil)
	}
	sharedValue, _, err := v.readYAML(statusFile)
	if err != nil {
		return 0, 0, err
	}
	shared, ok := sharedValue.(map[string]any)
	if !ok {
		return 0, 0, v.fail("schemas", "shared-schema-shape", statusFile, "", "#", "shared schema file must be a mapping", nil)
	}
	v.parsedFiles[statusFile] = shared
	status, _ := shared["OpenDartStatus"].(map[string]any)
	enum, _ := status["enum"].([]any)
	descriptions, _ := status["x-opendart-code-descriptions"].(map[string]any)
	if len(enum) != ExpectedFullTotals.MessageCodes || len(descriptions) != ExpectedFullTotals.MessageCodes {
		return 0, 0, v.fail("schemas", "message-code-inventory", statusFile, "", "#/OpenDartStatus", "message-code inventory or descriptions changed", nil)
	}
	expectedXML, normalizeErr := genericCatalogValue(commonSchemas(nil)["OpenDartXmlError"])
	if normalizeErr != nil {
		return 0, 0, v.fail("schemas", "xml-error-schema", statusFile, "", "#/OpenDartXmlError", "shared XML error policy cannot be normalized", normalizeErr)
	}
	if !reflect.DeepEqual(shared["OpenDartXmlError"], expectedXML) {
		return 0, 0, v.fail("schemas", "xml-error-schema", statusFile, "", "#/OpenDartXmlError", "shared XML error schema changed", nil)
	}
	return len(enum), responseFieldCount, nil
}

func (v *catalogValidator) expectedResponseDiagnostics(schemaFile string) []any {
	relative, err := filepath.Rel(v.rootDir, schemaFile)
	if err != nil {
		return nil
	}
	var identity string
	switch filepath.ToSlash(relative) {
	case "schemas/ds001/2019001.yaml":
		identity = "DS001-2019001"
	case "schemas/ds002/2019011.yaml":
		identity = "DS002-2019011"
	default:
		return nil
	}
	result := make([]any, 0)
	for key, evidence := range responseFieldConflicts {
		if !strings.HasPrefix(key, identity+":") {
			continue
		}
		fieldKey := strings.TrimPrefix(key, identity+":")
		diagnostics := responseFieldSourceDiagnostics(Endpoint{EndpointSummary: EndpointSummary{LogicalOperationID: identity}}, []ResponseField{{Key: fieldKey, Name: evidence.Name, Description: evidence.Description}})
		result = append(result, diagnostics)
	}
	return result
}

func (v *catalogValidator) collectReferences(file string, value any, operation string, referencedSchemas map[string]bool) error {
	for _, reference := range catalogReferences(value, "#") {
		if !reference.Valid {
			return v.fail("references", "reference-type", file, operation, reference.Pointer, "$ref must be a string", nil)
		}
		target, err := v.resolveReference(file, reference.Value, operation, reference.Pointer)
		if err != nil {
			return err
		}
		if within(filepath.Join(v.rootDir, "schemas"), target) {
			referencedSchemas[target] = true
		}
	}
	return nil
}

func (v *catalogValidator) validateAllReferences(file string, value any) error {
	for _, reference := range catalogReferences(value, "#") {
		if !reference.Valid {
			return v.fail("references", "reference-type", file, "", reference.Pointer, "$ref must be a string", nil)
		}
		if _, err := v.resolveReference(file, reference.Value, "", reference.Pointer); err != nil {
			return err
		}
	}
	return nil
}

func (v *catalogValidator) readYAML(file string) (any, string, error) {
	data, err := os.ReadFile(file)
	if err != nil {
		return nil, "", v.fail("parse", "read-yaml", file, "", "", "YAML artifact cannot be read", err)
	}
	var value any
	if err := yaml.Unmarshal(data, &value); err != nil {
		return nil, "", v.fail("parse", "yaml", file, "", "", "YAML parsing failed", err)
	}
	return normalizeCatalogValue(value), string(data), nil
}

func (v *catalogValidator) resolveReference(fromFile, reference, operation, location string) (string, error) {
	if hasCatalogURIScheme(reference) {
		return "", v.fail("references", "reference-uri", fromFile, operation, location, "URI-scheme $ref is forbidden", nil)
	}
	filePart := strings.SplitN(reference, "#", 2)[0]
	if strings.HasPrefix(filePart, "//") || filepath.IsAbs(filepath.FromSlash(filePart)) {
		return "", v.fail("references", "reference-absolute", fromFile, operation, location, "absolute $ref is forbidden", nil)
	}
	target := fromFile
	if filePart != "" {
		target = filepath.Clean(filepath.Join(filepath.Dir(fromFile), filepath.FromSlash(filePart)))
	}
	if !within(v.rootDir, target) {
		return "", v.fail("references", "reference-escape", fromFile, operation, location, "$ref escapes the OpenDART specification directory", nil)
	}
	physicalTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		return "", v.fail("references", "reference-target", fromFile, operation, location, "$ref target cannot be resolved", err)
	}
	if !within(v.physicalRoot, physicalTarget) {
		return "", v.fail("references", "reference-physical-escape", fromFile, operation, location, "$ref resolves outside the OpenDART specification directory", nil)
	}
	relativeTarget, err := filepath.Rel(v.rootDir, target)
	if err != nil {
		return "", v.fail("references", "reference-target", fromFile, operation, location, "$ref target cannot be resolved relative to the specification directory", err)
	}
	expectedPhysicalTarget := filepath.Join(v.physicalRoot, relativeTarget)
	if filepath.Clean(physicalTarget) != filepath.Clean(expectedPhysicalTarget) {
		return "", v.fail("references", "reference-symlink", fromFile, operation, location, "$ref uses a symlink below the specification directory", nil)
	}
	info, err := os.Stat(physicalTarget)
	if err != nil {
		return "", v.fail("references", "reference-target", fromFile, operation, location, "$ref target cannot be inspected", err)
	}
	if !info.Mode().IsRegular() {
		return "", v.fail("references", "reference-target", fromFile, operation, location, "$ref target is not a regular file", nil)
	}
	return target, nil
}

func (v *catalogValidator) fail(phase, rule, artifact, operation, location, message string, cause error) error {
	return catalogFailure(phase, rule, artifact, operation, location, message, cause)
}

func catalogFailure(phase, rule, artifact, operation, location, message string, cause error) error {
	return &CatalogError{Diagnostic: CatalogDiagnostic{Rule: rule, Phase: phase, Artifact: artifact, Operation: operation, Location: location, Message: message}, cause: cause}
}

type catalogReference struct {
	Pointer string
	Value   string
	Valid   bool
}

func catalogReferences(value any, pointer string) []catalogReference {
	result := make([]catalogReference, 0)
	var visit func(any, string)
	visit = func(current any, currentPointer string) {
		switch typed := current.(type) {
		case []any:
			for index, child := range typed {
				visit(child, fmt.Sprintf("%s/%d", currentPointer, index))
			}
		case map[string]any:
			keys := sortedMapKeys(typed)
			for _, key := range keys {
				childPointer := currentPointer + "/" + escapePointer(key)
				child := typed[key]
				if key == "$ref" {
					text, ok := child.(string)
					result = append(result, catalogReference{Pointer: childPointer, Value: text, Valid: ok})
				}
				visit(child, childPointer)
			}
		}
	}
	visit(value, pointer)
	return result
}

func extensionValues(value any, key string) []any {
	result := make([]any, 0)
	var visit func(any)
	visit = func(current any) {
		switch typed := current.(type) {
		case []any:
			for _, child := range typed {
				visit(child)
			}
		case map[string]any:
			for childKey, child := range typed {
				if childKey == key {
					result = append(result, child)
				}
				visit(child)
			}
		}
	}
	visit(value)
	return result
}

func sameMembers(left, right []any) bool {
	if len(left) != len(right) {
		return false
	}
	remaining := append([]any(nil), right...)
	for _, value := range left {
		found := -1
		for index, candidate := range remaining {
			if reflect.DeepEqual(value, candidate) {
				found = index
				break
			}
		}
		if found < 0 {
			return false
		}
		remaining = append(remaining[:found], remaining[found+1:]...)
	}
	return true
}

func yamlFilesBelow(directory string) ([]string, error) {
	files := make([]string, 0)
	err := filepath.WalkDir(directory, func(path string, entry os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".yaml") {
			files = append(files, path)
		}
		return nil
	})
	sort.Strings(files)
	return files, err
}

func decodeBasicInfo(value any) (BasicInfo, error) {
	var result BasicInfo
	err := decodeCatalogValue(value, &result)
	return result, err
}

func decodeRequestArguments(value any) ([]RequestArgument, error) {
	var result []RequestArgument
	err := decodeCatalogValue(value, &result)
	return result, err
}

func decodeReferenceTables(value any) ([]ReferenceTable, error) {
	var result []ReferenceTable
	err := decodeCatalogValue(value, &result)
	return result, err
}

func decodeResponseFields(value any) ([]ResponseField, error) {
	var result []ResponseField
	err := decodeCatalogValue(value, &result)
	return result, err
}

func decodeSourceTableHeaders(value any) (SourceTableHeaders, error) {
	var result SourceTableHeaders
	err := decodeCatalogValue(value, &result)
	return result, err
}

func decodeCatalogValue(value, result any) error {
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return err
	}
	return yaml.Unmarshal(encoded, result)
}

func genericCatalogValue(value any) (any, error) {
	encoded, err := yaml.Marshal(value)
	if err != nil {
		return nil, err
	}
	var result any
	if err := yaml.Unmarshal(encoded, &result); err != nil {
		return nil, err
	}
	return normalizeCatalogValue(result), nil
}

func genericCatalogSlice(value []any) ([]any, error) {
	normalized, err := genericCatalogValue(value)
	if err != nil {
		return nil, err
	}
	result, ok := normalized.([]any)
	if !ok {
		return nil, fmt.Errorf("normalized catalog value is %T, want sequence", normalized)
	}
	return result, nil
}

func normalizeCatalogValue(value any) any {
	switch typed := value.(type) {
	case time.Time:
		if typed.Hour() == 0 && typed.Minute() == 0 && typed.Second() == 0 && typed.Nanosecond() == 0 {
			return typed.Format(time.DateOnly)
		}
		return typed.Format(time.RFC3339Nano)
	case map[string]any:
		for key, child := range typed {
			typed[key] = normalizeCatalogValue(child)
		}
	case []any:
		for index, child := range typed {
			typed[index] = normalizeCatalogValue(child)
		}
	}
	return value
}

func sortedMapKeys(value map[string]any) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedBoolKeys(value map[string]bool) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func sortedLogicalKeys(value map[string]*logicalCatalogOperation) []string {
	keys := make([]string, 0, len(value))
	for key := range value {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}

func boolMapKeys(value map[string]bool) []string { return sortedBoolKeys(value) }

func stringValue(value any) string {
	text, _ := value.(string)
	return text
}

func integerValue(value any) int {
	switch typed := value.(type) {
	case int:
		return typed
	case int64:
		return int(typed)
	case uint64:
		return int(typed)
	case float64:
		if typed == float64(int(typed)) {
			return int(typed)
		}
		return -1
	default:
		return -1
	}
}

func intMap(value any) (map[string]int, bool) {
	input, ok := value.(map[string]any)
	if !ok {
		return nil, false
	}
	result := make(map[string]int, len(input))
	for key, raw := range input {
		count := integerValue(raw)
		if count < 0 {
			return nil, false
		}
		result[key] = count
	}
	return result, true
}

func escapePointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func within(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}

func hasCatalogURIScheme(value string) bool {
	for index, character := range value {
		if character == ':' {
			return index > 0
		}
		if !(character >= 'A' && character <= 'Z') && !(character >= 'a' && character <= 'z') && !(index > 0 && character >= '0' && character <= '9') && !(index > 0 && (character == '+' || character == '-' || character == '.')) {
			return false
		}
	}
	return false
}

func firstCatalogDifference(actual, expected any) string {
	if reflect.DeepEqual(actual, expected) {
		return ""
	}
	actualMap, actualMapOK := actual.(map[string]any)
	expectedMap, expectedMapOK := expected.(map[string]any)
	if actualMapOK && expectedMapOK {
		keys := make(map[string]bool, len(actualMap)+len(expectedMap))
		for key := range actualMap {
			keys[key] = true
		}
		for key := range expectedMap {
			keys[key] = true
		}
		ordered := make([]string, 0, len(keys))
		for key := range keys {
			ordered = append(ordered, key)
		}
		sort.Strings(ordered)
		for _, key := range ordered {
			actualValue, actualOK := actualMap[key]
			expectedValue, expectedOK := expectedMap[key]
			if !actualOK || !expectedOK || !reflect.DeepEqual(actualValue, expectedValue) {
				return "/" + escapePointer(key) + firstCatalogDifference(actualValue, expectedValue)
			}
		}
	}
	actualSlice, actualSliceOK := actual.([]any)
	expectedSlice, expectedSliceOK := expected.([]any)
	if actualSliceOK && expectedSliceOK {
		length := min(len(actualSlice), len(expectedSlice))
		for index := range length {
			if !reflect.DeepEqual(actualSlice[index], expectedSlice[index]) {
				return fmt.Sprintf("/%d%s", index, firstCatalogDifference(actualSlice[index], expectedSlice[index]))
			}
		}
		return fmt.Sprintf("/%d", length)
	}
	return ""
}

func (v *catalogValidator) validateArtifactClosure(pathFiles, schemaFiles map[string]bool) error {
	actualPaths, err := yamlFilesBelow(filepath.Join(v.rootDir, "paths"))
	if err != nil {
		return v.fail("artifacts", "path-files", filepath.Join(v.rootDir, "paths"), "", "", "path fragments cannot be enumerated", err)
	}
	if !reflect.DeepEqual(actualPaths, sortedBoolKeys(pathFiles)) {
		return v.fail("artifacts", "path-file-closure", filepath.Join(v.rootDir, "paths"), "", "", "orphaned or missing path fragments detected", nil)
	}
	actualSchemas, err := yamlFilesBelow(filepath.Join(v.rootDir, "schemas"))
	if err != nil {
		return v.fail("artifacts", "schema-files", filepath.Join(v.rootDir, "schemas"), "", "", "schema fragments cannot be enumerated", err)
	}
	if !reflect.DeepEqual(actualSchemas, sortedBoolKeys(schemaFiles)) {
		return v.fail("artifacts", "schema-file-closure", filepath.Join(v.rootDir, "schemas"), "", "", "orphaned or missing schema fragments detected", nil)
	}
	return nil
}
