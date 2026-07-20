package rust

import (
	"strings"
	"testing"

	"github.com/cpaikr/opendart/internal/sdkgen/model"
)

func TestRenderUsesLintCleanParameterConstruction(t *testing.T) {
	source := model.Model{
		SchemaVersion: model.SchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Logical: []model.LogicalOperation{
			{
				ID:       "optional",
				RustName: "OptionalInput",
				Group:    "group",
				Parameters: []model.Parameter{
					{WireName: "page_no", RustName: "page_no", Shape: model.ScalarString},
				},
				Variants: []model.PhysicalReference{{OperationID: "optional.json", Representation: model.RepresentationJSON}},
			},
			{
				ID:       "required",
				RustName: "RequiredInput",
				Group:    "group",
				Parameters: []model.Parameter{
					{WireName: "corp_code", RustName: "corp_code", Required: true, Shape: model.ScalarString},
				},
				Variants: []model.PhysicalReference{{OperationID: "required.json", Representation: model.RepresentationJSON}},
			},
		},
		Physical: []model.PhysicalOperation{
			{OperationID: "optional.json", LogicalID: "optional", RustConstant: "OPTIONAL_JSON", Path: "/api/optional.json", PrimaryRepresentation: model.RepresentationJSON, ExpectedRepresentations: []model.Representation{model.RepresentationJSON}, Responses: []model.Response{{Selector: "default", HTTPStatusEvidence: "not-documented", Media: []model.ResponseMedia{{Name: "application/json", ContentTypeStatus: "inferred-from-documented-output-format", Shape: model.ResponseShape{Kind: "object", Description: "source description", AdditionalPropertiesPolicy: "allowed"}}}}}},
			{OperationID: "required.json", LogicalID: "required", RustConstant: "REQUIRED_JSON", Path: "/api/required.json", PrimaryRepresentation: model.RepresentationJSON, ExpectedRepresentations: []model.Representation{model.RepresentationJSON}, Responses: []model.Response{{Selector: "default", HTTPStatusEvidence: "not-documented", Media: []model.ResponseMedia{{Name: "application/json", ContentTypeStatus: "inferred-from-documented-output-format", Shape: model.ResponseShape{Kind: "object", AdditionalPropertiesPolicy: "allowed"}}}}}},
		},
	}

	files, err := Render(source)
	if err != nil {
		t.Fatalf("Render() error = %v", err)
	}
	generated := string(files["operations/group.rs"])
	if !strings.Contains(generated, "#[derive(Clone, Debug, Default, Eq, PartialEq)]\npub struct OptionalInput") {
		t.Fatal("optional-only operation input does not derive Default")
	}
	if !strings.Contains(generated, "let parameters = vec![") {
		t.Fatal("required-only operation does not use a vector literal")
	}
	if strings.Contains(generated, "let mut parameters = Vec::with_capacity(1);\n        parameters.push") {
		t.Fatal("required-only operation uses push-based vector initialization")
	}
	responses := string(files["responses/group.rs"])
	if !strings.Contains(responses, "pub struct OptionalInputJsonResponse") || !strings.Contains(responses, "source description") {
		t.Fatal("typed response or selected description was not emitted")
	}
	if !strings.Contains(responses, `#[cfg_attr(feature = "serde-json", derive(serde::Serialize))]`) {
		t.Fatal("typed responses do not gate serialization on the public feature")
	}
	if !strings.Contains(responses, `#[cfg_attr(feature = "serde-json", serde(flatten))]`) {
		t.Fatal("typed responses do not flatten additive source fields")
	}
}

func TestRenderEscapesRustStringsAndArrayInputs(t *testing.T) {
	source := model.Model{
		SchemaVersion: model.SchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Logical: []model.LogicalOperation{{
			ID: "array", RustName: "ArrayInput", Group: "group",
			Parameters: []model.Parameter{{WireName: "corp_code", RustName: "corp_code", Required: true, Shape: model.StringArray, MinItems: int64Pointer(1), MaxItems: int64Pointer(100)}},
			Variants:   []model.PhysicalReference{{OperationID: "array.json", Representation: model.RepresentationJSON}},
		}},
		Physical: []model.PhysicalOperation{{
			OperationID: "array.json", LogicalID: "array", RustConstant: "ARRAY_JSON", Path: "/api/array.json", PrimaryRepresentation: model.RepresentationJSON, ExpectedRepresentations: []model.Representation{model.RepresentationJSON},
			Responses: []model.Response{{Selector: "default", HTTPStatusEvidence: "not-documented", Media: []model.ResponseMedia{{Name: "application/json", ContentTypeStatus: "inferred-from-documented-output-format", Shape: model.ResponseShape{Kind: "object", Description: "back\bseparator\u2028", AdditionalPropertiesPolicy: "allowed"}}}}},
		}},
	}

	files, err := Render(source)
	if err != nil {
		t.Fatal(err)
	}
	operation := string(files["operations/group.rs"])
	if !strings.Contains(operation, `for value in &self.corp_code { require_nonempty(identity, "corp_code", value)?; }`) {
		t.Fatal("array elements are not checked for empty strings")
	}
	responses := string(files["responses/group.rs"])
	if !strings.Contains(responses, `back\u{8}separator\u{2028}`) {
		t.Fatalf("response description was not retained safely:\n%s", responses)
	}
}

func TestRenderUsesRepresentationAwareArrayDecoding(t *testing.T) {
	arrayShape := model.ResponseShape{
		Kind: "object",
		Properties: []model.ResponseProperty{{
			Name: "items", RustName: "items",
			Shape: model.ResponseShape{Kind: "array", Items: &model.ResponseShape{Kind: "opaque"}},
		}},
	}
	media := func(name string) []model.Response {
		shape := arrayShape
		if name == "application/xml" {
			shape.XMLName = "result"
			shape.XMLNodeType = "element"
		}
		return []model.Response{{Selector: "default", Media: []model.ResponseMedia{{
			Name: name, ContentTypeStatus: "inferred-from-documented-output-format", Shape: shape,
		}}}}
	}
	source := model.Model{
		SchemaVersion: model.SchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Logical: []model.LogicalOperation{{
			ID: "items", RustName: "Items", Group: "group",
			Variants: []model.PhysicalReference{
				{OperationID: "items.json", Representation: model.RepresentationJSON},
				{OperationID: "items.xml", Representation: model.RepresentationXML},
			},
		}},
		Physical: []model.PhysicalOperation{
			{OperationID: "items.json", LogicalID: "items", RustConstant: "ITEMS_JSON", Path: "/api/items.json", PrimaryRepresentation: model.RepresentationJSON, ExpectedRepresentations: []model.Representation{model.RepresentationJSON}, Responses: media("application/json")},
			{OperationID: "items.xml", LogicalID: "items", RustConstant: "ITEMS_XML", Path: "/api/items.xml", PrimaryRepresentation: model.RepresentationXML, ExpectedRepresentations: []model.Representation{model.RepresentationXML}, Responses: media("application/xml")},
		},
	}

	files, err := Render(source)
	if err != nil {
		t.Fatal(err)
	}
	responses := string(files["responses/group.rs"])
	if !strings.Contains(responses, "decode_array(value, path, decode_source_value)") {
		t.Fatal("JSON array decoder no longer requires an explicit source array")
	}
	if !strings.Contains(responses, "decode_xml_array(value, path, decode_source_value)") {
		t.Fatal("XML array decoder does not normalize a singleton element")
	}
	if !strings.Contains(responses, "ObjectDecoder::new(value, path)") {
		t.Fatal("JSON object decoder no longer requires an explicit source object")
	}
	if !strings.Contains(responses, "ObjectDecoder::new_xml(value, path)") {
		t.Fatal("XML object decoder does not normalize an empty element")
	}
}

func TestRenderOmitsStructuredRequestImportForBinaryOnlyGroup(t *testing.T) {
	source := model.Model{
		SchemaVersion: model.SchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Logical: []model.LogicalOperation{{
			ID: "archive", RustName: "Archive", Group: "group",
			Variants: []model.PhysicalReference{{OperationID: "archive.xml", Representation: model.RepresentationZIP}},
		}},
		Physical: []model.PhysicalOperation{{
			OperationID: "archive.xml", LogicalID: "archive", RustConstant: "ARCHIVE_XML", Path: "/api/archive.xml",
			PrimaryRepresentation: model.RepresentationZIP, ExpectedRepresentations: []model.Representation{model.RepresentationZIP, model.RepresentationXML},
			Responses: []model.Response{{Selector: "default", Media: []model.ResponseMedia{
				{Name: "application/zip", Shape: model.ResponseShape{Kind: "binary"}},
				{Name: "application/xml", Shape: model.ResponseShape{Kind: "object", XMLName: "result", XMLNodeType: "element"}},
			}}},
		}},
	}

	files, err := Render(source)
	if err != nil {
		t.Fatal(err)
	}
	operation := string(files["operations/group.rs"])
	if strings.Contains(operation, "PreparedRequest") {
		t.Fatalf("binary-only operation imports a structured request:\n%s", operation)
	}
	if !strings.Contains(operation, "PreparedBinaryRequest") {
		t.Fatalf("binary-only operation omits its request type:\n%s", operation)
	}
	if !strings.Contains(operation, `Some("result")`) {
		t.Fatalf("binary operation omits its alternate XML root:\n%s", operation)
	}
}

func TestRenderRejectsPublicRustSymbolCollisions(t *testing.T) {
	base := model.Model{
		SchemaVersion: model.SchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Logical: []model.LogicalOperation{{
			ID: "logical", RustName: "Input", Group: "group",
			Variants: []model.PhysicalReference{{OperationID: "input.json", Representation: model.RepresentationJSON}},
		}},
		Physical: []model.PhysicalOperation{{
			OperationID: "input.json", LogicalID: "logical", RustConstant: "INPUT_JSON", Path: "/api/input.json",
			PrimaryRepresentation: model.RepresentationJSON, ExpectedRepresentations: []model.Representation{model.RepresentationJSON},
			Responses: []model.Response{{Selector: "default", Media: []model.ResponseMedia{{Name: "application/json", ContentTypeStatus: "inferred-from-documented-output-format", Shape: model.ResponseShape{Kind: "object"}}}}},
		}},
	}

	t.Run("request runtime import", func(t *testing.T) {
		source := base
		source.Logical = append([]model.LogicalOperation(nil), base.Logical...)
		source.Logical[0].RustName = "PreparedRequest"
		if _, err := Render(source); err == nil || !strings.Contains(err.Error(), "runtime import") {
			t.Fatalf("error = %v", err)
		}
	})

	t.Run("response evidence accessor", func(t *testing.T) {
		source := base
		source.Physical = append([]model.PhysicalOperation(nil), base.Physical...)
		shape := &source.Physical[0].Responses[0].Media[0].Shape
		shape.Properties = []model.ResponseProperty{{
			Name: "additional_fields", RustName: "additional_fields", Shape: model.ResponseShape{Kind: "opaque"},
		}}
		if _, err := Render(source); err == nil || !strings.Contains(err.Error(), "evidence accessors") {
			t.Fatalf("error = %v", err)
		}
	})
}

func int64Pointer(value int64) *int64 { return &value }
