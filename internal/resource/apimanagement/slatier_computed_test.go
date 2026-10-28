package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// The SLA tier endpoints do not always echo description and status back on update. Both
// are Optional+Computed, so the plan carries them as unknown when they are absent from
// configuration, and an unknown that survives apply fails the whole operation with
// "Provider returned invalid result object after apply" — which is what an SLA tier
// attached to an MCP bridge hit on a limit change.
func TestFlattenTier_ResolvesUnknownComputedFields(t *testing.T) {
	r := &SLATierResource{}
	data := &SLATierResourceModel{
		Description: types.StringUnknown(),
		Status:      types.StringUnknown(),
	}

	tier := &apimanagement.SLATier{Name: "gold"} // platform returned neither field

	diags := r.flattenTier(context.Background(), tier, data, "org-1", "env-1")
	if diags.HasError() {
		t.Fatalf("flattenTier: %v", diags)
	}

	if data.Status.IsUnknown() {
		t.Error("status must not stay unknown after apply; Terraform rejects the result object")
	}
	if data.Description.IsUnknown() {
		t.Error("description must not stay unknown after apply")
	}
	if !data.Status.IsNull() {
		t.Errorf("status = %v, want null when the platform reports none", data.Status)
	}
}

// A value the platform does report must still win.
func TestFlattenTier_KeepsReportedValues(t *testing.T) {
	r := &SLATierResource{}
	data := &SLATierResourceModel{
		Description: types.StringUnknown(),
		Status:      types.StringUnknown(),
	}

	tier := &apimanagement.SLATier{Name: "gold", Description: "tier docs", Status: "ACTIVE"}

	if diags := r.flattenTier(context.Background(), tier, data, "org-1", "env-1"); diags.HasError() {
		t.Fatalf("flattenTier: %v", diags)
	}
	if data.Status.ValueString() != "ACTIVE" {
		t.Errorf("status = %q, want ACTIVE", data.Status.ValueString())
	}
	if data.Description.ValueString() != "tier docs" {
		t.Errorf("description = %q, want \"tier docs\"", data.Description.ValueString())
	}
}

// A configured value must not be overwritten with null just because the update response
// omitted it.
func TestFlattenTier_LeavesConfiguredValueAlone(t *testing.T) {
	r := &SLATierResource{}
	data := &SLATierResourceModel{
		Description: types.StringValue("set by config"),
		Status:      types.StringValue("ACTIVE"),
	}

	if diags := r.flattenTier(context.Background(), &apimanagement.SLATier{Name: "gold"}, data, "org-1", "env-1"); diags.HasError() {
		t.Fatalf("flattenTier: %v", diags)
	}
	if data.Status.ValueString() != "ACTIVE" {
		t.Errorf("a configured status must survive a response that omits it, got %v", data.Status)
	}
	if data.Description.ValueString() != "set by config" {
		t.Errorf("a configured description must survive, got %v", data.Description)
	}
}
