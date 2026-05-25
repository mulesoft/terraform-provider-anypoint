package agentstools

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"
)

func TestEndpointFromObject(t *testing.T) {
	t.Run("null object returns nil", func(t *testing.T) {
		obj := types.ObjectNull(endpointAttrTypes)
		result := endpointFromObject(obj)
		if result != nil {
			t.Errorf("endpointFromObject(null) = %v, want nil", result)
		}
	})

	t.Run("unknown object returns nil", func(t *testing.T) {
		obj := types.ObjectUnknown(endpointAttrTypes)
		result := endpointFromObject(obj)
		if result != nil {
			t.Errorf("endpointFromObject(unknown) = %v, want nil", result)
		}
	})

	t.Run("valid object returns populated model", func(t *testing.T) {
		obj, diags := types.ObjectValue(endpointAttrTypes, map[string]attr.Value{
			"deployment_type":  types.StringValue("omniGateway"),
			"type":             types.StringValue("http"),
			"base_path":        types.StringValue("/api"),
			"uri":              types.StringValue("https://example.com"),
			"response_timeout": types.Int64Value(30000),
		})
		if diags.HasError() {
			t.Fatalf("ObjectValue error: %v", diags.Errors())
		}
		result := endpointFromObject(obj)
		if result == nil {
			t.Fatal("endpointFromObject() returned nil for valid object")
		}
		if result.DeploymentType.ValueString() != "omniGateway" {
			t.Errorf("DeploymentType = %s, want omniGateway", result.DeploymentType.ValueString())
		}
		if result.Type.ValueString() != "http" {
			t.Errorf("Type = %s, want http", result.Type.ValueString())
		}
		if result.BasePath.ValueString() != "/api" {
			t.Errorf("BasePath = %s, want /api", result.BasePath.ValueString())
		}
		if result.URI.ValueString() != "https://example.com" {
			t.Errorf("URI = %s, want https://example.com", result.URI.ValueString())
		}
		if result.ResponseTimeout.ValueInt64() != 30000 {
			t.Errorf("ResponseTimeout = %d, want 30000", result.ResponseTimeout.ValueInt64())
		}
	})
}

func TestEndpointToObject(t *testing.T) {
	t.Run("nil endpoint returns null object", func(t *testing.T) {
		result := endpointToObject(nil)
		if !result.IsNull() {
			t.Errorf("endpointToObject(nil) should return null object")
		}
	})

	t.Run("valid endpoint returns populated object", func(t *testing.T) {
		ep := &EndpointModel{
			DeploymentType:  types.StringValue("omniGateway"),
			Type:            types.StringValue("http"),
			BasePath:        types.StringValue("/api"),
			URI:             types.StringValue("https://example.com"),
			ResponseTimeout: types.Int64Value(5000),
		}
		result := endpointToObject(ep)
		if result.IsNull() || result.IsUnknown() {
			t.Fatal("endpointToObject() returned null/unknown for valid endpoint")
		}
		attrs := result.Attributes()
		if attrs["deployment_type"].(types.String).ValueString() != "omniGateway" {
			t.Errorf("deployment_type = %v, want omniGateway", attrs["deployment_type"])
		}
	})
}

func TestDeploymentFromObject(t *testing.T) {
	t.Run("null object returns nil", func(t *testing.T) {
		obj := types.ObjectNull(deploymentAttrTypes)
		result := deploymentFromObject(obj)
		if result != nil {
			t.Errorf("deploymentFromObject(null) = %v, want nil", result)
		}
	})

	t.Run("unknown object returns nil", func(t *testing.T) {
		obj := types.ObjectUnknown(deploymentAttrTypes)
		result := deploymentFromObject(obj)
		if result != nil {
			t.Errorf("deploymentFromObject(unknown) = %v, want nil", result)
		}
	})

	t.Run("valid object returns populated model", func(t *testing.T) {
		obj, diags := types.ObjectValue(deploymentAttrTypes, map[string]attr.Value{
			"environment_id":  types.StringValue("env-123"),
			"type":            types.StringValue("SharedSpace"),
			"expected_status": types.StringValue("Running"),
			"overwrite":       types.BoolValue(true),
			"target_id":       types.StringValue("target-456"),
			"target_name":     types.StringValue("my-target"),
			"gateway_version": types.StringValue("1.0.0"),
		})
		if diags.HasError() {
			t.Fatalf("ObjectValue error: %v", diags.Errors())
		}
		result := deploymentFromObject(obj)
		if result == nil {
			t.Fatal("deploymentFromObject() returned nil for valid object")
		}
		if result.EnvironmentID.ValueString() != "env-123" {
			t.Errorf("EnvironmentID = %s, want env-123", result.EnvironmentID.ValueString())
		}
		if result.Type.ValueString() != "SharedSpace" {
			t.Errorf("Type = %s, want SharedSpace", result.Type.ValueString())
		}
		if !result.Overwrite.ValueBool() {
			t.Errorf("Overwrite = false, want true")
		}
	})
}

func TestDeploymentToObject(t *testing.T) {
	t.Run("nil deployment returns null object", func(t *testing.T) {
		result := deploymentToObject(nil)
		if !result.IsNull() {
			t.Errorf("deploymentToObject(nil) should return null object")
		}
	})

	t.Run("valid deployment returns populated object", func(t *testing.T) {
		dep := &DeploymentModel{
			EnvironmentID:  types.StringValue("env-123"),
			Type:           types.StringValue("SharedSpace"),
			ExpectedStatus: types.StringValue("Running"),
			Overwrite:      types.BoolValue(false),
			TargetID:       types.StringValue("target-456"),
			TargetName:     types.StringValue("my-target"),
			GatewayVersion: types.StringValue("2.0.0"),
		}
		result := deploymentToObject(dep)
		if result.IsNull() || result.IsUnknown() {
			t.Fatal("deploymentToObject() returned null/unknown for valid deployment")
		}
		attrs := result.Attributes()
		if attrs["environment_id"].(types.String).ValueString() != "env-123" {
			t.Errorf("environment_id = %v, want env-123", attrs["environment_id"])
		}
	})
}

func TestMergeDeploymentObjects(t *testing.T) {
	makeDeploymentObj := func(envID, typ, expectedStatus, targetID, targetName, version string, overwrite bool) types.Object {
		dep := &DeploymentModel{
			EnvironmentID:  types.StringValue(envID),
			Type:           types.StringValue(typ),
			ExpectedStatus: types.StringValue(expectedStatus),
			Overwrite:      types.BoolValue(overwrite),
			TargetID:       types.StringValue(targetID),
			TargetName:     types.StringValue(targetName),
			GatewayVersion: types.StringValue(version),
		}
		return deploymentToObject(dep)
	}

	t.Run("null planned returns apiDep unchanged", func(t *testing.T) {
		apiDep := makeDeploymentObj("env-1", "SharedSpace", "Running", "target-1", "t1", "1.0.0", false)
		plannedDep := types.ObjectNull(deploymentAttrTypes)
		result := mergeDeploymentObjects(apiDep, plannedDep)
		if !result.Equal(apiDep) {
			t.Error("Expected apiDep unchanged when plannedDep is null")
		}
	})

	t.Run("null apiDep returns plannedDep", func(t *testing.T) {
		apiDep := types.ObjectNull(deploymentAttrTypes)
		plannedDep := makeDeploymentObj("env-2", "SharedSpace", "Running", "target-2", "t2", "2.0.0", true)
		result := mergeDeploymentObjects(apiDep, plannedDep)
		if !result.Equal(plannedDep) {
			t.Error("Expected plannedDep when apiDep is null")
		}
	})

	t.Run("planned values override api values", func(t *testing.T) {
		apiDep := makeDeploymentObj("env-api", "TypeA", "Running", "target-api", "t-api", "1.0.0", false)
		plannedDep := makeDeploymentObj("env-plan", "TypeB", "Started", "target-plan", "t-plan", "2.0.0", true)
		result := mergeDeploymentObjects(apiDep, plannedDep)
		if result.IsNull() {
			t.Fatal("Merge result should not be null")
		}
		attrs := result.Attributes()
		if attrs["environment_id"].(types.String).ValueString() != "env-plan" {
			t.Errorf("EnvironmentID = %v, want env-plan", attrs["environment_id"])
		}
		if attrs["type"].(types.String).ValueString() != "TypeB" {
			t.Errorf("Type = %v, want TypeB", attrs["type"])
		}
		if attrs["target_id"].(types.String).ValueString() != "target-plan" {
			t.Errorf("TargetID = %v, want target-plan", attrs["target_id"])
		}
	})
}
