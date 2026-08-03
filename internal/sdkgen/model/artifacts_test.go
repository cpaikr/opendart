package model_test

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"reflect"
	"runtime"
	"testing"

	"github.com/cpaikr/opendart/internal/rustinterface"
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
	batches := canonicalInterfaceBatches(t)
	first, err := model.BuildArtifacts(surface, batches)
	if err != nil {
		t.Fatal(err)
	}
	second, err := model.BuildArtifacts(surface, batches)
	if err != nil {
		t.Fatal(err)
	}
	if first.Semantic.Checksum == "" || first.SDK.Checksum == "" || first.CLIInterface.Checksum == "" || first.CLIDispatch.Checksum == "" {
		t.Fatalf("missing projection identity: %#v", first)
	}
	if first.Semantic.Checksum != second.Semantic.Checksum || first.SDK.Checksum != second.SDK.Checksum || first.CLIInterface.Checksum != second.CLIInterface.Checksum || first.CLIDispatch.Checksum != second.CLIDispatch.Checksum {
		t.Fatal("artifact identities are not deterministic")
	}
	if first.Semantic.Checksum == first.SDK.Checksum || first.Semantic.Checksum == first.CLIInterface.Checksum || first.Semantic.Checksum == first.CLIDispatch.Checksum || first.SDK.Checksum == first.CLIInterface.Checksum || first.SDK.Checksum == first.CLIDispatch.Checksum || first.CLIInterface.Checksum == first.CLIDispatch.Checksum {
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
	changed, err := model.BuildArtifacts(surface, batches)
	if err != nil {
		t.Fatal(err)
	}
	if changed.SDK.Checksum != first.SDK.Checksum {
		t.Fatal("CLI-only prose changed the SDK projection")
	}
	if changed.CLIInterface.Checksum == first.CLIInterface.Checksum || changed.Semantic.Checksum == first.Semantic.Checksum {
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
	if reflect.DeepEqual(firstFiles.CLIInterface, changedFiles.CLIInterface) {
		t.Fatal("CLI-only prose did not rewrite generated CLI bytes")
	}
}

func TestBuildArtifactsProjectsCanonicalDiscoveryFacts(t *testing.T) {
	artifacts, err := model.BuildArtifacts(canonicalSurface(t), canonicalInterfaceBatches(t))
	if err != nil {
		t.Fatal(err)
	}
	var company model.CLIOperation
	for _, operation := range artifacts.CLIInterface.Operations {
		if operation.LogicalID == "DS001-2019002" {
			company = operation
			break
		}
	}
	if company.Name != "company-overview" || company.Group != "DS001" || company.Description == "" {
		t.Fatalf("company discovery = %#v", company)
	}
	if len(company.Parameters) != 1 || company.Parameters[0].Flag != "company-code" || company.Parameters[0].SourceName != "corp_code" || company.Parameters[0].Description == "" || !company.Parameters[0].Required {
		t.Fatalf("company parameters = %#v", company.Parameters)
	}
	if len(company.Representations) != 2 || !company.Representations[0].Selector || company.Representations[0].ResponseShape.Kind != "structured_source" {
		t.Fatalf("company representations = %#v", company.Representations)
	}
	var companyDispatch model.CLIDispatchOperation
	for _, operation := range artifacts.CLIDispatch.Operations {
		if operation.LogicalID == "DS001-2019002" {
			companyDispatch = operation
			break
		}
	}
	if !reflect.DeepEqual(companyDispatch.Representations[0].TestArgv, []string{"--company-code", "00126380", "--representation", "json"}) {
		t.Fatalf("company JSON test argv = %#v", companyDispatch.Representations[0].TestArgv)
	}
	for _, operation := range artifacts.CLIDispatch.Operations {
		for _, representation := range operation.Representations {
			if len(representation.TestArgv) == 0 {
				t.Fatalf("%s/%s has no generated test invocation", operation.Name, representation.Name)
			}
			if representation.Name == model.RepresentationZIP && representation.ResponseType != "opendart::BinaryReply<opendart::BodyStream>" {
				t.Fatalf("ZIP response type = %q", representation.ResponseType)
			}
		}
	}
}

func TestBuildArtifactsRejectsCLIOnlyCollisionsAndDivergence(t *testing.T) {
	t.Run("missing parameter description", func(t *testing.T) {
		surface := canonicalSurface(t)
		logicalID := firstOperationWithParameters(t, &surface).LogicalOperationID
		for index := range surface.Operations {
			if surface.Operations[index].LogicalOperationID == logicalID {
				surface.Operations[index].Parameters[0].Description = " \n "
			}
		}
		_, err := model.BuildArtifacts(surface, canonicalInterfaceBatches(t))
		assertArtifactRule(t, err, "missing-cli-parameter-description")
	})

	t.Run("reserved flag", func(t *testing.T) {
		surface := canonicalSurface(t)
		logicalID := firstOperationWithParameters(t, &surface).LogicalOperationID
		batches := canonicalInterfaceBatches(t)
		for batchIndex := range batches {
			for operationIndex := range batches[batchIndex].Operations {
				operation := &batches[batchIndex].Operations[operationIndex]
				if operation.LogicalID == logicalID {
					operation.Parameters[0].CLIFlag = "output"
				}
			}
		}
		_, err := model.BuildArtifacts(surface, batches)
		assertArtifactRule(t, err, "reserved-cli-flag")
	})

	t.Run("variant description", func(t *testing.T) {
		surface := canonicalSurface(t)
		logicalID := surface.Operations[0].LogicalOperationID
		for index := range surface.Operations {
			if surface.Operations[index].LogicalOperationID == logicalID && surface.Operations[index].OperationID != surface.Operations[0].OperationID {
				surface.Operations[index].Description = "divergent"
				_, err := model.BuildArtifacts(surface, canonicalInterfaceBatches(t))
				assertArtifactRule(t, err, "incompatible-cli-description")
				return
			}
		}
		t.Fatal("canonical surface has no paired logical operation")
	})

	t.Run("missing operation mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		batches[0].Operations = batches[0].Operations[1:]
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "missing-cli-interface-operation")
	})

	t.Run("duplicate operation mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		batches[1].Operations = append(batches[1].Operations, batches[0].Operations[0])
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "duplicate-cli-interface-operation")
	})

	t.Run("orphan operation mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		orphan := batches[0].Operations[0]
		orphan.LogicalID = "DS001-orphan"
		orphan.CLIAlias = orphan.LogicalID
		batches[0].Operations = append(batches[0].Operations, orphan)
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "orphan-cli-interface-operation")
	})

	t.Run("stale alias", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		batches[0].Operations[0].CLIAlias = "legacy"
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "stale-cli-alias")
	})

	t.Run("missing parameter mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		batches[0].Operations[0].Parameters = batches[0].Operations[0].Parameters[1:]
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "missing-cli-parameter-mapping")
	})

	t.Run("orphan parameter mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		parameter := batches[0].Operations[0].Parameters[0]
		parameter.OpenAPIName = "orphan"
		batches[0].Operations[0].Parameters = append(batches[0].Operations[0].Parameters, parameter)
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "orphan-cli-parameter-mapping")
	})

	t.Run("duplicate parameter mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		batches[0].Operations[0].Parameters = append(batches[0].Operations[0].Parameters, batches[0].Operations[0].Parameters[0])
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "duplicate-cli-parameter-mapping")
	})

	t.Run("missing physical mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		batches[0].Operations[0].Physical = batches[0].Operations[0].Physical[1:]
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "missing-cli-physical-mapping")
	})

	t.Run("orphan physical mapping", func(t *testing.T) {
		batches := canonicalInterfaceBatches(t)
		physical := batches[0].Operations[0].Physical[0]
		physical.OperationID = "orphan"
		batches[0].Operations[0].Physical = append(batches[0].Operations[0].Physical, physical)
		_, err := model.BuildArtifacts(canonicalSurface(t), batches)
		assertArtifactRule(t, err, "orphan-cli-physical-mapping")
	})
}

func canonicalInterfaceBatches(t *testing.T) []rustinterface.Batch {
	t.Helper()
	_, current, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("cannot locate interface manifest fixtures")
	}
	batches, err := rustinterface.ReadAll(filepath.Join(filepath.Dir(current), "..", "..", "..", "sdk", "rust", "interface"))
	if err != nil {
		t.Fatal(err)
	}
	return batches
}

func assertArtifactRule(t *testing.T, err error, rule string) {
	t.Helper()
	var modelError *model.Error
	if !errors.As(err, &modelError) || modelError.Rule != rule {
		t.Fatalf("error = %#v, want rule %q", err, rule)
	}
}
