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

func TestNewPrivateSpaceConnectionResource(t *testing.T) {
	r := NewPrivateSpaceConnectionResource()

	if r == nil {
		t.Error("NewPrivateSpaceConnectionResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestPrivateSpaceConnectionResource_Metadata(t *testing.T) {
	r := NewPrivateSpaceConnectionResource()
	testutil.TestResourceMetadata(t, r, "_private_space_connection")
}

func TestPrivateSpaceConnectionResource_Schema(t *testing.T) {
	res := NewPrivateSpaceConnectionResource()

	requiredAttrs := []string{"private_space_id", "name", "type"}
	optionalAttrs := []string{}
	computedAttrs := []string{"id", "status"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestPrivateSpaceConnectionResource_Configure(t *testing.T) {
	res := NewPrivateSpaceConnectionResource().(*PrivateSpaceConnectionResource)

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

func TestPrivateSpaceConnectionResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewPrivateSpaceConnectionResource().(*PrivateSpaceConnectionResource)

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

func TestPrivateSpaceConnectionResource_ImportState(t *testing.T) {
	res := NewPrivateSpaceConnectionResource()

	ctx := context.Background()

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{
		ID: "test-space:test-conn",
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

func TestPrivateSpaceConnectionResourceModel_Validation(t *testing.T) {
	model := PrivateSpaceConnectionResourceModel{}
	_ = model.ID
}

func BenchmarkPrivateSpaceConnectionResource_Schema(b *testing.B) {
	res := NewPrivateSpaceConnectionResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
