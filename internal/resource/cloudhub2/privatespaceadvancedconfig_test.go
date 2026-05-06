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

func TestNewPrivateSpaceAdvancedConfigResource(t *testing.T) {
	r := NewPrivateSpaceAdvancedConfigResource()

	if r == nil {
		t.Error("NewPrivateSpaceAdvancedConfigResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestPrivateSpaceAdvancedConfigResource_Metadata(t *testing.T) {
	r := NewPrivateSpaceAdvancedConfigResource()
	testutil.TestResourceMetadata(t, r, "_privatespace_advanced_config")
}

func TestPrivateSpaceAdvancedConfigResource_Schema(t *testing.T) {
	res := NewPrivateSpaceAdvancedConfigResource()

	requiredAttrs := []string{"private_space_id"}
	optionalAttrs := []string{"organization_id", "ingress_configuration", "enable_iam_role"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestPrivateSpaceAdvancedConfigResource_Configure(t *testing.T) {
	res := NewPrivateSpaceAdvancedConfigResource().(*PrivateSpaceAdvancedConfigResource)

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

func TestPrivateSpaceAdvancedConfigResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewPrivateSpaceAdvancedConfigResource().(*PrivateSpaceAdvancedConfigResource)

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

func TestPrivateSpaceAdvancedConfigResource_ImportState(t *testing.T) {
	res := NewPrivateSpaceAdvancedConfigResource()

	ctx := context.Background()

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{
		ID: "test-id",
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

func TestPrivateSpaceAdvancedConfigResourceModel_Validation(t *testing.T) {
	model := PrivateSpaceAdvancedConfigResourceModel{}
	_ = model.ID
}

func TestPrivateSpaceAdvancedConfigResource_Read(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":            "test-ps-id",
				"name":          "test-private-space",
				"enableIAMRole": false,
				"ingressConfiguration": map[string]interface{}{
					"readResponseTimeout": 60,
					"protocol":            "HTTPS",
					"logs": map[string]interface{}{
						"filters":      []interface{}{},
						"portLogLevel": "INFO",
					},
					"deployment": map[string]interface{}{},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewPrivateSpaceAdvancedConfigResource().(*PrivateSpaceAdvancedConfigResource)
	res.client = &ch2client.PrivateSpaceAdvancedConfigClient{
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
	ingressObjType := objType.AttributeTypes["ingress_configuration"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"private_space_id":     tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"ingress_configuration": tftypes.NewValue(ingressObjType, nil),
		"enable_iam_role":      tftypes.NewValue(tftypes.Bool, false),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestPrivateSpaceAdvancedConfigResource_Read_NotFound(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewPrivateSpaceAdvancedConfigResource().(*PrivateSpaceAdvancedConfigResource)
	res.client = &ch2client.PrivateSpaceAdvancedConfigClient{
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
	ingressObjType := objType.AttributeTypes["ingress_configuration"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"private_space_id":     tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"ingress_configuration": tftypes.NewValue(ingressObjType, nil),
		"enable_iam_role":      tftypes.NewValue(tftypes.Bool, false),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

func BenchmarkPrivateSpaceAdvancedConfigResource_Schema(b *testing.B) {
	res := NewPrivateSpaceAdvancedConfigResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
