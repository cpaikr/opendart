package openapi

import (
	"errors"
	"path/filepath"
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/base"
	"github.com/pb33f/libopenapi/datamodel/high/v3"
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
