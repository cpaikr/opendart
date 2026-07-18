package openapi

import (
	"archive/zip"
	"bytes"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"mime"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"time"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	validatorconfig "github.com/pb33f/libopenapi-validator/config"
	validatorerrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
	"github.com/pb33f/libopenapi/datamodel/high/v3"
	"go.yaml.in/yaml/v4"
)

const (
	maxCompatibilityArchiveEntries = 64
	maxCompatibilityArchiveBytes   = 8 << 20
)

// Document is the repository-owned OpenAPI boundary. Third-party model and
// validator types stay private so later tooling is not coupled to a library API.
type Document struct {
	root      string
	document  libopenapi.Document
	model     *libopenapi.DocumentModel[v3.Document]
	validator validator.Validator
}

type Comparison struct {
	TotalChanges    int            `json:"totalChanges"`
	BreakingChanges int            `json:"breakingChanges"`
	Details         []ChangeDetail `json:"details,omitempty"`
}

type ChangeDetail struct {
	Location string `json:"location"`
	Original string `json:"original,omitempty"`
	New      string `json:"new,omitempty"`
}

type CompatibilityReport struct {
	Root                    string `json:"root"`
	OpenAPI                 string `json:"openapi"`
	DocumentValid           bool   `json:"documentValid"`
	RenderDeterministic     bool   `json:"renderDeterministic"`
	BundleDeterministic     bool   `json:"bundleDeterministic"`
	BundleSemanticChanges   int    `json:"bundleSemanticChanges"`
	BaselineSemanticChanges int    `json:"baselineSemanticChanges"`
}

func Load(root string) (*Document, error) {
	absoluteRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve root: %w", err)
	}
	files, err := validateLocalReferences(absoluteRoot)
	if err != nil {
		return nil, err
	}
	document, model, err := loadModel(absoluteRoot, files)
	if err != nil {
		return nil, err
	}
	responseValidator, validationSetupErrors := validator.NewValidator(
		document,
		validatorconfig.WithXmlBodyValidation(),
	)
	if len(validationSetupErrors) > 0 {
		document.Release()
		return nil, fmt.Errorf("initialize validator for %s: %s", absoluteRoot, joinErrors(validationSetupErrors))
	}
	return &Document{
		root:      absoluteRoot,
		document:  document,
		model:     model,
		validator: responseValidator,
	}, nil
}

func (d *Document) Close() {
	if d == nil {
		return
	}
	if d.validator != nil {
		d.validator.Release()
	}
	if d.document != nil {
		d.document.Release()
	}
}

func (d *Document) Version() string {
	return d.document.GetVersion()
}

func (d *Document) ValidateDocument() error {
	valid, validationErrors := d.validator.ValidateDocument()
	if valid {
		return nil
	}
	return validationError("OpenAPI document "+d.root, validationErrors)
}

func (d *Document) LintCompatibility() error {
	if err := d.ValidateDocument(); err != nil {
		return err
	}
	if d.model.Model.Paths == nil || d.model.Model.Paths.PathItems == nil {
		return nil
	}
	for pathName, pathItem := range d.model.Model.Paths.PathItems.FromOldest() {
		for method, operation := range pathItem.GetOperations().FromOldest() {
			if operation == nil || operation.Responses == nil {
				continue
			}
			for code, response := range operation.Responses.Codes.FromOldest() {
				if response != nil && response.Description == "" {
					return fmt.Errorf("lint OpenAPI document %s: %s %s response %s has no description", d.root, method, pathName, code)
				}
			}
			if response := operation.Responses.Default; response != nil && response.Description == "" {
				return fmt.Errorf("lint OpenAPI document %s: %s %s default response has no description", d.root, method, pathName)
			}
		}
	}
	return nil
}

func (d *Document) Render() ([]byte, error) {
	rendered, err := d.model.Model.Render()
	if err != nil {
		return nil, fmt.Errorf("render %s: %w", d.root, err)
	}
	return rendered, nil
}

func (d *Document) Bundle() ([]byte, error) {
	return d.bundle(false)
}

func (d *Document) bundle(inlineLocalReferences bool) ([]byte, error) {
	files, err := validateLocalReferences(d.root)
	if err != nil {
		return nil, err
	}
	document, model, err := loadModel(d.root, files)
	if err != nil {
		return nil, err
	}
	defer document.Release()
	bundled, err := bundler.BundleDocumentWithConfig(&model.Model, &bundler.BundleInlineConfig{
		InlineLocalRefs: &inlineLocalReferences,
	})
	if err != nil {
		return nil, fmt.Errorf("bundle %s: %w", d.root, err)
	}
	return bundled, nil
}

func (d *Document) ValidateResponse(method, path, contentType string, status int, body []byte) error {
	mediaType, _, err := mime.ParseMediaType(contentType)
	if err != nil {
		return errors.New("validate response: Content-Type is malformed")
	}
	contentType = strings.ToLower(mediaType)
	if contentType == "application/zip" {
		if !d.declaresResponseMedia(method, path, status, contentType) {
			return fmt.Errorf("%s %s response %d does not declare %s", method, path, status, contentType)
		}
		return validateZIP(body)
	}
	if contentType == "application/xml" {
		expectedRoot := d.expectedXMLRoot(method, path, status)
		if expectedRoot != "" {
			if err := validateXMLRoot(body, expectedRoot); err != nil {
				return fmt.Errorf("validate %s %s response representation: %w", method, path, err)
			}
		}
	}

	request, err := http.NewRequest(method, "https://opendart.invalid"+path, nil)
	if err != nil {
		return fmt.Errorf("build validation request: %w", err)
	}
	response := &http.Response{
		StatusCode: status,
		Header:     http.Header{"Content-Type": []string{contentType}},
		Body:       io.NopCloser(bytes.NewReader(body)),
	}
	valid, validationErrors := d.validator.ValidateHttpResponse(request, response)
	if valid {
		return nil
	}
	return validationError(fmt.Sprintf("%s %s response", method, path), validationErrors)
}

func (d *Document) declaresResponseMedia(method, path string, status int, contentType string) bool {
	return d.responseMedia(method, path, status, contentType) != nil
}

func (d *Document) responseMedia(method, path string, status int, contentType string) *v3.MediaType {
	if d.model.Model.Paths == nil || d.model.Model.Paths.PathItems == nil {
		return nil
	}
	pathItem := d.model.Model.Paths.PathItems.GetOrZero(path)
	if pathItem == nil {
		return nil
	}
	var operation *v3.Operation
	switch strings.ToUpper(method) {
	case http.MethodGet:
		operation = pathItem.Get
	case http.MethodPost:
		operation = pathItem.Post
	default:
		return nil
	}
	if operation == nil || operation.Responses == nil {
		return nil
	}
	response := operation.Responses.FindResponseByCode(status)
	if response == nil {
		response = operation.Responses.Default
	}
	if response == nil || response.Content == nil {
		return nil
	}
	return response.Content.GetOrZero(contentType)
}

func (d *Document) expectedXMLRoot(method, path string, status int) string {
	media := d.responseMedia(method, path, status, "application/xml")
	if media == nil || media.Schema == nil {
		return ""
	}
	schema := media.Schema.Schema()
	if schema == nil || schema.XML == nil {
		return ""
	}
	return schema.XML.Name
}

func Compare(leftRoot, rightRoot string) (Comparison, error) {
	return compareWithComponents(leftRoot, rightRoot, nil)
}

func compareBundleBaseline(sourceRoot, bundleRoot string) (Comparison, error) {
	components, err := declaredComponents(sourceRoot)
	if err != nil {
		return Comparison{}, err
	}
	return compareWithComponents(sourceRoot, bundleRoot, components)
}

func compareWithComponents(leftRoot, rightRoot string, components map[string]map[string]bool) (Comparison, error) {
	left, err := canonicalComparisonDocument(leftRoot, components)
	if err != nil {
		return Comparison{}, fmt.Errorf("load original: %w", err)
	}
	defer left.document.Release()
	right, err := canonicalComparisonDocument(rightRoot, components)
	if err != nil {
		return Comparison{}, fmt.Errorf("load updated: %w", err)
	}
	defer right.document.Release()

	changes, compareErrors := libopenapi.CompareDocuments(left.document, right.document)
	if compareErrors != nil {
		return Comparison{}, fmt.Errorf("compare documents: %w", compareErrors)
	}
	details := semanticChanges(left.value, right.value, "#")
	comparison := Comparison{
		TotalChanges:    len(details),
		BreakingChanges: changes.TotalBreakingChanges(),
		Details:         details,
	}
	return comparison, nil
}

func RunCompatibilityGate(root, baseline string) (CompatibilityReport, error) {
	document, err := Load(root)
	if err != nil {
		return CompatibilityReport{}, err
	}
	defer document.Close()
	if document.Version() != "3.2.0" {
		return CompatibilityReport{}, fmt.Errorf("%s uses OpenAPI %s, want 3.2.0", root, document.Version())
	}
	if err := document.LintCompatibility(); err != nil {
		return CompatibilityReport{}, err
	}

	renderedA, err := document.Render()
	if err != nil {
		return CompatibilityReport{}, err
	}
	renderedB, err := document.Render()
	if err != nil {
		return CompatibilityReport{}, err
	}
	if !bytes.Equal(renderedA, renderedB) {
		return CompatibilityReport{}, errors.New("OpenAPI rendering is nondeterministic")
	}
	bundleA, err := document.Bundle()
	if err != nil {
		return CompatibilityReport{}, err
	}
	bundleB, err := document.Bundle()
	if err != nil {
		return CompatibilityReport{}, err
	}
	if !bytes.Equal(bundleA, bundleB) {
		return CompatibilityReport{}, errors.New("OpenAPI bundling is nondeterministic")
	}

	temporaryDirectory, err := os.MkdirTemp("", "opendart-go-bundle-")
	if err != nil {
		return CompatibilityReport{}, fmt.Errorf("create bundle comparison directory: %w", err)
	}
	defer os.RemoveAll(temporaryDirectory)
	bundlePath := filepath.Join(temporaryDirectory, "openapi.bundle.yaml")
	if err := os.WriteFile(bundlePath, bundleA, 0o600); err != nil {
		return CompatibilityReport{}, fmt.Errorf("write temporary bundle: %w", err)
	}
	bundleComparison, err := Compare(root, bundlePath)
	if err != nil {
		return CompatibilityReport{}, err
	}
	if bundleComparison.TotalChanges != 0 {
		return CompatibilityReport{}, fmt.Errorf("Go bundle changed OpenAPI meaning: %s", comparisonSummary(bundleComparison))
	}

	report := CompatibilityReport{
		Root:                  document.root,
		OpenAPI:               document.Version(),
		DocumentValid:         true,
		RenderDeterministic:   true,
		BundleDeterministic:   true,
		BundleSemanticChanges: bundleComparison.TotalChanges,
	}
	if baseline != "" {
		baselineComparison, err := compareBundleBaseline(root, baseline)
		if err != nil {
			return CompatibilityReport{}, err
		}
		report.BaselineSemanticChanges = baselineComparison.TotalChanges
		if baselineComparison.TotalChanges != 0 {
			return CompatibilityReport{}, fmt.Errorf("accepted baseline differs semantically: %s", comparisonSummary(baselineComparison))
		}
	}
	return report, nil
}

func comparisonSummary(comparison Comparison) string {
	details := make([]string, 0, min(5, len(comparison.Details)))
	for _, detail := range comparison.Details[:min(5, len(comparison.Details))] {
		details = append(details, fmt.Sprintf("%s: %s -> %s", detail.Location, detail.Original, detail.New))
	}
	if len(details) == 0 {
		return fmt.Sprintf("%d changes", comparison.TotalChanges)
	}
	return fmt.Sprintf("%d changes (%s)", comparison.TotalChanges, strings.Join(details, "; "))
}

type canonicalDocument struct {
	document libopenapi.Document
	value    any
}

func canonicalComparisonDocument(root string, components map[string]map[string]bool) (canonicalDocument, error) {
	document, err := Load(root)
	if err != nil {
		return canonicalDocument{}, err
	}
	defer document.Close()
	bundled, err := document.bundle(true)
	if err != nil {
		return canonicalDocument{}, err
	}
	var semanticValue any
	if err := yaml.Unmarshal(bundled, &semanticValue); err != nil {
		return canonicalDocument{}, fmt.Errorf("decode canonical bundle for %s: %w", root, err)
	}
	semanticValue = normalizeOpenAPIValue(semanticValue)
	if components != nil {
		filterComponents(semanticValue, components)
	}
	canonicalJSON, err := json.Marshal(semanticValue)
	if err != nil {
		return canonicalDocument{}, fmt.Errorf("encode canonical bundle for %s: %w", root, err)
	}
	canonical, err := libopenapi.NewDocument(canonicalJSON)
	if err != nil {
		return canonicalDocument{}, fmt.Errorf("parse canonical bundle for %s: %w", root, err)
	}
	if _, err := canonical.BuildV3Model(); err != nil {
		canonical.Release()
		return canonicalDocument{}, fmt.Errorf("build canonical bundle for %s: %w", root, err)
	}
	return canonicalDocument{document: canonical, value: semanticValue}, nil
}

func semanticChanges(left, right any, location string) []ChangeDetail {
	if reflect.DeepEqual(left, right) {
		return nil
	}
	leftMap, leftIsMap := left.(map[string]any)
	rightMap, rightIsMap := right.(map[string]any)
	if leftIsMap && rightIsMap {
		keys := make(map[string]bool, len(leftMap)+len(rightMap))
		for key := range leftMap {
			keys[key] = true
		}
		for key := range rightMap {
			keys[key] = true
		}
		orderedKeys := make([]string, 0, len(keys))
		for key := range keys {
			orderedKeys = append(orderedKeys, key)
		}
		sort.Strings(orderedKeys)
		var changes []ChangeDetail
		for _, key := range orderedKeys {
			leftValue, leftExists := leftMap[key]
			rightValue, rightExists := rightMap[key]
			childLocation := location + "/" + escapeJSONPointer(key)
			if !leftExists || !rightExists {
				changes = append(changes, ChangeDetail{
					Location: childLocation,
					Original: summarizeValue(leftValue, leftExists),
					New:      summarizeValue(rightValue, rightExists),
				})
				continue
			}
			changes = append(changes, semanticChanges(leftValue, rightValue, childLocation)...)
		}
		return changes
	}
	leftArray, leftIsArray := left.([]any)
	rightArray, rightIsArray := right.([]any)
	if leftIsArray && rightIsArray {
		var changes []ChangeDetail
		length := max(len(leftArray), len(rightArray))
		for index := range length {
			leftExists := index < len(leftArray)
			rightExists := index < len(rightArray)
			childLocation := fmt.Sprintf("%s/%d", location, index)
			if !leftExists || !rightExists {
				var leftValue, rightValue any
				if leftExists {
					leftValue = leftArray[index]
				}
				if rightExists {
					rightValue = rightArray[index]
				}
				changes = append(changes, ChangeDetail{
					Location: childLocation,
					Original: summarizeValue(leftValue, leftExists),
					New:      summarizeValue(rightValue, rightExists),
				})
				continue
			}
			changes = append(changes, semanticChanges(leftArray[index], rightArray[index], childLocation)...)
		}
		return changes
	}
	return []ChangeDetail{{
		Location: location,
		Original: summarizeValue(left, true),
		New:      summarizeValue(right, true),
	}}
}

func summarizeValue(value any, exists bool) string {
	if !exists {
		return "<missing>"
	}
	switch value.(type) {
	case map[string]any:
		return "{object}"
	case []any:
		return "[array]"
	}
	encoded, err := json.Marshal(value)
	if err != nil {
		return fmt.Sprintf("<%T>", value)
	}
	return string(encoded)
}

func escapeJSONPointer(value string) string {
	return strings.ReplaceAll(strings.ReplaceAll(value, "~", "~0"), "/", "~1")
}

func declaredComponents(root string) (map[string]map[string]bool, error) {
	source, err := os.ReadFile(root)
	if err != nil {
		return nil, fmt.Errorf("read component inventory from %s: %w", root, err)
	}
	var document map[string]any
	if err := yaml.Unmarshal(source, &document); err != nil {
		return nil, fmt.Errorf("parse component inventory from %s: %w", root, err)
	}
	inventory := make(map[string]map[string]bool)
	componentObject, _ := document["components"].(map[string]any)
	for category, value := range componentObject {
		entries, ok := value.(map[string]any)
		if !ok {
			continue
		}
		inventory[category] = make(map[string]bool, len(entries))
		for name := range entries {
			inventory[category][name] = true
		}
	}
	return inventory, nil
}

func filterComponents(value any, inventory map[string]map[string]bool) {
	document, ok := value.(map[string]any)
	if !ok {
		return
	}
	componentObject, ok := document["components"].(map[string]any)
	if !ok {
		return
	}
	for category, value := range componentObject {
		entries, ok := value.(map[string]any)
		if !ok {
			continue
		}
		allowed := inventory[category]
		for name := range entries {
			if !allowed[name] {
				delete(entries, name)
			}
		}
	}
}

func normalizeOpenAPIValue(value any) any {
	switch typed := value.(type) {
	case time.Time:
		if typed.Hour() == 0 && typed.Minute() == 0 && typed.Second() == 0 && typed.Nanosecond() == 0 {
			return typed.Format(time.DateOnly)
		}
		return typed.Format(time.RFC3339Nano)
	case map[string]any:
		for key, child := range typed {
			typed[key] = normalizeOpenAPIValue(child)
		}
		return typed
	case []any:
		for index, child := range typed {
			typed[index] = normalizeOpenAPIValue(child)
		}
		return typed
	default:
		return value
	}
}

func loadModel(root string, files []string) (libopenapi.Document, *libopenapi.DocumentModel[v3.Document], error) {
	source, err := os.ReadFile(root)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", root, err)
	}
	configuration := &datamodel.DocumentConfiguration{
		BasePath:                filepath.Dir(root),
		SpecFilePath:            root,
		FileFilter:              files,
		AllowFileReferences:     true,
		AllowRemoteReferences:   false,
		ExtractRefsSequentially: true,
		Logger:                  slog.New(slog.NewTextHandler(io.Discard, nil)),
	}
	document, err := libopenapi.NewDocumentWithConfiguration(source, configuration)
	if err != nil {
		return nil, nil, fmt.Errorf("parse %s: %w", root, err)
	}
	model, err := document.BuildV3Model()
	if err != nil {
		document.Release()
		return nil, nil, fmt.Errorf("build %s: %w", root, err)
	}
	return document, model, nil
}

func validationError(subject string, validationErrors []*validatorerrors.ValidationError) error {
	if len(validationErrors) == 0 {
		return fmt.Errorf("validate %s: validator rejected input without diagnostics", subject)
	}
	messages := make([]string, 0, len(validationErrors))
	for _, validationErr := range validationErrors {
		parts := []string{validationErr.ValidationType}
		if validationErr.ValidationSubType != "" {
			parts[0] += "/" + validationErr.ValidationSubType
		}
		if validationErr.SpecPath != "" {
			parts = append(parts, "spec="+validationErr.SpecPath)
		}
		if validationErr.SpecLine > 0 {
			parts = append(parts, fmt.Sprintf("line=%d", validationErr.SpecLine))
		}
		for _, schemaErr := range validationErr.SchemaValidationErrors {
			if schemaErr.KeywordLocation != "" {
				parts = append(parts, "schema="+schemaErr.KeywordLocation)
			}
		}
		messages = append(messages, strings.Join(parts, ", "))
	}
	return fmt.Errorf("validate %s: %d issue(s): %s", subject, len(validationErrors), strings.Join(messages, "; "))
}

func joinErrors(errs []error) string {
	messages := make([]string, 0, len(errs))
	for _, err := range errs {
		messages = append(messages, err.Error())
	}
	return strings.Join(messages, "; ")
}

func validateZIP(body []byte) error {
	reader, err := zip.NewReader(bytes.NewReader(body), int64(len(body)))
	if err != nil {
		return errors.New("validate ZIP response: archive is malformed")
	}
	if len(reader.File) == 0 {
		return errors.New("validate ZIP response: archive is empty")
	}
	if len(reader.File) > maxCompatibilityArchiveEntries {
		return fmt.Errorf("validate ZIP response: archive has %d entries, limit is %d", len(reader.File), maxCompatibilityArchiveEntries)
	}
	var totalBytes uint64
	foundXML := false
	for index, file := range reader.File {
		archiveName := strings.ReplaceAll(file.Name, "\\", "/")
		cleanName := path.Clean(archiveName)
		firstSegment := strings.SplitN(cleanName, "/", 2)[0]
		if strings.HasPrefix(cleanName, "/") || cleanName == ".." || strings.HasPrefix(cleanName, "../") || strings.Contains(firstSegment, ":") {
			return fmt.Errorf("validate ZIP response: entry %d has an unsafe path", index)
		}
		if file.FileInfo().IsDir() {
			continue
		}
		if file.UncompressedSize64 > uint64(maxCompatibilityArchiveBytes)-totalBytes {
			return fmt.Errorf("validate ZIP response: expanded archive exceeds %d bytes", maxCompatibilityArchiveBytes)
		}
		totalBytes += file.UncompressedSize64
		entry, err := file.Open()
		if err != nil {
			return fmt.Errorf("validate ZIP response: entry %d cannot be opened", index)
		}
		remaining := maxCompatibilityArchiveBytes - int(totalBytes-file.UncompressedSize64)
		data, readErr := io.ReadAll(io.LimitReader(entry, int64(remaining)+1))
		closeErr := entry.Close()
		if readErr != nil {
			return fmt.Errorf("validate ZIP response: entry %d cannot be read", index)
		}
		if closeErr != nil {
			return fmt.Errorf("validate ZIP response: entry %d cannot be closed", index)
		}
		if len(data) > remaining {
			return fmt.Errorf("validate ZIP response: expanded archive exceeds %d bytes", maxCompatibilityArchiveBytes)
		}
		switch strings.ToLower(filepath.Ext(file.Name)) {
		case ".xml", ".xbrl":
			foundXML = true
			decoder := xml.NewDecoder(bytes.NewReader(data))
			for {
				if _, err := decoder.Token(); err != nil {
					if errors.Is(err, io.EOF) {
						break
					}
					return fmt.Errorf("validate ZIP response: XML entry %d is malformed", index)
				}
			}
		}
	}
	if !foundXML {
		return errors.New("validate ZIP response: archive contains no XML document")
	}
	return nil
}

func validateXMLRoot(body []byte, expected string) error {
	decoder := xml.NewDecoder(bytes.NewReader(body))
	for {
		token, err := decoder.Token()
		if err != nil {
			if errors.Is(err, io.EOF) {
				return errors.New("XML response has no root element")
			}
			return errors.New("XML response is malformed")
		}
		if start, ok := token.(xml.StartElement); ok {
			if start.Name.Local != expected {
				return errors.New("XML root does not match the declared schema")
			}
			return nil
		}
	}
}

func validateLocalReferences(root string) ([]string, error) {
	physicalRoot, err := filepath.EvalSymlinks(filepath.Dir(root))
	if err != nil {
		return nil, fmt.Errorf("resolve OpenAPI directory: %w", err)
	}
	pending := []string{root}
	visited := make(map[string]bool)
	var files []string
	for len(pending) > 0 {
		current := pending[len(pending)-1]
		pending = pending[:len(pending)-1]
		if visited[current] {
			continue
		}
		visited[current] = true
		source, err := os.ReadFile(current)
		if err != nil {
			return nil, fmt.Errorf("read referenced OpenAPI file %s: %w", current, err)
		}
		var node yaml.Node
		if err := yaml.Unmarshal(source, &node); err != nil {
			return nil, fmt.Errorf("parse referenced OpenAPI file %s: %w", current, err)
		}
		for _, reference := range references(&node) {
			filePart := strings.SplitN(reference, "#", 2)[0]
			if filePart == "" {
				continue
			}
			if hasURIScheme(filePart) || strings.HasPrefix(filePart, "//") || filepath.IsAbs(filePart) {
				return nil, fmt.Errorf("reference %q in %s is not a confined local reference", reference, current)
			}
			target := filepath.Clean(filepath.Join(filepath.Dir(current), filepath.FromSlash(filePart)))
			if !isWithin(filepath.Dir(root), target) {
				return nil, fmt.Errorf("reference %q in %s escapes the OpenAPI directory", reference, current)
			}
			physicalTarget, err := filepath.EvalSymlinks(target)
			if err != nil {
				return nil, fmt.Errorf("resolve reference %q in %s: %w", reference, current, err)
			}
			if !isWithin(physicalRoot, physicalTarget) {
				return nil, fmt.Errorf("reference %q in %s resolves outside the OpenAPI directory", reference, current)
			}
			info, err := os.Stat(physicalTarget)
			if err != nil {
				return nil, fmt.Errorf("stat reference %q in %s: %w", reference, current, err)
			}
			if !info.Mode().IsRegular() {
				return nil, fmt.Errorf("reference %q in %s does not target a regular file", reference, current)
			}
			pending = append(pending, target)
		}
	}
	for file := range visited {
		relative, err := filepath.Rel(filepath.Dir(root), file)
		if err != nil {
			return nil, fmt.Errorf("relativize %s: %w", file, err)
		}
		files = append(files, filepath.ToSlash(relative))
	}
	sort.Strings(files)
	return files, nil
}

func references(node *yaml.Node) []string {
	var result []string
	var visit func(*yaml.Node)
	visit = func(current *yaml.Node) {
		if current.Kind == yaml.MappingNode {
			for index := 0; index+1 < len(current.Content); index += 2 {
				key := current.Content[index]
				value := current.Content[index+1]
				if key.Value == "$ref" && value.Kind == yaml.ScalarNode {
					result = append(result, value.Value)
				}
				visit(value)
			}
			return
		}
		for _, child := range current.Content {
			visit(child)
		}
	}
	visit(node)
	return result
}

func hasURIScheme(reference string) bool {
	for index, character := range reference {
		if character == ':' {
			return index > 0
		}
		if !(character >= 'A' && character <= 'Z') &&
			!(character >= 'a' && character <= 'z') &&
			!(index > 0 && character >= '0' && character <= '9') &&
			!(index > 0 && (character == '+' || character == '-' || character == '.')) {
			return false
		}
	}
	return false
}

func isWithin(root, target string) bool {
	relative, err := filepath.Rel(root, target)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator)) && !filepath.IsAbs(relative)
}
