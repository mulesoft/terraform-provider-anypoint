package cloudhub2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewPrivateSpaceConfigResource(t *testing.T) {
	r := NewPrivateSpaceConfigResource()
	if r == nil {
		t.Error("NewPrivateSpaceConfigResource() returned nil")
	}
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("PrivateSpaceConfigResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("PrivateSpaceConfigResource should implement ResourceWithImportState")
	}
}

func TestPrivateSpaceConfigResource_Metadata(t *testing.T) {
	r := NewPrivateSpaceConfigResource()
	testutil.TestResourceMetadata(t, r, "_private_space_config")
}

func TestPrivateSpaceConfigResource_Schema(t *testing.T) {
	r := NewPrivateSpaceConfigResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}
	r.Schema(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	required := []string{"name"}
	for _, attr := range required {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing required attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("Schema() attribute %q should be required", attr)
		}
	}
	computed := []string{"id", "status"}
	for _, attr := range computed {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Schema() attribute %q should be computed", attr)
		}
	}
}

func TestPrivateSpaceConfigResource_Configure(t *testing.T) {
	res := NewPrivateSpaceConfigResource().(*PrivateSpaceConfigResource)
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	ctx := context.Background()
	req := resource.ConfigureRequest{ProviderData: providerData}
	resp := &resource.ConfigureResponse{}
	res.Configure(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has unexpected errors: %v", resp.Diagnostics.Errors())
	}
	if res.spaceClient == nil {
		t.Error("Configure() should set spaceClient")
	}
	if res.networkClient == nil {
		t.Error("Configure() should set networkClient")
	}
	if res.firewallClient == nil {
		t.Error("Configure() should set firewallClient")
	}
}

func TestPrivateSpaceConfigResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewPrivateSpaceConfigResource().(*PrivateSpaceConfigResource)
	ctx := context.Background()
	req := resource.ConfigureRequest{ProviderData: "invalid"}
	resp := &resource.ConfigureResponse{}
	res.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
}

func TestPrivateSpaceConfigResource_ImportState(t *testing.T) {
	r := NewPrivateSpaceConfigResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestPrivateSpaceConfigResource_ImportState_SimpleID(t *testing.T) {
	res := NewPrivateSpaceConfigResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{ID: "test-space-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)},
	}

	res.(resource.ResourceWithImportState).ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with simple ID reported errors: %v", resp.Diagnostics.Errors())
	}

	var model PrivateSpaceConfigResourceModel
	resp.State.Get(ctx, &model)
	if model.ID.ValueString() != "test-space-id" {
		t.Errorf("expected id = %q, got %q", "test-space-id", model.ID.ValueString())
	}
	if !model.OrganizationID.IsNull() {
		t.Errorf("expected organization_id to be null for simple import, got %q", model.OrganizationID.ValueString())
	}
}

func TestPrivateSpaceConfigResource_ImportState_CompositeID(t *testing.T) {
	res := NewPrivateSpaceConfigResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{ID: "test-org-id/test-space-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)},
	}

	res.(resource.ResourceWithImportState).ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with composite ID reported errors: %v", resp.Diagnostics.Errors())
	}

	var model PrivateSpaceConfigResourceModel
	resp.State.Get(ctx, &model)
	if model.ID.ValueString() != "test-space-id" {
		t.Errorf("expected id = %q, got %q", "test-space-id", model.ID.ValueString())
	}
	if model.OrganizationID.ValueString() != "test-org-id" {
		t.Errorf("expected organization_id = %q, got %q", "test-org-id", model.OrganizationID.ValueString())
	}
}

func TestPrivateSpaceConfigResourceModel_Validation(t *testing.T) {
	model := PrivateSpaceConfigResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.OrganizationID
	_ = model.EnableIAMRole
	_ = model.EnableEgress
	_ = model.Status
	_ = model.MuleAppDeploymentCount
	_ = model.DaysLeftForRelaxedQuota
	_ = model.VPCMigrationInProgress
	_ = model.ManagedFirewallRules
	_ = model.GlobalSpaceStatus
	_ = model.Network
	_ = model.FirewallRules
}
