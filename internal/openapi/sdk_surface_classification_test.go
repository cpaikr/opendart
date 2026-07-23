package openapi

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/high/v3"
	"github.com/pb33f/libopenapi/orderedmap"
	"go.yaml.in/yaml/v4"
)

func TestSDKSchemaClassificationFailsClosed(t *testing.T) {
	defaultValue := &yaml.Node{Kind: yaml.ScalarNode, Value: "implicit"}
	if got := unsupportedSDKParameterSchema(&base.Schema{Type: []string{"string"}, Default: defaultValue}, false); got != "default" {
		t.Fatalf("request default classified as %q", got)
	}
	if got := unsupportedSDKParameterSchema(&base.Schema{Type: []string{"array"}, UniqueItems: boolPointer(true)}, false); got != "uniqueItems" {
		t.Fatalf("request uniqueItems classified as %q", got)
	}
	ordinaryEnum := &base.Schema{Type: []string{"string"}, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Value: "closed"}}}
	if got := unsupportedSDKResponseSchema(ordinaryEnum, false); got != "enum" {
		t.Fatalf("ordinary response enum classified as %q", got)
	}
	if got := unsupportedSDKResponseSchema(ordinaryEnum, true); got != "" {
		t.Fatalf("OpenDartStatus enum classified as %q", got)
	}
	if got := unsupportedSDKResponseSchema(&base.Schema{Type: []string{"string"}, Nullable: boolPointer(true)}, false); got != "nullable" {
		t.Fatalf("nullable response classified as %q", got)
	}
	if got := unsupportedSDKResponseSchema(&base.Schema{Type: []string{"string"}, Description: "emitted", Format: "source-format"}, false); got != "" {
		t.Fatalf("supported response annotations classified as %q", got)
	}
}

func TestDecimalRangeExtensionIsRequestOnly(t *testing.T) {
	extensions := orderedmap.New[string, *yaml.Node]()
	extensions.Set("x-opendart-decimal-range", &yaml.Node{Kind: yaml.MappingNode})
	if !supportedSDKParameterSchemaExtensions(extensions) {
		t.Fatal("request decimal range extension was rejected")
	}
	if supportedSDKSchemaExtensions(extensions) {
		t.Fatal("request-only decimal range extension was accepted on a response schema")
	}
}

func TestRequestConstraintClassificationRejectsDiscardedOrMalformedValues(t *testing.T) {
	one := int64(1)
	decimalRange := orderedmap.New[string, *yaml.Node]()
	decimalRange.Set("x-opendart-decimal-range", &yaml.Node{Kind: yaml.MappingNode})
	for name, schema := range map[string]*base.Schema{
		"array format":          {Type: []string{"array"}, Format: "opendart-date"},
		"array enum":            {Type: []string{"array"}, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value"}}},
		"array min length":      {Type: []string{"array"}, MinLength: &one},
		"integer format":        {Type: []string{"integer"}, Format: "opendart-date"},
		"integer enum":          {Type: []string{"integer"}, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!str", Value: "value"}}},
		"integer min length":    {Type: []string{"integer"}, MinLength: &one},
		"integer max length":    {Type: []string{"integer"}, MaxLength: &one},
		"integer decimal range": {Type: []string{"integer"}, Extensions: decimalRange},
	} {
		if got := unsupportedSDKParameterSchema(schema, false); got == "" {
			t.Fatalf("%s was accepted and would be discarded", name)
		}
		if _, err := inspectSDKStringConstraints(schema); err == nil {
			t.Fatalf("%s was silently discarded by constraint inspection", name)
		}
	}

	invalidEnum := &base.Schema{Type: []string{"string"}, Enum: []*yaml.Node{{Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"}}}
	if _, err := inspectSDKStringConstraints(invalidEnum); err == nil {
		t.Fatal("non-string enum scalar was accepted")
	}

	extensions := orderedmap.New[string, *yaml.Node]()
	extensions.Set("x-opendart-decimal-range", &yaml.Node{Kind: yaml.MappingNode, Content: []*yaml.Node{
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "minimum"}, {Kind: yaml.ScalarNode, Tag: "!!int", Value: "1"},
		{Kind: yaml.ScalarNode, Tag: "!!str", Value: "future"}, {Kind: yaml.ScalarNode, Tag: "!!int", Value: "2"},
	}})
	if _, err := inspectSDKStringConstraints(&base.Schema{Type: []string{"string"}, Extensions: extensions}); err == nil {
		t.Fatal("unknown decimal-range field was accepted")
	}
	if got := unsupportedSDKParameterSchema(&base.Schema{Type: []string{"array"}, Extensions: extensions}, false); got != "x-opendart-decimal-range" {
		t.Fatalf("array decimal range classified as %q", got)
	}
}

func boolPointer(value bool) *bool { return &value }

func TestSDKSurfaceRejectsRequestBodies(t *testing.T) {
	document, err := Load(filepath.Join("..", "..", "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()

	for pathName, pathItem := range document.model.Model.Paths.PathItems.FromOldest() {
		for method, operation := range pathItem.GetOperations().FromOldest() {
			if operation == nil {
				continue
			}
			operation.RequestBody = &v3.RequestBody{}
			_, err := document.InspectSDKSurface()
			var surfaceError *SDKSurfaceError
			if !errors.As(err, &surfaceError) || surfaceError.Rule != "unsupported-request-body" || surfaceError.Operation != operation.OperationId || surfaceError.Location != pathName+"/"+method+"/requestBody" {
				t.Fatalf("error = %#v", err)
			}
			return
		}
	}
	t.Fatal("canonical OpenAPI contains no operation")
}
