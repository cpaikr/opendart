package rust

import (
	"strings"
	"testing"

	"github.com/cpaikr/opendart/internal/sdkgen/model"
)

func TestRenderCLISeparatesPublicInterfaceAndPrivateDispatch(t *testing.T) {
	interfaceSource := model.CLIInterfaceModel{
		SchemaVersion: model.CLIInterfaceProjectionSchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Operations: []model.CLIOperation{{
			Name: "same", LogicalID: "same", Group: "DS001",
			Representations: []model.CLIRepresentation{{
				Name: model.RepresentationJSON, PhysicalID: "same.json",
				ResponseShape: model.CLIResponseShape{Kind: "structured_source"},
			}},
		}},
	}
	dispatchSource := model.CLIDispatchModel{
		SchemaVersion: model.CLIDispatchProjectionSchemaVersion,
		Checksum:      strings.Repeat("b", 64),
		Operations: []model.CLIDispatchOperation{{
			Name: "same", LogicalID: "same", SDKInputType: "PrivateInput",
			Representations: []model.CLIDispatchRepresentation{{
				Name: model.RepresentationJSON, PhysicalID: "same.json",
				PrepareMethod: "prepare_json", ResponseType: "opendart::responses::PrivateResponse",
			}},
		}},
	}

	interfaceFiles, err := renderCLIInterface(interfaceSource)
	if err != nil {
		t.Fatal(err)
	}
	dispatchFiles, err := renderCLIDispatch(dispatchSource)
	if err != nil {
		t.Fatal(err)
	}
	command := string(interfaceFiles["command.rs"])
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
	for name, body := range interfaceFiles {
		text := string(body)
		for _, private := range []string{"PrivateInput", "PrivateResponse", "prepare_json", "sdk_field", "response_type"} {
			if strings.Contains(text, private) {
				t.Fatalf("public interface file %s contains private symbol %q", name, private)
			}
		}
	}
	dispatch := string(dispatchFiles["adapter.rs"])
	if strings.Contains(dispatch, `"same" | "same"`) {
		t.Fatalf("dispatch repeats an identical match pattern:\n%s", dispatch)
	}
	if !strings.Contains(dispatch, `Some(("same", _matches))`) || !strings.Contains(dispatch, "PrivateInput") {
		t.Fatalf("dispatch omits private adapter wiring:\n%s", dispatch)
	}
	dispatchCases := string(dispatchFiles["dispatch_cases.json"])
	if !strings.Contains(dispatchCases, `"physical_id":"same.json"`) || !strings.Contains(dispatchCases, `"argv":["call","same"]`) {
		t.Fatalf("dispatch fixture omits the generated identity case:\n%s", dispatchCases)
	}
}

func TestRenderCLICatalogIncludesCanonicalStringConstraints(t *testing.T) {
	minimum, maximum := int64(1), int64(100)
	source := model.CLIInterfaceModel{
		SchemaVersion: model.CLIInterfaceProjectionSchemaVersion,
		Checksum:      strings.Repeat("a", 64),
		Operations: []model.CLIOperation{{
			Name: "list", LogicalID: "list",
			Parameters: []model.CLIParameter{{
				Flag: "page-size", SourceName: "page_count", Shape: model.ScalarString,
				Constraints: model.StringConstraints{AllowedValues: []string{"10", "100"}, DecimalMinimum: &minimum, DecimalMaximum: &maximum},
			}},
			Representations: []model.CLIRepresentation{{Name: model.RepresentationJSON, PhysicalID: "list.json", ResponseShape: model.CLIResponseShape{Kind: "structured_source"}}},
		}},
	}
	files, err := renderCLIInterface(source)
	if err != nil {
		t.Fatal(err)
	}
	catalog := string(files["catalog.rs"])
	for _, fragment := range []string{`source_name: "page_count"`, `constraints: Some(StringConstraintSpec`, `allowed_values: &["10", "100"]`, `decimal_minimum: Some(1)`, `decimal_maximum: Some(100)`} {
		if !strings.Contains(catalog, fragment) {
			t.Fatalf("catalog omits constraint %q:\n%s", fragment, catalog)
		}
	}
}
