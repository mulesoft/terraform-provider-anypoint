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

func TestNewPrivateSpaceUpgradeResource(t *testing.T) {
	r := NewPrivateSpaceUpgradeResource()

	if r == nil {
		t.Error("NewPrivateSpaceUpgradeResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestPrivateSpaceUpgradeResource_Metadata(t *testing.T) {
	r := NewPrivateSpaceUpgradeResource()
	testutil.TestResourceMetadata(t, r, "_private_space_upgrade")
}

func TestPrivateSpaceUpgradeResource_Schema(t *testing.T) {
	res := NewPrivateSpaceUpgradeResource()

	requiredAttrs := []string{"private_space_id", "date", "opt_in"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id", "scheduled_update_time", "status"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestPrivateSpaceUpgradeResource_Configure(t *testing.T) {
	res := NewPrivateSpaceUpgradeResource().(*PrivateSpaceUpgradeResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestPrivateSpaceUpgradeResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewPrivateSpaceUpgradeResource().(*PrivateSpaceUpgradeResource)

	ctx := context.Background()
	req := resource.ConfigureRequest{
		ProviderData: "invalid-data",
	}
	resp := &resource.ConfigureResponse{}

	res.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}

	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestPrivateSpaceUpgradeResource_ImportState(t *testing.T) {
	res := NewPrivateSpaceUpgradeResource()

	ctx := context.Background()

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{
		ID: "test-space:2025-08-12:true",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(stateType, nil),
		},
	}

	if importableResource, ok := res.(resource.ResourceWithImportState); ok {
		importableResource.ImportState(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("ImportState() has errors: %v", resp.Diagnostics.Errors())
		}
	} else {
		t.Error("Resource does not implement ResourceWithImportState")
	}
}

func TestPrivateSpaceUpgradeResourceModel_Validation(t *testing.T) {
	model := PrivateSpaceUpgradeResourceModel{}
	_ = model.ID
}

func BenchmarkPrivateSpaceUpgradeResource_Schema(b *testing.B) {
	res := NewPrivateSpaceUpgradeResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
