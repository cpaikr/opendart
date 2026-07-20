package model

import (
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
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
					Name:  "application/json",
					Shape: ResponseShape{Kind: "object"},
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

	_, err := buildCLIProjection(surface, sdk)
	modelError, ok := err.(*Error)
	if !ok || modelError.Rule != "mixed-cli-representation-kinds" {
		t.Fatalf("error = %#v", err)
	}
}

func openAPISurfaceForCLIProjection(logicalID, operationID, description string) openapispec.SDKSurface {
	return openapispec.SDKSurface{Operations: []openapispec.SDKSurfaceOperation{{
		OperationID:        operationID,
		LogicalOperationID: logicalID,
		Description:        description,
	}}}
}
