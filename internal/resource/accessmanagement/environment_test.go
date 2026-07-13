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

func TestNewEnvironmentResource(t *testing.T) {
	r := NewEnvironmentResource()

	if r == nil {
		t.Error("NewEnvironmentResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("EnvironmentResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("EnvironmentResource should implement ResourceWithImportState")
	}
}

func TestEnvironmentResource_Metadata(t *testing.T) {
	r := NewEnvironmentResource()
	testutil.TestResourceMetadata(t, r, "_environment")
}

func TestEnvironmentResource_Schema(t *testing.T) {
	res := NewEnvironmentResource()

	requiredAttrs := []string{"name"}
	optionalAttrs := []string{"type", "is_production", "organization_id", "client_id", "arc_namespace"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestEnvironmentResource_Configure(t *testing.T) {
	res := NewEnvironmentResource().(*EnvironmentResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestEnvironmentResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewEnvironmentResource().(*EnvironmentResource)

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

func TestEnvironmentResource_ImportState(t *testing.T) {
	r := NewEnvironmentResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestEnvironmentResourceModel_Validation(t *testing.T) {
	model := EnvironmentResourceModel{}
	_ = model.ID
}

func TestEnvironmentResource_Read(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/environments/test-env-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-env-id",
				"name":           "My Environment",
				"type":           "sandbox",
				"isProduction":   false,
				"organizationId": "test-org-id",
				"clientId":       "test-client-id",
				"arcNamespace":   nil,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewEnvironmentResource().(*EnvironmentResource)
	res.client = &accessmanagement.EnvironmentClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-env-id"),
		"name":            tftypes.NewValue(tftypes.String, "My Environment"),
		"type":            tftypes.NewValue(tftypes.String, "sandbox"),
		"is_production":   tftypes.NewValue(tftypes.Bool, false),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"client_id":       tftypes.NewValue(tftypes.String, "test-client-id"),
		"arc_namespace":   tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got EnvironmentResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "My Environment" {
		t.Errorf("Expected Name 'My Environment', got %s", got.Name.ValueString())
	}
}

func TestEnvironmentResource_Read_NotFound(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/environments/test-env-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewEnvironmentResource().(*EnvironmentResource)
	res.client = &accessmanagement.EnvironmentClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-env-id"),
		"name":            tftypes.NewValue(tftypes.String, "My Environment"),
		"type":            tftypes.NewValue(tftypes.String, "sandbox"),
		"is_production":   tftypes.NewValue(tftypes.Bool, false),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"client_id":       tftypes.NewValue(tftypes.String, "test-client-id"),
		"arc_namespace":   tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

func BenchmarkEnvironmentResource_Schema(b *testing.B) {
	res := NewEnvironmentResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
