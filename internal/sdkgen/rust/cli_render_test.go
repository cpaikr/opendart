package rust

import (
	"encoding/json"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/rustinterface"
	"github.com/cpaikr/opendart/internal/sdkgen/model"
)

func TestRenderCanonicalDispatchHasExactIndependentCoverage(t *testing.T) {
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate canonical inputs")
	}
	repository := filepath.Join(filepath.Dir(current), "..", "..", "..")
	document, err := openapispec.Load(filepath.Join(repository, "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	defer document.Close()
	surface, err := document.InspectSDKSurface()
	if err != nil {
		t.Fatal(err)
	}
	batches, err := rustinterface.ReadAll(filepath.Join(repository, "sdk", "rust", "interface"))
	if err != nil {
		t.Fatal(err)
	}
	artifacts, err := model.BuildArtifacts(surface, batches)
	if err != nil {
		t.Fatal(err)
	}
	files, err := RenderArtifacts(artifacts)
	if err != nil {
		t.Fatal(err)
	}
	var cases []struct {
		LogicalID      string `json:"logical_id"`
		PhysicalID     string `json:"physical_id"`
		Representation string `json:"representation"`
	}
	if err := json.Unmarshal(files.CLIDispatch["dispatch_cases.json"], &cases); err != nil {
		t.Fatal(err)
	}
	logicalIDs := make(map[string]bool)
	actualPhysical := make(map[string]bool)
	for _, dispatchCase := range cases {
		logicalIDs[dispatchCase.LogicalID] = true
		actualPhysical[dispatchCase.PhysicalID+"\x00"+dispatchCase.Representation] = true
	}
	if len(logicalIDs) != 85 || len(cases) != 167 || len(actualPhysical) != 167 {
		t.Fatalf("rendered dispatch coverage = %d logical/%d cases/%d unique physical", len(logicalIDs), len(cases), len(actualPhysical))
	}
	expectedPhysical := make(map[string]bool, len(surface.Operations))
	for _, operation := range surface.Operations {
		representation := independentlyInspectRepresentation(t, operation)
		expectedPhysical[operation.OperationID+"\x00"+representation] = true
	}
	if len(expectedPhysical) != 167 {
		t.Fatalf("OpenAPI physical inventory = %d, want 167", len(expectedPhysical))
	}
	for identity := range expectedPhysical {
		if !actualPhysical[identity] {
			t.Fatalf("rendered dispatch omitted OpenAPI identity %q", identity)
		}
	}
}

func independentlyInspectRepresentation(t *testing.T, operation openapispec.SDKSurfaceOperation) string {
	t.Helper()
	representations := make(map[string]bool)
	for _, response := range operation.Responses {
		for _, media := range response.MediaTypes {
			if media.ContentTypeStatus != "inferred-from-documented-output-format" {
				continue
			}
			switch media.Name {
			case "application/json":
				representations["json"] = true
			case "application/xml":
				representations["xml"] = true
			case "application/zip":
				representations["zip"] = true
			default:
				t.Fatalf("%s has unsupported primary media %q", operation.OperationID, media.Name)
			}
		}
	}
	if len(representations) != 1 {
		t.Fatalf("%s has %d primary representations", operation.OperationID, len(representations))
	}
	for representation := range representations {
		return representation
	}
	panic("unreachable")
}

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
			Name: "same", LogicalID: "same", RustModule: "private_module", RustInput: "PrivateInput",
			Representations: []model.CLIDispatchRepresentation{{
				Name: model.RepresentationJSON, PhysicalID: "same.json",
				PrepareMethod: "prepare_json", ResponseType: "opendart::operations::private_module::PrivateResponse",
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

func TestBinaryDispatchDoesNotRequireAStructuredResponseType(t *testing.T) {
	expression := preparedExpression(model.CLIDispatchRepresentation{
		Name: model.RepresentationZIP, PrepareMethod: "prepare_archive",
		ResponseType: "opendart::operations::private_module::AbsentArchiveResponse",
	})
	if strings.Contains(expression, "AbsentArchiveResponse") || !strings.Contains(expression, "input.prepare_archive()?") {
		t.Fatalf("binary dispatch expression = %q", expression)
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
