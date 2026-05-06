package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewVPNConnectionResource(t *testing.T) {
	r := NewVPNConnectionResource()

	if r == nil {
		t.Error("NewVPNConnectionResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestVPNConnectionResource_Metadata(t *testing.T) {
	r := NewVPNConnectionResource()
	testutil.TestResourceMetadata(t, r, "_vpn_connection")
}

func TestVPNConnectionResource_Schema(t *testing.T) {
	res := NewVPNConnectionResource()

	requiredAttrs := []string{"private_space_id", "name", "vpns"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestVPNConnectionResource_Configure(t *testing.T) {
	res := NewVPNConnectionResource().(*VPNConnectionResource)

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

func TestVPNConnectionResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewVPNConnectionResource().(*VPNConnectionResource)

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

func TestVPNConnectionResource_ImportState(t *testing.T) {
	res := NewVPNConnectionResource()

	ctx := context.Background()

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{
		ID: "test-space/test-conn",
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

func TestVPNConnectionResourceModel_Validation(t *testing.T) {
	model := VPNConnectionResourceModel{}
	_ = model.ID
}

func TestVPNConnectionResource_Read(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/connections/test-vpn-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":   "test-vpn-id",
				"name": "test-vpn",
				"vpns": []interface{}{},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewVPNConnectionResource().(*VPNConnectionResource)
	res.client = &ch2client.VPNConnectionClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	vpnsElemType := objType.AttributeTypes["vpns"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, "test-vpn-id"),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":             tftypes.NewValue(tftypes.String, "test-vpn"),
		"vpns":             tftypes.NewValue(tftypes.List{ElementType: vpnsElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got VPNConnectionResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "test-vpn" {
		t.Errorf("Expected Name 'test-vpn', got %s", got.Name.ValueString())
	}
}

func TestVPNConnectionResource_Read_NotFound(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/connections/test-vpn-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewVPNConnectionResource().(*VPNConnectionResource)
	res.client = &ch2client.VPNConnectionClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	vpnsElemType := objType.AttributeTypes["vpns"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, "test-vpn-id"),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":             tftypes.NewValue(tftypes.String, "test-vpn"),
		"vpns":             tftypes.NewValue(tftypes.List{ElementType: vpnsElemType}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

func BenchmarkVPNConnectionResource_Schema(b *testing.B) {
	res := NewVPNConnectionResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
