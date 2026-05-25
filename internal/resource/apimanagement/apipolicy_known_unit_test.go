package apimanagement

import (
	"context"
	"math/big"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// --- fieldSchemaType ---

func TestFieldSchemaType(t *testing.T) {
	tests := []struct {
		input string
		want  attr.Type
	}{
		{"string", types.StringType},
		{"int", types.NumberType},
		{"bool", types.BoolType},
		{"string_array", types.ListType{ElemType: types.StringType}},
		{"unknown_type", types.DynamicType},
		{"", types.DynamicType},
	}
	for _, tt := range tests {
		got := fieldSchemaType(tt.input)
		if got != tt.want {
			t.Errorf("fieldSchemaType(%q) = %T, want %T", tt.input, got, tt.want)
		}
	}
}

// --- KnownPolicyTypes ---

func TestKnownPolicyTypes(t *testing.T) {
	types_ := KnownPolicyTypes()
	if len(types_) == 0 {
		t.Error("KnownPolicyTypes() should return non-empty list")
	}
	// http-caching is a well-known policy that should always be present
	found := false
	for _, pt := range types_ {
		if pt == "http-caching" {
			found = true
			break
		}
	}
	if !found {
		t.Error("KnownPolicyTypes() should include 'http-caching'")
	}
}

// --- numberAtLeastValidator ---

func TestNumberAtLeastValidator(t *testing.T) {
	v := numberAtLeastValidator{min: big.NewFloat(5)}

	t.Run("Description is non-empty", func(t *testing.T) {
		if v.Description(context.Background()) == "" {
			t.Error("Description() should be non-empty")
		}
	})

	t.Run("MarkdownDescription is non-empty", func(t *testing.T) {
		if v.MarkdownDescription(context.Background()) == "" {
			t.Error("MarkdownDescription() should be non-empty")
		}
	})

	t.Run("value above min passes", func(t *testing.T) {
		req := validator.NumberRequest{
			ConfigValue: types.NumberValue(big.NewFloat(10)),
		}
		resp := &validator.NumberResponse{}
		v.ValidateNumber(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ValidateNumber() unexpected error for value above min: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("value equal to min passes", func(t *testing.T) {
		req := validator.NumberRequest{
			ConfigValue: types.NumberValue(big.NewFloat(5)),
		}
		resp := &validator.NumberResponse{}
		v.ValidateNumber(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ValidateNumber() unexpected error for value equal to min: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("value below min fails", func(t *testing.T) {
		req := validator.NumberRequest{
			ConfigValue: types.NumberValue(big.NewFloat(3)),
		}
		resp := &validator.NumberResponse{}
		v.ValidateNumber(context.Background(), req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ValidateNumber() should error for value below min")
		}
	})

	t.Run("null value passes", func(t *testing.T) {
		req := validator.NumberRequest{
			ConfigValue: types.NumberNull(),
		}
		resp := &validator.NumberResponse{}
		v.ValidateNumber(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Error("ValidateNumber() should not error for null value")
		}
	})
}

// --- numberAtMostValidator ---

func TestNumberAtMostValidator(t *testing.T) {
	v := numberAtMostValidator{max: big.NewFloat(100)}

	t.Run("Description is non-empty", func(t *testing.T) {
		if v.Description(context.Background()) == "" {
			t.Error("Description() should be non-empty")
		}
	})

	t.Run("MarkdownDescription is non-empty", func(t *testing.T) {
		if v.MarkdownDescription(context.Background()) == "" {
			t.Error("MarkdownDescription() should be non-empty")
		}
	})

	t.Run("value below max passes", func(t *testing.T) {
		req := validator.NumberRequest{
			ConfigValue: types.NumberValue(big.NewFloat(50)),
		}
		resp := &validator.NumberResponse{}
		v.ValidateNumber(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ValidateNumber() unexpected error for value below max: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("value above max fails", func(t *testing.T) {
		req := validator.NumberRequest{
			ConfigValue: types.NumberValue(big.NewFloat(150)),
		}
		resp := &validator.NumberResponse{}
		v.ValidateNumber(context.Background(), req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ValidateNumber() should error for value above max")
		}
	})

	t.Run("null value passes", func(t *testing.T) {
		req := validator.NumberRequest{
			ConfigValue: types.NumberNull(),
		}
		resp := &validator.NumberResponse{}
		v.ValidateNumber(context.Background(), req, resp)
		if resp.Diagnostics.HasError() {
			t.Error("ValidateNumber() should not error for null value")
		}
	})
}

// --- KnownPolicyResource.resolveVersion ---

func TestKnownPolicyResource_resolveVersion(t *testing.T) {
	info, _ := apimgmtclient.LookupPolicy("http-caching")
	r := &KnownPolicyResource{policyInfo: info}

	t.Run("returns explicit version when set", func(t *testing.T) {
		data := &KnownPolicyResourceModel{
			AssetVersion: types.StringValue("2.0.0"),
		}
		v := r.resolveVersion(data)
		if v != "2.0.0" {
			t.Errorf("resolveVersion() = %q, want 2.0.0", v)
		}
	})

	t.Run("returns default version when null", func(t *testing.T) {
		data := &KnownPolicyResourceModel{
			AssetVersion: types.StringNull(),
		}
		v := r.resolveVersion(data)
		if v != info.DefaultVersion {
			t.Errorf("resolveVersion() = %q, want %q", v, info.DefaultVersion)
		}
	})

	t.Run("returns default version when empty string", func(t *testing.T) {
		data := &KnownPolicyResourceModel{
			AssetVersion: types.StringValue(""),
		}
		v := r.resolveVersion(data)
		if v != info.DefaultVersion {
			t.Errorf("resolveVersion() = %q, want %q", v, info.DefaultVersion)
		}
	})
}

// --- KnownPolicyResource.mergeConfigFromState ---

func TestKnownPolicyResource_mergeConfigFromState(t *testing.T) {
	info, _ := apimgmtclient.LookupPolicy("http-caching")
	r := &KnownPolicyResource{policyInfo: info}

	attrTypes := map[string]attr.Type{
		"field_a": types.StringType,
		"field_b": types.StringType,
	}

	makeObj := func(a, b string) types.Object {
		vals := map[string]attr.Value{
			"field_a": types.StringValue(a),
			"field_b": types.StringValue(b),
		}
		if a == "" {
			vals["field_a"] = types.StringNull()
		}
		obj, _ := types.ObjectValue(attrTypes, vals)
		return obj
	}

	t.Run("api null fields replaced with state values (write-only)", func(t *testing.T) {
		stateConfig := makeObj("state-a", "state-b")
		// API returns null for field_a (write-only), non-null for field_b
		apiVals := map[string]attr.Value{
			"field_a": types.StringNull(),
			"field_b": types.StringValue("api-b"),
		}
		apiConfig, _ := types.ObjectValue(attrTypes, apiVals)

		result := r.mergeConfigFromState(stateConfig, apiConfig)
		attrs := result.Attributes()
		if attrs["field_a"].(types.String).ValueString() != "state-a" {
			t.Errorf("field_a = %v, want state-a (restored from state)", attrs["field_a"])
		}
		if attrs["field_b"].(types.String).ValueString() != "api-b" {
			t.Errorf("field_b = %v, want api-b (API value wins)", attrs["field_b"])
		}
	})

	t.Run("api non-null values are preserved", func(t *testing.T) {
		stateConfig := makeObj("state-a", "state-b")
		apiConfig := makeObj("api-a", "api-b")

		result := r.mergeConfigFromState(stateConfig, apiConfig)
		attrs := result.Attributes()
		if attrs["field_a"].(types.String).ValueString() != "api-a" {
			t.Errorf("field_a = %v, want api-a", attrs["field_a"])
		}
	})
}

// --- generateConfigurationSchema ---

func TestGenerateConfigurationSchema(t *testing.T) {
	t.Run("known policy produces non-empty schema", func(t *testing.T) {
		s := generateConfigurationSchema("http-caching")
		if len(s.Attributes) == 0 {
			t.Error("generateConfigurationSchema() for http-caching should produce non-empty schema")
		}
	})

	t.Run("unknown policy produces empty schema", func(t *testing.T) {
		s := generateConfigurationSchema("totally-unknown-policy-xyz")
		if len(s.Attributes) != 0 {
			t.Errorf("generateConfigurationSchema() for unknown policy should produce empty schema, got %d attrs", len(s.Attributes))
		}
	})
}
