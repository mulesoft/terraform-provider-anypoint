package apimanagement

import (
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// --- resolvePolicyIdentifiers ---

func TestAPIPolicyResource_resolvePolicyIdentifiers(t *testing.T) {
	r := &APIPolicyResource{}

	t.Run("known policy_type resolves identifiers", func(t *testing.T) {
		data := &APIPolicyResourceModel{
			PolicyType:   types.StringValue("http-caching"),
			GroupID:      types.StringNull(),
			AssetID:      types.StringNull(),
			AssetVersion: types.StringNull(),
		}
		groupID, assetID, assetVersion, err := r.resolvePolicyIdentifiers(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if groupID == "" {
			t.Error("groupID should not be empty for known policy")
		}
		if assetID != "http-caching" {
			t.Errorf("assetID = %q, want http-caching", assetID)
		}
		if assetVersion == "" {
			t.Error("assetVersion should not be empty for known policy")
		}
	})

	t.Run("unknown policy_type returns error", func(t *testing.T) {
		data := &APIPolicyResourceModel{
			PolicyType:   types.StringValue("completely-unknown-policy-xyz"),
			GroupID:      types.StringNull(),
			AssetID:      types.StringNull(),
			AssetVersion: types.StringNull(),
		}
		_, _, _, err := r.resolvePolicyIdentifiers(data)
		if err == nil {
			t.Fatal("expected error for unknown policy_type")
		}
	})

	t.Run("no policy_type with all explicit fields succeeds", func(t *testing.T) {
		data := &APIPolicyResourceModel{
			PolicyType:   types.StringValue(""),
			GroupID:      types.StringValue("com.mulesoft.policies"),
			AssetID:      types.StringValue("my-custom-policy"),
			AssetVersion: types.StringValue("1.2.3"),
		}
		groupID, assetID, assetVersion, err := r.resolvePolicyIdentifiers(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if groupID != "com.mulesoft.policies" {
			t.Errorf("groupID = %q, want com.mulesoft.policies", groupID)
		}
		if assetID != "my-custom-policy" {
			t.Errorf("assetID = %q, want my-custom-policy", assetID)
		}
		if assetVersion != "1.2.3" {
			t.Errorf("assetVersion = %q, want 1.2.3", assetVersion)
		}
	})

	t.Run("no policy_type with missing fields returns error", func(t *testing.T) {
		data := &APIPolicyResourceModel{
			PolicyType:   types.StringValue(""),
			GroupID:      types.StringValue("com.mulesoft.policies"),
			AssetID:      types.StringValue("my-policy"),
			AssetVersion: types.StringValue(""), // missing
		}
		_, _, _, err := r.resolvePolicyIdentifiers(data)
		if err == nil {
			t.Fatal("expected error when asset_version is empty")
		}
	})

	t.Run("known policy_type with override fields uses overrides", func(t *testing.T) {
		data := &APIPolicyResourceModel{
			PolicyType:   types.StringValue("http-caching"),
			GroupID:      types.StringValue("override-group"),
			AssetID:      types.StringValue("override-asset"),
			AssetVersion: types.StringValue("9.9.9"),
		}
		groupID, assetID, assetVersion, err := r.resolvePolicyIdentifiers(data)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if groupID != "override-group" {
			t.Errorf("groupID = %q, want override-group", groupID)
		}
		if assetID != "override-asset" {
			t.Errorf("assetID = %q, want override-asset", assetID)
		}
		if assetVersion != "9.9.9" {
			t.Errorf("assetVersion = %q, want 9.9.9", assetVersion)
		}
	})
}

// --- validateConfigurationData ---

func TestAPIPolicyResource_validateConfigurationData(t *testing.T) {
	r := &APIPolicyResource{}

	t.Run("invalid JSON returns error", func(t *testing.T) {
		errs := r.validateConfigurationData("http-caching", `{invalid json}`)
		if len(errs) == 0 {
			t.Error("Expected errors for invalid JSON")
		}
	})

	t.Run("valid JSON for unknown assetID returns no errors", func(t *testing.T) {
		errs := r.validateConfigurationData("unknown-policy-xyz", `{"key": "value"}`)
		if len(errs) != 0 {
			t.Errorf("Expected no errors for unknown policy, got: %v", errs)
		}
	})

	t.Run("valid JSON passes for empty config", func(t *testing.T) {
		errs := r.validateConfigurationData("unknown-policy-xyz", `{}`)
		if len(errs) != 0 {
			t.Errorf("Expected no errors for empty config of unknown policy, got: %v", errs)
		}
	})
}

// --- flattenPolicy ---

func TestAPIPolicyResource_flattenPolicy(t *testing.T) {
	r := &APIPolicyResource{}

	t.Run("basic policy is flattened correctly", func(t *testing.T) {
		policy := &apimgmtclient.APIPolicy{
			ID:               42,
			APIID:            100,
			GroupID:          "com.mulesoft.policies",
			AssetID:          "http-caching",
			AssetVersion:     "1.0.0",
			Order:            2,
			Disabled:         false,
			PolicyTemplateID: "template-xyz",
			Label:            "my-label",
			ConfigurationData: map[string]interface{}{
				"key": "value",
			},
		}
		data := &APIPolicyResourceModel{
			PolicyType: types.StringNull(),
		}

		r.flattenPolicy(policy, data, "org-1", "env-2")

		if data.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", data.ID.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
		if data.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", data.EnvironmentID.ValueString())
		}
		if data.APIInstanceID.ValueString() != "100" {
			t.Errorf("APIInstanceID = %q, want 100", data.APIInstanceID.ValueString())
		}
		if data.GroupID.ValueString() != "com.mulesoft.policies" {
			t.Errorf("GroupID = %q, want com.mulesoft.policies", data.GroupID.ValueString())
		}
		if data.AssetID.ValueString() != "http-caching" {
			t.Errorf("AssetID = %q, want http-caching", data.AssetID.ValueString())
		}
		if data.AssetVersion.ValueString() != "1.0.0" {
			t.Errorf("AssetVersion = %q, want 1.0.0", data.AssetVersion.ValueString())
		}
		if data.Order.ValueInt64() != 2 {
			t.Errorf("Order = %d, want 2", data.Order.ValueInt64())
		}
		if data.Disabled.ValueBool() != false {
			t.Errorf("Disabled = true, want false")
		}
		if data.PolicyTemplateID.ValueString() != "template-xyz" {
			t.Errorf("PolicyTemplateID = %q, want template-xyz", data.PolicyTemplateID.ValueString())
		}
		if data.Label.ValueString() != "my-label" {
			t.Errorf("Label = %q, want my-label", data.Label.ValueString())
		}
		if data.ConfigurationData.IsNull() {
			t.Error("ConfigurationData should not be null")
		}
	})

	t.Run("known assetID sets policy_type", func(t *testing.T) {
		policy := &apimgmtclient.APIPolicy{
			ID:       1,
			AssetID:  "http-caching",
			GroupID:  "com.mulesoft.policies",
			AssetVersion: "1.0.0",
		}
		data := &APIPolicyResourceModel{
			PolicyType: types.StringNull(),
		}
		r.flattenPolicy(policy, data, "org-1", "env-1")
		if data.PolicyType.ValueString() != "http-caching" {
			t.Errorf("PolicyType = %q, want http-caching", data.PolicyType.ValueString())
		}
	})

	t.Run("APIID zero does not set api_instance_id", func(t *testing.T) {
		policy := &apimgmtclient.APIPolicy{
			ID:      5,
			APIID:   0, // zero → skip
			AssetID: "unknown-custom-policy",
		}
		data := &APIPolicyResourceModel{
			APIInstanceID: types.StringValue("existing"),
			PolicyType:    types.StringNull(),
		}
		r.flattenPolicy(policy, data, "org-1", "env-1")
		// APIID is 0 so APIInstanceID should remain unchanged
		if data.APIInstanceID.ValueString() != "existing" {
			t.Errorf("APIInstanceID should not change when APIID=0, got %q", data.APIInstanceID.ValueString())
		}
	})

	t.Run("nil ConfigurationData leaves field null-ish", func(t *testing.T) {
		policy := &apimgmtclient.APIPolicy{
			ID:                1,
			AssetID:           "x",
			ConfigurationData: nil,
		}
		data := &APIPolicyResourceModel{PolicyType: types.StringNull()}
		r.flattenPolicy(policy, data, "o", "e")
		// No panic expected; ConfigurationData should remain zero value
	})
}
