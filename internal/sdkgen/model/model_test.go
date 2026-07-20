package model_test

import (
	"errors"
	"path/filepath"
	"runtime"
	"slices"
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/sdkgen/model"
)

func TestBuildCoversCanonicalPhysicalInventoryDeterministically(t *testing.T) {
	surface := canonicalSurface(t)
	first, err := model.Build(surface)
	if err != nil {
		t.Fatal(err)
	}
	second, err := model.Build(surface)
	if err != nil {
		t.Fatal(err)
	}
	if first.Checksum == "" || first.Checksum != second.Checksum {
		t.Fatalf("projection checksums = %q and %q", first.Checksum, second.Checksum)
	}

	sourceIDs := make([]string, 0, len(surface.Operations))
	for _, operation := range surface.Operations {
		sourceIDs = append(sourceIDs, operation.OperationID)
	}
	generatedIDs := make([]string, 0, len(first.Physical))
	for _, operation := range first.Physical {
		generatedIDs = append(generatedIDs, operation.OperationID)
	}
	slices.Sort(sourceIDs)
	if !slices.Equal(sourceIDs, generatedIDs) {
		t.Fatal("normalized physical inventory differs from the canonical projection")
	}

	surface.Operations[0].SourceCheckedAt = "future-date-that-must-not-affect-generated-code"
	withoutVolatileProvenance, err := model.Build(surface)
	if err != nil {
		t.Fatal(err)
	}
	if withoutVolatileProvenance.Checksum != first.Checksum {
		t.Fatal("source checked-at date changed the SDK projection checksum")
	}
}

func TestBuildPreservesCommaSerializationAndZIPErrorRouting(t *testing.T) {
	generated, err := model.Build(canonicalSurface(t))
	if err != nil {
		t.Fatal(err)
	}
	multi := findLogical(t, generated, "DS003-2019017")
	corpCode := findParameter(t, multi, "corp_code")
	if corpCode.Shape != model.StringArray || corpCode.Explode || corpCode.MinItems == nil || *corpCode.MinItems != 1 || corpCode.MaxItems == nil || *corpCode.MaxItems != 100 {
		t.Fatalf("multi-company parameter = %#v", corpCode)
	}
	zip := findPhysical(t, generated, "get_corpCode_xml")
	if zip.PrimaryRepresentation != model.RepresentationZIP || !slices.Equal(zip.ExpectedRepresentations, []model.Representation{model.RepresentationZIP, model.RepresentationXML}) {
		t.Fatalf("ZIP routing = %#v", zip.ExpectedRepresentations)
	}
	if zip.Responses[0].Media[0].Shape.XMLName != "result" || zip.Responses[0].Media[0].Shape.XMLNodeType != "element" {
		t.Fatalf("ZIP XML error metadata = %#v", zip.Responses[0].Media[0].Shape)
	}
	company := findPhysical(t, generated, "get_company_json")
	status := findShapePath(t, company.Responses[0].Media[0].Shape, "status")
	if !status.OpenStatus || len(status.StatusValues) == 0 {
		t.Fatalf("open source status shape = %#v", status)
	}
	if company.Responses[0].Media[0].Shape.AdditionalPropertiesPolicy != "allowed" {
		t.Fatalf("company root additional-properties policy = %q", company.Responses[0].Media[0].Shape.AdditionalPropertiesPolicy)
	}
	if findShapePath(t, company.Responses[0].Media[0].Shape, "corp_name").Description == "" {
		t.Fatal("selected response property description was not retained")
	}
	for _, operation := range generated.Physical {
		if operation.PrimaryRepresentation == model.RepresentationZIP && !slices.Equal(operation.ExpectedRepresentations, []model.Representation{model.RepresentationZIP, model.RepresentationXML}) {
			t.Fatalf("ZIP operation %q routing = %#v", operation.OperationID, operation.ExpectedRepresentations)
		}
	}
}

func TestBuildFailsClosedOnUnsupportedOrContradictoryInputs(t *testing.T) {
	tests := []struct {
		name string
		edit func(*openapispec.SDKSurfaceOperation)
		rule string
	}{
		{name: "style", edit: func(operation *openapispec.SDKSurfaceOperation) { operation.Parameters[0].Style = "spaceDelimited" }, rule: "unsupported-parameter-serialization"},
		{name: "security scopes", edit: func(operation *openapispec.SDKSurfaceOperation) {
			operation.Security[0].Schemes[0].Scopes = []string{"write"}
		}, rule: "unsupported-authentication"},
		{name: "trusted target", edit: func(operation *openapispec.SDKSurfaceOperation) {
			operation.RelativeTarget = "https://example.invalid/path"
		}, rule: "untrusted-target"},
		{name: "allow reserved", edit: func(operation *openapispec.SDKSurfaceOperation) {
			operation.Parameters[0].AllowReserved = true
		}, rule: "unsupported-parameter-serialization"},
		{name: "unsafe query name", edit: func(operation *openapispec.SDKSurfaceOperation) {
			operation.Parameters[0].Name += "&injected"
		}, rule: "unsafe-parameter-name"},
		{name: "unknown routing evidence", edit: func(operation *openapispec.SDKSurfaceOperation) {
			operation.Responses[0].MediaTypes[0].ContentTypeStatus = "future-classification"
		}, rule: "unsupported-content-type-evidence"},
		{name: "missing required property", edit: func(operation *openapispec.SDKSurfaceOperation) {
			operation.Responses[0].MediaTypes[0].Schema.Required = append(operation.Responses[0].MediaTypes[0].Schema.Required, "future")
		}, rule: "unsupported-response-schema"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			surface := canonicalSurface(t)
			operation := firstOperationWithParameters(t, &surface)
			test.edit(operation)
			_, err := model.Build(surface)
			var modelError *model.Error
			if !errors.As(err, &modelError) || modelError.Rule != test.rule || modelError.Operation == "" || modelError.Location == "" {
				t.Fatalf("error = %#v", err)
			}
		})
	}
}

func TestBuildRejectsInvalidTopLevelXMLMetadata(t *testing.T) {
	for _, test := range []struct {
		name string
		edit func(*openapispec.SDKSurfaceSchema)
	}{
		{name: "missing root", edit: func(schema *openapispec.SDKSurfaceSchema) { schema.XMLName = "" }},
		{name: "non-element root", edit: func(schema *openapispec.SDKSurfaceSchema) { schema.XMLNodeType = "attribute" }},
	} {
		t.Run(test.name, func(t *testing.T) {
			surface := canonicalSurface(t)
			for operationIndex := range surface.Operations {
				for responseIndex := range surface.Operations[operationIndex].Responses {
					for mediaIndex := range surface.Operations[operationIndex].Responses[responseIndex].MediaTypes {
						media := &surface.Operations[operationIndex].Responses[responseIndex].MediaTypes[mediaIndex]
						if media.Name != "application/xml" {
							continue
						}
						test.edit(&media.Schema)
						_, err := model.Build(surface)
						assertModelRule(t, err, "unsupported-xml-root")
						return
					}
				}
			}
			t.Fatal("canonical surface has no XML response")
		})
	}
}

func TestBuildRejectsIncompatibleLogicalMetadataAndRoutingMatrix(t *testing.T) {
	t.Run("logical metadata", func(t *testing.T) {
		surface := canonicalSurface(t)
		first := surface.Operations[0]
		for index := range surface.Operations {
			if surface.Operations[index].LogicalOperationID == first.LogicalOperationID && surface.Operations[index].OperationID != first.OperationID {
				surface.Operations[index].APIID = "different"
				_, err := model.Build(surface)
				assertModelRule(t, err, "incompatible-logical-metadata")
				return
			}
		}
		t.Fatal("canonical surface has no paired logical operation")
	})

	t.Run("unsupported representation set", func(t *testing.T) {
		surface := canonicalSurface(t)
		operation := firstOperationWithParameters(t, &surface)
		media := operation.Responses[0].MediaTypes[0]
		media.Name = "application/xml"
		media.ContentTypeStatus = "empirically-observed-error-response"
		operation.Responses[0].MediaTypes = append(operation.Responses[0].MediaTypes, media)
		_, err := model.Build(surface)
		assertModelRule(t, err, "unsupported-response-routing")
	})
}

func TestBuildRejectsGeneratedMethodCollisionsAndEscapesRust2024Keywords(t *testing.T) {
	t.Run("preparation method", func(t *testing.T) {
		surface := canonicalSurface(t)
		operation := firstOperationWithParameters(t, &surface)
		operation.Parameters[0].Name = "prepare_json"
		_, err := model.Build(surface)
		assertModelRule(t, err, "rust-name-collision")
	})

	t.Run("private assembly method", func(t *testing.T) {
		surface := canonicalSurface(t)
		operation := firstOperationWithParameters(t, &surface)
		operation.Parameters[0].Name = "prepare_parts"
		_, err := model.Build(surface)
		assertModelRule(t, err, "rust-name-collision")
	})

	t.Run("getter and builder", func(t *testing.T) {
		surface := canonicalSurface(t)
		operation := firstOperationWithParameters(t, &surface)
		operation.Parameters[0].Name = "foo"
		operation.Parameters[0].Required = false
		other := operation.Parameters[0]
		other.Name = "with_foo"
		operation.Parameters = append(operation.Parameters, other)
		_, err := model.Build(surface)
		assertModelRule(t, err, "rust-name-collision")
	})

	t.Run("edition 2024 keyword", func(t *testing.T) {
		surface := canonicalSurface(t)
		operation := firstOperationWithParameters(t, &surface)
		logicalID := operation.LogicalOperationID
		for index := range surface.Operations {
			if surface.Operations[index].LogicalOperationID == logicalID {
				surface.Operations[index].Parameters[0].Name = "gen"
			}
		}
		generated, err := model.Build(surface)
		if err != nil {
			t.Fatal(err)
		}
		logical := findLogical(t, generated, logicalID)
		if parameter := findParameter(t, logical, "gen"); parameter.RustName != "gen_" {
			t.Fatalf("Rust name = %q, want gen_", parameter.RustName)
		}
	})
}

func assertModelRule(t *testing.T, err error, rule string) {
	t.Helper()
	var modelError *model.Error
	if !errors.As(err, &modelError) || modelError.Rule != rule || modelError.Operation == "" || modelError.Location == "" {
		t.Fatalf("error = %#v, want rule %q", err, rule)
	}
}

func canonicalSurface(t *testing.T) openapispec.SDKSurface {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate test source")
	}
	root := filepath.Join(filepath.Dir(current), "..", "..", "..", "openapi", "openapi.yaml")
	document, err := openapispec.Load(root)
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	surface, err := document.InspectSDKSurface()
	if err != nil {
		t.Fatal(err)
	}
	return surface
}

func firstOperationWithParameters(t *testing.T, surface *openapispec.SDKSurface) *openapispec.SDKSurfaceOperation {
	t.Helper()
	for index := range surface.Operations {
		if len(surface.Operations[index].Parameters) != 0 {
			return &surface.Operations[index]
		}
	}
	t.Fatal("canonical surface has no parameterized operation")
	return nil
}

func findLogical(t *testing.T, generated model.Model, id string) model.LogicalOperation {
	t.Helper()
	for _, operation := range generated.Logical {
		if operation.ID == id {
			return operation
		}
	}
	t.Fatalf("logical operation %q not found", id)
	return model.LogicalOperation{}
}

func findPhysical(t *testing.T, generated model.Model, id string) model.PhysicalOperation {
	t.Helper()
	for _, operation := range generated.Physical {
		if operation.OperationID == id {
			return operation
		}
	}
	t.Fatalf("physical operation %q not found", id)
	return model.PhysicalOperation{}
}

func findParameter(t *testing.T, operation model.LogicalOperation, name string) model.Parameter {
	t.Helper()
	for _, parameter := range operation.Parameters {
		if parameter.WireName == name {
			return parameter
		}
	}
	t.Fatalf("parameter %q not found", name)
	return model.Parameter{}
}

func findShapePath(t *testing.T, shape model.ResponseShape, name string) model.ResponseShape {
	t.Helper()
	for _, property := range shape.Properties {
		if property.Name == name {
			return property.Shape
		}
	}
	t.Fatalf("response property %q not found", name)
	return model.ResponseShape{}
}
