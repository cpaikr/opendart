package rust

import (
	"slices"
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
			{OperationID: "optional.json", LogicalID: "optional", RustConstant: "OPTIONAL_JSON", Path: "/api/optional.json", ExpectedRepresentations: []model.Representation{model.RepresentationJSON}, Responses: []model.Response{{Selector: "default", HTTPStatusEvidence: "not-documented", Media: []model.ResponseMedia{{Name: "application/json", ContentTypeStatus: "inferred-from-documented-output-format", Shape: model.ResponseShape{Kind: "opaque", Description: "source description", AdditionalPropertiesPolicy: "unspecified"}}}}}},
			{OperationID: "required.json", LogicalID: "required", Path: "/api/required.json", ExpectedRepresentations: []model.Representation{model.RepresentationJSON}},
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
	wires := string(files["wire_shapes.rs"])
	if !strings.Contains(wires, "http_status_evidence: \"not-documented\"") || !strings.Contains(wires, "description: \"source description\"") {
		t.Fatal("response evidence or selected description was not emitted")
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
			OperationID: "array.json", LogicalID: "array", RustConstant: "ARRAY_JSON", Path: "/api/array.json", ExpectedRepresentations: []model.Representation{model.RepresentationJSON},
			Responses: []model.Response{{Selector: "default", HTTPStatusEvidence: "not-documented", Media: []model.ResponseMedia{{Name: "application/json", ContentTypeStatus: "inferred-from-documented-output-format", Shape: model.ResponseShape{Kind: "opaque", Description: "back\bseparator\u2028", AdditionalPropertiesPolicy: "unspecified"}}}}},
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
	wires := string(files["wire_shapes.rs"])
	if !strings.Contains(wires, `description: "back\u{8}separator\u{2028}"`) {
		t.Fatalf("Rust string literal was not escaped safely:\n%s", wires)
	}
}

func TestFlattenShapeUsesUnambiguousJSONPointerPaths(t *testing.T) {
	shape := model.ResponseShape{Kind: "object", Properties: []model.ResponseProperty{
		{Name: "a.b", Shape: model.ResponseShape{Kind: "string"}},
		{Name: "a", Shape: model.ResponseShape{Kind: "object", Properties: []model.ResponseProperty{{Name: "b", Shape: model.ResponseShape{Kind: "string"}}}}},
		{Name: "slash/tilde~", Shape: model.ResponseShape{Kind: "string"}},
	}}
	var nodes []flatShape
	flattenShape("$", true, shape, &nodes)
	paths := make([]string, 0, len(nodes))
	for _, node := range nodes {
		paths = append(paths, node.path)
	}
	want := []string{"$", "$/a.b", "$/a", "$/a/b", "$/slash~1tilde~0"}
	if !slices.Equal(paths, want) {
		t.Fatalf("paths = %#v, want %#v", paths, want)
	}
}

func int64Pointer(value int64) *int64 { return &value }
