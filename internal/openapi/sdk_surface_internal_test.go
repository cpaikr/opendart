package openapi

import (
	"testing"

	"github.com/pb33f/libopenapi/datamodel/high/v3"
)

func TestEffectiveSDKParametersAppliesOperationPrecedence(t *testing.T) {
	inherited := &v3.Parameter{Name: "identifier", In: "query", Style: "form"}
	inheritedOnly := &v3.Parameter{Name: "inherited", In: "header"}
	override := &v3.Parameter{Name: "identifier", In: "query", Style: "spaceDelimited"}
	operationOnly := &v3.Parameter{Name: "operation", In: "query"}

	effective, err := effectiveSDKParameters(
		"GET /things",
		[]*v3.Parameter{inherited, inheritedOnly},
		[]*v3.Parameter{override, operationOnly},
	)
	if err != nil {
		t.Fatal(err)
	}
	if len(effective) != 3 || effective[0] != override || effective[1] != inheritedOnly || effective[2] != operationOnly {
		t.Fatalf("effective parameters = %#v", effective)
	}
}

func TestEffectiveSDKParametersRejectsNilDefinitions(t *testing.T) {
	if _, err := effectiveSDKParameters("GET /things", []*v3.Parameter{nil}, nil); err == nil {
		t.Fatal("nil inherited parameter was accepted")
	}
	if _, err := effectiveSDKParameters("GET /things", nil, []*v3.Parameter{nil}); err == nil {
		t.Fatal("nil operation parameter was accepted")
	}
}

func TestEffectiveSDKParametersRejectsDuplicatesWithinOneLevel(t *testing.T) {
	parameter := &v3.Parameter{Name: "identifier", In: "query"}
	if _, err := effectiveSDKParameters("GET /things", []*v3.Parameter{parameter, parameter}, nil); err == nil {
		t.Fatal("duplicate inherited parameter was accepted")
	}
	if _, err := effectiveSDKParameters("GET /things", nil, []*v3.Parameter{parameter, parameter}); err == nil {
		t.Fatal("duplicate operation parameter was accepted")
	}
}
