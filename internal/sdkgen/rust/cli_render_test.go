package rust

import (
	"strings"
	"testing"

	"github.com/cpaikr/opendart/internal/sdkgen/model"
)

func TestRenderCLIDeduplicatesIdenticalNameAndLogicalID(t *testing.T) {
	source := model.CLIModel{
		SchemaVersion: model.CLIProjectionSchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Operations: []model.CLIOperation{{
			Name:         "same",
			LogicalID:    "same",
			SDKInputType: "Same",
			Representations: []model.CLIRepresentation{{
				Name:          model.RepresentationJSON,
				PrepareMethod: "prepare_json",
				ResponseType:  "opendart::responses::SameJsonResponse",
				ResponseShape: model.CLIResponseShape{Kind: "source_value"},
			}},
		}},
	}

	files, err := renderCLI(source)
	if err != nil {
		t.Fatal(err)
	}
	command := string(files["command.rs"])
	if !strings.Contains(command, "if operation.logical_id != operation.name") {
		t.Fatal("command renderer does not guard identical clap aliases")
	}
	for _, filter := range []string{`Arg::new("query")`, `Arg::new("group")`, `Arg::new("representation")`} {
		if !strings.Contains(command, filter) {
			t.Fatalf("command renderer omits operations-list filter %s:\n%s", filter, command)
		}
	}
	if !strings.Contains(command, `.value_parser(["json", "xml", "zip"])`) {
		t.Fatalf("command renderer omits the closed discovery representation set:\n%s", command)
	}
	for _, fragment := range []string{
		`Command::new("list")
        .display_name("opendart")`,
		`Command::new("operations")
        .display_name("opendart")`,
		`Command::new("describe")
                .display_name("opendart")`,
		`Command::new("call")
        .display_name("opendart")`,
		`.override_usage("opendart call <OPERATION> [OPTIONS]")`,
		`Command::new(operation.name)
            .display_name("opendart")
            .hide(true)`,
		`Command::new("opendart")
        .display_name("opendart")`,
		`Discover operation names with 'opendart operations list'`,
		`.value_name("REPRESENTATION")`,
		`.value_name("PATH")`,
	} {
		if !strings.Contains(command, fragment) {
			t.Fatalf("command renderer omits help/version contract %q:\n%s", fragment, command)
		}
	}
	dispatch := string(files["dispatch.rs"])
	if strings.Contains(dispatch, `"same" | "same"`) {
		t.Fatalf("dispatch repeats an identical match pattern:\n%s", dispatch)
	}
	if !strings.Contains(dispatch, `Some(("same", _matches))`) {
		t.Fatalf("dispatch omits the deduplicated match pattern:\n%s", dispatch)
	}
}

func TestRenderCLICatalogIncludesCanonicalStringConstraints(t *testing.T) {
	minimum, maximum := int64(1), int64(100)
	source := model.CLIModel{
		SchemaVersion: model.CLIProjectionSchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Operations: []model.CLIOperation{{
			Name: "list", LogicalID: "list", SDKInputType: "List",
			Parameters: []model.CLIParameter{{
				Flag: "page-count", WireName: "page_count", SDKField: "page_count", Shape: model.ScalarString,
				Constraints: model.StringConstraints{AllowedValues: []string{"10", "100"}, DecimalMinimum: &minimum, DecimalMaximum: &maximum},
			}},
			Representations: []model.CLIRepresentation{{Name: model.RepresentationJSON, ResponseShape: model.CLIResponseShape{Kind: "source_value"}}},
		}},
	}
	files, err := renderCLI(source)
	if err != nil {
		t.Fatal(err)
	}
	catalog := string(files["catalog.rs"])
	for _, fragment := range []string{`constraints: Some(StringConstraintSpec`, `allowed_values: &["10", "100"]`, `decimal_minimum: Some(1)`, `decimal_maximum: Some(100)`} {
		if !strings.Contains(catalog, fragment) {
			t.Fatalf("catalog omits constraint %q:\n%s", fragment, catalog)
		}
	}
}
