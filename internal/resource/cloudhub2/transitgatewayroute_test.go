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
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTransitGatewayRouteResource(t *testing.T) {
	r := NewTransitGatewayRouteResource()

	if r == nil {
		t.Error("NewTransitGatewayRouteResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestTransitGatewayRouteResource_Metadata(t *testing.T) {
	r := NewTransitGatewayRouteResource()
	testutil.TestResourceMetadata(t, r, "_transit_gateway_route")
}

func TestTransitGatewayRouteResource_Schema(t *testing.T) {
	res := NewTransitGatewayRouteResource()

	requiredAttrs := []string{"cidr", "transit_gateway_id", "private_space_id", "organization_id"}
	optionalAttrs := []string{}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTransitGatewayRouteResource_Configure(t *testing.T) {
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestTransitGatewayRouteResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)

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

func TestTransitGatewayRouteResource_Read(t *testing.T) {
	routesPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123/routes"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		routesPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []string{"10.0.0.0/8", "172.16.0.0/12"})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-ps-id/tgw-123/10.0.0.0/8"),
		"cidr":               tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
		"transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-123"),
		"private_space_id":   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewayRouteResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.CIDR.ValueString() != "10.0.0.0/8" {
		t.Errorf("Expected CIDR '10.0.0.0/8', got %s", got.CIDR.ValueString())
	}
}

func TestTransitGatewayRouteResource_Read_RouteNotFound(t *testing.T) {
	routesPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123/routes"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		routesPath: func(w http.ResponseWriter, r *http.Request) {
			// Route list does NOT contain our route — it was removed externally
			testutil.JSONResponse(w, http.StatusOK, []string{"172.16.0.0/12"})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-ps-id/tgw-123/10.0.0.0/8"),
		"cidr":               tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
		"transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-123"),
		"private_space_id":   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() should not error when route is missing, got: %v", resp.Diagnostics.Errors())
	}

	if !resp.State.Raw.IsNull() {
		t.Error("Read() should remove resource from state when route is not found in list")
	}
}

func TestTransitGatewayRouteResource_Create(t *testing.T) {
	routesPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123/routes"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		routesPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				testutil.JSONResponse(w, http.StatusOK, []string{"10.0.0.0/8"})
				return
			}
			if r.Method == "PATCH" {
				w.WriteHeader(http.StatusOK)
				return
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"cidr":               tftypes.NewValue(tftypes.String, "172.16.0.0/12"),
		"transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-123"),
		"private_space_id":   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewayRouteResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	expectedID := "test-org-id/test-ps-id/tgw-123/172.16.0.0/12"
	if got.ID.ValueString() != expectedID {
		t.Errorf("Expected ID '%s', got '%s'", expectedID, got.ID.ValueString())
	}
}

func TestTransitGatewayRouteResource_Create_DuplicateRoute(t *testing.T) {
	routesPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123/routes"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		routesPath: func(w http.ResponseWriter, r *http.Request) {
			// Route already exists in the list
			testutil.JSONResponse(w, http.StatusOK, []string{"10.0.0.0/8", "172.16.0.0/12"})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, nil),
		"cidr":               tftypes.NewValue(tftypes.String, "172.16.0.0/12"),
		"transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-123"),
		"private_space_id":   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Create() should error when route already exists")
	}
}

func TestTransitGatewayRouteResource_Delete(t *testing.T) {
	routesPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123/routes"
	patchCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		routesPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				testutil.JSONResponse(w, http.StatusOK, []string{"10.0.0.0/8", "172.16.0.0/12"})
				return
			}
			if r.Method == "PATCH" {
				patchCalled = true
				w.WriteHeader(http.StatusOK)
				return
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	stateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-ps-id/tgw-123/10.0.0.0/8"),
		"cidr":               tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
		"transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-123"),
		"private_space_id":   tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
	})

	req := resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	resp := &resource.DeleteResponse{}
	res.Delete(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() reported errors: %v", resp.Diagnostics.Errors())
	}
	if !patchCalled {
		t.Error("Delete() should call PATCH with updated route list")
	}
}

func TestTransitGatewayRouteResource_ImportState_ValidID(t *testing.T) {
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	initialState := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, ""),
		"cidr":               tftypes.NewValue(tftypes.String, ""),
		"transit_gateway_id": tftypes.NewValue(tftypes.String, ""),
		"private_space_id":   tftypes.NewValue(tftypes.String, ""),
		"organization_id":    tftypes.NewValue(tftypes.String, ""),
	})

	// Import ID: org_id/ps_id/tgw_id/cidr (CIDR has a slash, so 5 parts)
	req := resource.ImportStateRequest{ID: "org-123/ps-456/tgw-789/10.0.0.0/8"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: initialState},
	}

	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with valid ID should not error: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewayRouteResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.OrganizationID.ValueString() != "org-123" {
		t.Errorf("Expected org_id 'org-123', got '%s'", got.OrganizationID.ValueString())
	}
	if got.PrivateSpaceID.ValueString() != "ps-456" {
		t.Errorf("Expected private_space_id 'ps-456', got '%s'", got.PrivateSpaceID.ValueString())
	}
	if got.TransitGatewayID.ValueString() != "tgw-789" {
		t.Errorf("Expected transit_gateway_id 'tgw-789', got '%s'", got.TransitGatewayID.ValueString())
	}
	if got.CIDR.ValueString() != "10.0.0.0/8" {
		t.Errorf("Expected cidr '10.0.0.0/8', got '%s'", got.CIDR.ValueString())
	}
}

func TestTransitGatewayRouteResource_ImportState_InvalidID(t *testing.T) {
	res := NewTransitGatewayRouteResource().(*TransitGatewayRouteResource)

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	req := resource.ImportStateRequest{ID: "too/few/parts"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	res.ImportState(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ImportState() with too few parts should error")
	}
}
