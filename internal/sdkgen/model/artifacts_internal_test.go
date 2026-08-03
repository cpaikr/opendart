package model

import (
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/rustinterface"
)

func TestCLIProjectionRejectsMixedStructuredAndBinaryVariants(t *testing.T) {
	sdk := Model{
		SchemaVersion: SchemaVersion,
		Checksum:      "sdk-checksum",
		Logical: []LogicalOperation{{
			ID:       "mixed",
			RustName: "Mixed",
			Group:    "group",
			Variants: []PhysicalReference{
				{OperationID: "mixed.json", Representation: RepresentationJSON},
				{OperationID: "mixed.xml", Representation: RepresentationZIP},
			},
		}},
		Physical: []PhysicalOperation{
			{
				OperationID:           "mixed.json",
				LogicalID:             "mixed",
				PrimaryRepresentation: RepresentationJSON,
				Responses: []Response{{Media: []ResponseMedia{{
					Name:              "application/json",
					ContentTypeStatus: "inferred-from-documented-output-format",
					Shape:             ResponseShape{Kind: "object"},
				}}}},
			},
			{
				OperationID:           "mixed.xml",
				LogicalID:             "mixed",
				PrimaryRepresentation: RepresentationZIP,
			},
		},
	}
	surface := openAPISurfaceForCLIProjection("mixed", "mixed.json", "A mixed operation.")
	surface.Operations = append(surface.Operations, surface.Operations[0])
	surface.Operations[1].OperationID = "mixed.xml"
	batches := []rustinterface.Batch{{
		SchemaVersion: rustinterface.SchemaVersion,
		Family:        "GROUP",
		Operations: []rustinterface.Operation{{
			LogicalID: "mixed", CLICommand: "mixed", CLIAlias: "mixed",
			Physical: []rustinterface.Physical{{OperationID: "mixed.json"}, {OperationID: "mixed.xml"}},
		}},
	}}

	_, _, err := buildCLIProjections(surface, sdk, batches)
	var modelError *Error
	if !errors.As(err, &modelError) || modelError.Rule != "mixed-cli-representation-kinds" {
		t.Fatalf("error = %#v", err)
	}
}

func TestCLIInterfaceProjectionIgnoresPrivateRustSymbolRenames(t *testing.T) {
	surface, batches := canonicalCLIInputs(t)
	sdk, err := Build(surface)
	if err != nil {
		t.Fatal(err)
	}
	beforeInterface, beforeDispatch, err := buildCLIProjections(surface, sdk, batches)
	if err != nil {
		t.Fatal(err)
	}

	sdk.Logical[0].RustName = "PrivateInputSentinel"
	if len(sdk.Logical[0].Parameters) == 0 {
		t.Fatal("canonical first operation has no parameter")
	}
	sdk.Logical[0].Parameters[0].RustName = "private_field_sentinel"
	afterInterface, afterDispatch, err := buildCLIProjections(surface, sdk, batches)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(beforeInterface, afterInterface) {
		t.Fatal("private Rust symbol rename changed the public CLI interface")
	}
	if reflect.DeepEqual(beforeDispatch, afterDispatch) || beforeDispatch.Checksum == afterDispatch.Checksum {
		t.Fatal("private Rust symbol rename did not change private dispatch")
	}
}

func canonicalCLIInputs(t *testing.T) (openapispec.SDKSurface, []rustinterface.Batch) {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate canonical CLI inputs")
	}
	repository := filepath.Join(filepath.Dir(current), "..", "..", "..")
	document, err := openapispec.Load(filepath.Join(repository, "openapi", "openapi.yaml"))
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { document.Close() })
	if err := document.ValidateDocument(); err != nil {
		t.Fatal(err)
	}
	surface, err := document.InspectSDKSurface()
	if err != nil {
		t.Fatal(err)
	}
	batches, err := rustinterface.ReadAll(filepath.Join(repository, "sdk", "rust", "interface"))
	if err != nil {
		t.Fatal(err)
	}
	return surface, batches
}

func openAPISurfaceForCLIProjection(logicalID, operationID, description string) openapispec.SDKSurface {
	return openapispec.SDKSurface{Operations: []openapispec.SDKSurfaceOperation{{
		OperationID:        operationID,
		LogicalOperationID: logicalID,
		Description:        description,
	}}}
}
