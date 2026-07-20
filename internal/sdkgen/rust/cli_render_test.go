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
	dispatch := string(files["dispatch.rs"])
	if strings.Contains(dispatch, `"same" | "same"`) {
		t.Fatalf("dispatch repeats an identical match pattern:\n%s", dispatch)
	}
	if !strings.Contains(dispatch, `Some(("same", _matches))`) {
		t.Fatalf("dispatch omits the deduplicated match pattern:\n%s", dispatch)
	}
}
