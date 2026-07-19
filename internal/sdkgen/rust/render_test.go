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
