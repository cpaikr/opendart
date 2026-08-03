package model

import (
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/rustinterface"
)

func TestCanonicalCLIProjectionIsCompleteAndPublicIdentityIsPinned(t *testing.T) {
	surface, batches := canonicalCLIInputs(t)
	artifacts, err := BuildArtifacts(surface, batches)
	if err != nil {
		t.Fatal(err)
	}
	if len(artifacts.CLIInterface.Operations) != 85 || len(artifacts.CLIDispatch.Operations) != 85 {
		t.Fatalf("logical inventories = %d and %d", len(artifacts.CLIInterface.Operations), len(artifacts.CLIDispatch.Operations))
	}
	representations := map[Representation]int{}
	physical := make(map[string]bool)
	for _, operation := range artifacts.CLIDispatch.Operations {
		for _, representation := range operation.Representations {
			key := representation.PhysicalID + "\x00" + string(representation.Name)
			if physical[key] {
				t.Fatalf("duplicate physical dispatch %q", key)
			}
			physical[key] = true
			representations[representation.Name]++
		}
	}
	if len(physical) != 167 || representations[RepresentationJSON] != 82 || representations[RepresentationXML] != 82 || representations[RepresentationZIP] != 3 {
		t.Fatalf("physical inventory = %d, representations = %#v", len(physical), representations)
	}
	expectedPhysical := make(map[string]bool, len(surface.Operations))
	for _, source := range surface.Operations {
		representation, err := primaryRepresentation(source)
		if err != nil {
			t.Fatal(err)
		}
		expectedPhysical[source.OperationID+"\x00"+string(representation)] = true
	}
	if len(expectedPhysical) != 167 || !reflect.DeepEqual(physical, expectedPhysical) {
		t.Fatal("dispatch physical inventory differs from the independently inspected OpenAPI inventory")
	}
	const checksum = "878bf8599dc668834a8126074ee32759dd38e8a2a901e7ae07f59e4685b00767"
	if artifacts.CLIInterface.Checksum != checksum {
		t.Fatalf("public interface checksum = %q, want %q", artifacts.CLIInterface.Checksum, checksum)
	}
}

func TestCLIInterfaceProjectionIgnoresEveryPrivateRustSymbol(t *testing.T) {
	surface, batches := canonicalCLIInputs(t)
	before, err := BuildArtifacts(surface, batches)
	if err != nil {
		t.Fatal(err)
	}

	mutations := []struct {
		name   string
		mutate func(*rustinterface.Operation)
	}{
		{name: "module", mutate: func(operation *rustinterface.Operation) { operation.RustModule = "private_module_sentinel" }},
		{name: "input", mutate: func(operation *rustinterface.Operation) { operation.RustInput = "PrivateInputSentinel" }},
		{name: "parameter", mutate: func(operation *rustinterface.Operation) { operation.Parameters[0].RustName = "private_field_sentinel" }},
		{name: "method", mutate: func(operation *rustinterface.Operation) {
			operation.Physical[0].RustMethod = "prepare_private_sentinel"
		}},
		{name: "response", mutate: func(operation *rustinterface.Operation) {
			operation.Physical[0].RustResponse = "PrivateResponseSentinel"
		}},
	}
	for _, test := range mutations {
		t.Run(test.name, func(t *testing.T) {
			changedBatches := cloneBatches(batches)
			operation := &changedBatches[0].Operations[0]
			if len(operation.Parameters) == 0 || len(operation.Physical) == 0 {
				t.Fatal("canonical first operation lacks private bindings")
			}
			test.mutate(operation)
			after, err := BuildArtifacts(surface, changedBatches)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(before.CLIInterface, after.CLIInterface) {
				t.Fatal("private Rust symbol rename changed the public CLI interface")
			}
			if reflect.DeepEqual(before.CLIDispatch, after.CLIDispatch) || before.CLIDispatch.Checksum == after.CLIDispatch.Checksum {
				t.Fatal("private Rust symbol rename did not change private dispatch")
			}
		})
	}
}

func TestCLIProjectionRejectsMixedStructuredAndBinaryVariants(t *testing.T) {
	surface := openapispec.SDKSurface{Operations: []openapispec.SDKSurfaceOperation{
		cliSourceOperation("mixed.json", "mixed", "application/json"),
		cliSourceOperation("mixed.xml", "mixed", "application/zip"),
	}}
	batches := []rustinterface.Batch{{
		SchemaVersion: rustinterface.SchemaVersion, Family: "GROUP",
		Operations: []rustinterface.Operation{{
			LogicalID: "mixed", RustModule: "group", RustInput: "MixedInput", CLICommand: "mixed", CLIAlias: "mixed",
			Physical: []rustinterface.Physical{
				{OperationID: "mixed.json", RustMethod: "prepare_json", RustResponse: "MixedJsonResponse"},
				{OperationID: "mixed.xml", RustMethod: "prepare_archive", RustResponse: "MixedArchiveResponse"},
			},
		}},
	}}
	_, err := BuildArtifacts(surface, batches)
	if modelError, ok := err.(*Error); !ok || modelError.Rule != "mixed-cli-representation-kinds" {
		t.Fatalf("error = %#v", err)
	}
}

func cliSourceOperation(physical, logical, media string) openapispec.SDKSurfaceOperation {
	return openapispec.SDKSurfaceOperation{
		OperationID: physical, LogicalOperationID: logical, APIGroupCode: "GROUP", APIID: "api", GuideURL: "https://opendart.fss.or.kr/guide",
		Description: "A mixed operation.",
		Responses:   []openapispec.SDKSurfaceResponse{{MediaTypes: []openapispec.SDKSurfaceMediaType{{Name: media, ContentTypeStatus: "inferred-from-documented-output-format"}}}},
	}
}

func cloneBatches(source []rustinterface.Batch) []rustinterface.Batch {
	result := make([]rustinterface.Batch, len(source))
	for batchIndex, batch := range source {
		result[batchIndex] = batch
		result[batchIndex].Operations = append([]rustinterface.Operation(nil), batch.Operations...)
		for operationIndex, operation := range batch.Operations {
			result[batchIndex].Operations[operationIndex].Parameters = append([]rustinterface.Parameter(nil), operation.Parameters...)
			result[batchIndex].Operations[operationIndex].Physical = append([]rustinterface.Physical(nil), operation.Physical...)
		}
	}
	return result
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
