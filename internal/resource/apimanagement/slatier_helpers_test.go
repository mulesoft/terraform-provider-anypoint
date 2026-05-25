package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// --- expandLimits ---

func TestSLATierResource_expandLimits(t *testing.T) {
	r := &SLATierResource{}
	ctx := context.Background()

	t.Run("empty list returns empty slice", func(t *testing.T) {
		listVal, _ := types.ListValueFrom(ctx, slaLimitObjectType, []SLALimitModel{})
		limits, diags := r.expandLimits(ctx, listVal)
		if diags.HasError() {
			t.Fatalf("expandLimits() unexpected diags: %v", diags.Errors())
		}
		if len(limits) != 0 {
			t.Errorf("expandLimits() len = %d, want 0", len(limits))
		}
	})

	t.Run("single limit is expanded", func(t *testing.T) {
		models := []SLALimitModel{
			{
				TimePeriodInMilliseconds: types.Int64Value(1000),
				MaximumRequests:          types.Int64Value(100),
				Visible:                  types.BoolValue(true),
			},
		}
		listVal, _ := types.ListValueFrom(ctx, slaLimitObjectType, models)
		limits, diags := r.expandLimits(ctx, listVal)
		if diags.HasError() {
			t.Fatalf("expandLimits() unexpected diags: %v", diags.Errors())
		}
		if len(limits) != 1 {
			t.Fatalf("expandLimits() len = %d, want 1", len(limits))
		}
		if limits[0].TimePeriodInMilliseconds != 1000 {
			t.Errorf("TimePeriodInMilliseconds = %d, want 1000", limits[0].TimePeriodInMilliseconds)
		}
		if limits[0].MaximumRequests != 100 {
			t.Errorf("MaximumRequests = %d, want 100", limits[0].MaximumRequests)
		}
		if !limits[0].Visible {
			t.Error("Visible = false, want true")
		}
	})

	t.Run("multiple limits are expanded", func(t *testing.T) {
		models := []SLALimitModel{
			{
				TimePeriodInMilliseconds: types.Int64Value(60000),
				MaximumRequests:          types.Int64Value(1000),
				Visible:                  types.BoolValue(true),
			},
			{
				TimePeriodInMilliseconds: types.Int64Value(3600000),
				MaximumRequests:          types.Int64Value(10000),
				Visible:                  types.BoolValue(false),
			},
		}
		listVal, _ := types.ListValueFrom(ctx, slaLimitObjectType, models)
		limits, diags := r.expandLimits(ctx, listVal)
		if diags.HasError() {
			t.Fatalf("expandLimits() unexpected diags: %v", diags.Errors())
		}
		if len(limits) != 2 {
			t.Fatalf("expandLimits() len = %d, want 2", len(limits))
		}
		if limits[1].MaximumRequests != 10000 {
			t.Errorf("Second limit MaximumRequests = %d, want 10000", limits[1].MaximumRequests)
		}
		if limits[1].Visible {
			t.Error("Second limit Visible = true, want false")
		}
	})
}

// --- flattenTier ---

func TestSLATierResource_flattenTier(t *testing.T) {
	r := &SLATierResource{}
	ctx := context.Background()

	t.Run("basic tier is flattened", func(t *testing.T) {
		tier := &apimgmtclient.SLATier{
			ID:          42,
			Name:        "Gold",
			Description: "Gold tier",
			AutoApprove: true,
			Status:      "ACTIVE",
			Limits: []apimgmtclient.SLALimit{
				{TimePeriodInMilliseconds: 1000, MaximumRequests: 100, Visible: true},
			},
		}
		data := &SLATierResourceModel{}

		diags := r.flattenTier(ctx, tier, data, "org-1", "env-2")
		if diags.HasError() {
			t.Fatalf("flattenTier() unexpected diags: %v", diags.Errors())
		}

		if data.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", data.ID.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
		if data.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", data.EnvironmentID.ValueString())
		}
		if data.Name.ValueString() != "Gold" {
			t.Errorf("Name = %q, want Gold", data.Name.ValueString())
		}
		if !data.AutoApprove.ValueBool() {
			t.Error("AutoApprove = false, want true")
		}
		if data.Description.ValueString() != "Gold tier" {
			t.Errorf("Description = %q, want 'Gold tier'", data.Description.ValueString())
		}
		if data.Status.ValueString() != "ACTIVE" {
			t.Errorf("Status = %q, want ACTIVE", data.Status.ValueString())
		}
	})

	t.Run("limits list is populated correctly", func(t *testing.T) {
		tier := &apimgmtclient.SLATier{
			ID:   1,
			Name: "Bronze",
			Limits: []apimgmtclient.SLALimit{
				{TimePeriodInMilliseconds: 60000, MaximumRequests: 500, Visible: true},
				{TimePeriodInMilliseconds: 3600000, MaximumRequests: 5000, Visible: false},
			},
		}
		data := &SLATierResourceModel{}
		diags := r.flattenTier(ctx, tier, data, "o", "e")
		if diags.HasError() {
			t.Fatalf("flattenTier() unexpected diags: %v", diags.Errors())
		}

		var limitModels []SLALimitModel
		data.Limits.ElementsAs(ctx, &limitModels, false)
		if len(limitModels) != 2 {
			t.Fatalf("Limits len = %d, want 2", len(limitModels))
		}
		if limitModels[0].MaximumRequests.ValueInt64() != 500 {
			t.Errorf("First limit MaximumRequests = %d, want 500", limitModels[0].MaximumRequests.ValueInt64())
		}
	})

	t.Run("empty description and status are omitted", func(t *testing.T) {
		tier := &apimgmtclient.SLATier{
			ID:          3,
			Name:        "Silver",
			Description: "",
			Status:      "",
			Limits:      []apimgmtclient.SLALimit{},
		}
		data := &SLATierResourceModel{}
		diags := r.flattenTier(ctx, tier, data, "o", "e")
		if diags.HasError() {
			t.Fatalf("flattenTier() unexpected diags: %v", diags.Errors())
		}
		// Empty string fields — Description and Status should remain zero value (not overwritten)
		if !data.Description.IsNull() && data.Description.ValueString() != "" {
			t.Errorf("Description should be empty/null, got %q", data.Description.ValueString())
		}
	})
}
