package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewConnectedAppResource(t *testing.T) {
	r := NewConnectedAppResource()

	if r == nil {
		t.Error("NewConnectedAppResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("ConnectedAppResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("ConnectedAppResource should implement ResourceWithImportState")
	}
}

func TestConnectedAppResource_Metadata(t *testing.T) {
	r := NewConnectedAppResource()
	testutil.TestResourceMetadata(t, r, "_connected_app")
}

func TestConnectedAppResource_Schema(t *testing.T) {
	res := NewConnectedAppResource()

	requiredAttrs := []string{"client_id", "owner_org_id", "client_name", "client_secret", "grant_types", "audience"}
	optionalAttrs := []string{"public_keys", "redirect_uris", "scopes"}
	computedAttrs := []string{"enabled", "generate_iss_claim_without_token"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestConnectedAppResource_Configure(t *testing.T) {
	res := NewConnectedAppResource().(*ConnectedAppResource)

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

func TestConnectedAppResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewConnectedAppResource().(*ConnectedAppResource)

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

func TestConnectedAppResource_ImportState(t *testing.T) {
	r := NewConnectedAppResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestConnectedAppResourceModel_Validation(t *testing.T) {
	model := ConnectedAppResourceModel{}
	_ = model.ClientID
}

func TestConnectedAppResource_Read(t *testing.T) {
	basePath := "/accounts/api/connectedApplications/test-client-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"client_id":                       "test-client-id",
				"owner_org_id":                    "test-org-id",
				"client_name":                     "Test App",
				"client_secret":                   "test-secret",
				"public_keys":                     []string{},
				"redirect_uris":                   []string{},
				"grant_types":                     []string{"client_credentials"},
				"scopes":                          []string{},
				"enabled":                         true,
				"audience":                        "internal",
				"generate_iss_claim_without_token": false,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewConnectedAppResource().(*ConnectedAppResource)
	res.client = &accessmanagement.ConnectedAppClient{
		AnypointClient: &client.AnypointClient{
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
	publicKeysElemType := objType.AttributeTypes["public_keys"].(tftypes.List).ElementType
	redirectURIsElemType := objType.AttributeTypes["redirect_uris"].(tftypes.List).ElementType
	grantTypesElemType := objType.AttributeTypes["grant_types"].(tftypes.List).ElementType
	scopesElemType := objType.AttributeTypes["scopes"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"client_id":                        tftypes.NewValue(tftypes.String, "test-client-id"),
		"owner_org_id":                     tftypes.NewValue(tftypes.String, "test-org-id"),
		"client_name":                      tftypes.NewValue(tftypes.String, "Test App"),
		"client_secret":                    tftypes.NewValue(tftypes.String, "test-secret"),
		"public_keys":                      tftypes.NewValue(tftypes.List{ElementType: publicKeysElemType}, nil),
		"redirect_uris":                    tftypes.NewValue(tftypes.List{ElementType: redirectURIsElemType}, nil),
		"grant_types":                      tftypes.NewValue(tftypes.List{ElementType: grantTypesElemType}, nil),
		"scopes":                           tftypes.NewValue(tftypes.List{ElementType: scopesElemType}, nil),
		"enabled":                          tftypes.NewValue(tftypes.Bool, true),
		"audience":                         tftypes.NewValue(tftypes.String, "internal"),
		"generate_iss_claim_without_token": tftypes.NewValue(tftypes.Bool, false),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got ConnectedAppResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ClientName.ValueString() != "Test App" {
		t.Errorf("Expected ClientName 'Test App', got %s", got.ClientName.ValueString())
	}
}

func TestConnectedAppResource_Read_NotFound(t *testing.T) {
	basePath := "/accounts/api/connectedApplications/test-client-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewConnectedAppResource().(*ConnectedAppResource)
	res.client = &accessmanagement.ConnectedAppClient{
		AnypointClient: &client.AnypointClient{
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
	publicKeysElemType := objType.AttributeTypes["public_keys"].(tftypes.List).ElementType
	redirectURIsElemType := objType.AttributeTypes["redirect_uris"].(tftypes.List).ElementType
	grantTypesElemType := objType.AttributeTypes["grant_types"].(tftypes.List).ElementType
	scopesElemType := objType.AttributeTypes["scopes"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"client_id":                        tftypes.NewValue(tftypes.String, "test-client-id"),
		"owner_org_id":                     tftypes.NewValue(tftypes.String, "test-org-id"),
		"client_name":                      tftypes.NewValue(tftypes.String, "Test App"),
		"client_secret":                    tftypes.NewValue(tftypes.String, "test-secret"),
		"public_keys":                      tftypes.NewValue(tftypes.List{ElementType: publicKeysElemType}, nil),
		"redirect_uris":                    tftypes.NewValue(tftypes.List{ElementType: redirectURIsElemType}, nil),
		"grant_types":                      tftypes.NewValue(tftypes.List{ElementType: grantTypesElemType}, nil),
		"scopes":                           tftypes.NewValue(tftypes.List{ElementType: scopesElemType}, nil),
		"enabled":                          tftypes.NewValue(tftypes.Bool, true),
		"audience":                         tftypes.NewValue(tftypes.String, "internal"),
		"generate_iss_claim_without_token": tftypes.NewValue(tftypes.Bool, false),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

func BenchmarkConnectedAppResource_Schema(b *testing.B) {
	res := NewConnectedAppResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
