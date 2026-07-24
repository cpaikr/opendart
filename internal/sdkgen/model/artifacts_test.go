package model_test

import (
	"encoding/json"
	"errors"
	"reflect"
	"testing"

	openapispec "github.com/cpaikr/opendart/internal/openapi"
	"github.com/cpaikr/opendart/internal/sdkgen/model"
	rustemitter "github.com/cpaikr/opendart/internal/sdkgen/rust"
)

func TestConstraintProjectionsOmitOnlyZeroValues(t *testing.T) {
	for _, test := range []struct {
		name  string
		zero  any
		value any
	}{
		{
			name:  "SDK",
			zero:  model.Parameter{},
			value: model.Parameter{Constraints: model.StringConstraints{Format: "opendart-date"}},
		},
		{
			name:  "CLI",
			zero:  model.CLIParameter{},
			value: model.CLIParameter{Constraints: model.StringConstraints{Format: "opendart-date"}},
		},
	} {
		t.Run(test.name, func(t *testing.T) {
			zero, err := json.Marshal(test.zero)
			if err != nil {
				t.Fatal(err)
			}
			var zeroFields map[string]any
			if err := json.Unmarshal(zero, &zeroFields); err != nil {
				t.Fatal(err)
			}
			if _, exists := zeroFields["constraints"]; exists {
				t.Fatalf("zero constraints were serialized: %s", zero)
			}

			value, err := json.Marshal(test.value)
			if err != nil {
				t.Fatal(err)
			}
			var valueFields map[string]any
			if err := json.Unmarshal(value, &valueFields); err != nil {
				t.Fatal(err)
			}
			if _, exists := valueFields["constraints"]; !exists {
				t.Fatalf("nonzero constraints were omitted: %s", value)
			}
		})
	}
}

func TestBuildArtifactsSeparatesSemanticSDKAndCLIIdentities(t *testing.T) {
	surface := canonicalSurface(t)
	first, err := model.BuildArtifacts(surface)
	if err != nil {
		t.Fatal(err)
	}
	second, err := model.BuildArtifacts(surface)
	if err != nil {
		t.Fatal(err)
	}
	if first.Semantic.Checksum == "" || first.SDK.Checksum == "" || first.CLI.Checksum == "" {
		t.Fatalf("missing projection identity: %#v", first)
	}
	if first.Semantic.Checksum != second.Semantic.Checksum || first.SDK.Checksum != second.SDK.Checksum || first.CLI.Checksum != second.CLI.Checksum {
		t.Fatal("artifact identities are not deterministic")
	}
	if first.Semantic.Checksum == first.SDK.Checksum || first.Semantic.Checksum == first.CLI.Checksum || first.SDK.Checksum == first.CLI.Checksum {
		t.Fatal("distinct projection schemas unexpectedly share an identity")
	}

	logicalID := surface.Operations[0].LogicalOperationID
	for index := range surface.Operations {
		if surface.Operations[index].LogicalOperationID == logicalID {
			surface.Operations[index].Description = "CLI-only presentation change"
			for parameterIndex := range surface.Operations[index].Parameters {
				surface.Operations[index].Parameters[parameterIndex].Description = "CLI-only parameter presentation change"
			}
		}
	}
	changed, err := model.BuildArtifacts(surface)
	if err != nil {
		t.Fatal(err)
	}
	if changed.SDK.Checksum != first.SDK.Checksum {
		t.Fatal("CLI-only prose changed the SDK projection")
	}
	if changed.CLI.Checksum == first.CLI.Checksum || changed.Semantic.Checksum == first.Semantic.Checksum {
		t.Fatal("CLI-only prose did not change its owning projections")
	}
	firstFiles, err := rustemitter.RenderArtifacts(first)
	if err != nil {
		t.Fatal(err)
	}
	changedFiles, err := rustemitter.RenderArtifacts(changed)
	if err != nil {
		t.Fatal(err)
	}
	if !reflect.DeepEqual(firstFiles.SDK, changedFiles.SDK) {
		t.Fatal("CLI-only prose rewrote generated SDK bytes")
	}
	if reflect.DeepEqual(firstFiles.CLI, changedFiles.CLI) {
		t.Fatal("CLI-only prose did not rewrite generated CLI bytes")
	}
}

func TestBuildArtifactsProjectsCanonicalDiscoveryFacts(t *testing.T) {
	artifacts, err := model.BuildArtifacts(canonicalSurface(t))
	if err != nil {
		t.Fatal(err)
	}
	var company model.CLIOperation
	for _, operation := range artifacts.CLI.Operations {
		if operation.LogicalID == "DS001-2019002" {
			company = operation
			break
		}
	}
	if company.Name != "company" || company.Group != "DS001" || company.Description == "" || company.SDKInputType != "Company" {
		t.Fatalf("company discovery = %#v", company)
	}
	if len(company.Parameters) != 1 || company.Parameters[0].Flag != "corp-code" || company.Parameters[0].Description == "" || !company.Parameters[0].Required {
		t.Fatalf("company parameters = %#v", company.Parameters)
	}
	if len(company.Representations) != 2 || !company.Representations[0].Selector || company.Representations[0].ResponseShape.Kind != "object" {
		t.Fatalf("company representations = %#v", company.Representations)
	}
	for _, operation := range artifacts.CLI.Operations {
		for _, representation := range operation.Representations {
			if representation.Name == model.RepresentationZIP && representation.ResponseType != "opendart::BinaryReply<opendart::BodyStream>" {
				t.Fatalf("ZIP response type = %q", representation.ResponseType)
			}
		}
	}
}

func TestBuildArtifactsRejectsCLIOnlyCollisionsAndDivergence(t *testing.T) {
	t.Run("response shape context", func(t *testing.T) {
		surface := canonicalSurface(t)
		for operationIndex := range surface.Operations {
			operation := &surface.Operations[operationIndex]
			for responseIndex := range operation.Responses {
				response := &operation.Responses[responseIndex]
				for mediaIndex := range response.MediaTypes {
					media := &response.MediaTypes[mediaIndex]
					if media.ContentTypeStatus != "inferred-from-documented-output-format" || media.Name == "application/zip" {
						continue
					}
					media.Schema = openapispec.SDKSurfaceSchema{Types: []string{"string"}}
					_, err := model.BuildArtifacts(surface)
					var modelError *model.Error
					if !errors.As(err, &modelError) || modelError.Rule != "unsupported-cli-response-shape" || modelError.Operation != operation.OperationID || modelError.Location == "" {
						t.Fatalf("error = %#v, want structured response-shape context", err)
					}
					return
				}
			}
		}
		t.Fatal("canonical surface has no structured primary response")
	})

	t.Run("missing parameter description", func(t *testing.T) {
		surface := canonicalSurface(t)
		logicalID := firstOperationWithParameters(t, &surface).LogicalOperationID
		for index := range surface.Operations {
			if surface.Operations[index].LogicalOperationID == logicalID {
				surface.Operations[index].Parameters[0].Description = " \n "
			}
		}
		_, err := model.BuildArtifacts(surface)
		assertArtifactRule(t, err, "missing-cli-parameter-description")
	})

	t.Run("reserved flag", func(t *testing.T) {
		surface := canonicalSurface(t)
		logicalID := firstOperationWithParameters(t, &surface).LogicalOperationID
		for index := range surface.Operations {
			if surface.Operations[index].LogicalOperationID == logicalID {
				surface.Operations[index].Parameters[0].Name = "output"
			}
		}
		_, err := model.BuildArtifacts(surface)
		assertArtifactRule(t, err, "reserved-cli-flag")
	})

	t.Run("variant description", func(t *testing.T) {
		surface := canonicalSurface(t)
		logicalID := surface.Operations[0].LogicalOperationID
		for index := range surface.Operations {
			if surface.Operations[index].LogicalOperationID == logicalID && surface.Operations[index].OperationID != surface.Operations[0].OperationID {
				surface.Operations[index].Description = "divergent"
				_, err := model.BuildArtifacts(surface)
				assertArtifactRule(t, err, "incompatible-cli-description")
				return
			}
		}
		t.Fatal("canonical surface has no paired logical operation")
	})
}

func assertArtifactRule(t *testing.T, err error, rule string) {
	t.Helper()
	var modelError *model.Error
	if !errors.As(err, &modelError) || modelError.Rule != rule {
		t.Fatalf("error = %#v, want rule %q", err, rule)
	}
}
