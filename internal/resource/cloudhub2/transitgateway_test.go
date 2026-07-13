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

func TestNewTransitGatewayResource(t *testing.T) {
	r := NewTransitGatewayResource()

	if r == nil {
		t.Error("NewTransitGatewayResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestTransitGatewayResource_Metadata(t *testing.T) {
	r := NewTransitGatewayResource()
	testutil.TestResourceMetadata(t, r, "_transit_gateway")
}

func TestTransitGatewayResource_Schema(t *testing.T) {
	res := NewTransitGatewayResource()

	requiredAttrs := []string{"name", "resource_share_id", "resource_share_account", "routes", "private_space_id", "organization_id"}
	optionalAttrs := []string{}
	computedAttrs := []string{"id", "aws_transit_gateway_id", "status"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTransitGatewayResource_Configure(t *testing.T) {
	res := NewTransitGatewayResource().(*TransitGatewayResource)

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

func TestTransitGatewayResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTransitGatewayResource().(*TransitGatewayResource)

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

func TestTransitGatewayResource_Read(t *testing.T) {
	tgwPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-123"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":   "tgw-123",
				"name": "my-tgw",
				"spec": map[string]interface{}{
					"resourceShare": map[string]interface{}{
						"id":      "share-uuid-123",
						"account": "123456789012",
					},
					"region": "us-east-1",
				},
				"status": map[string]interface{}{
					"gateway":     "available",
					"attachment":  "available",
					"tgwResource": "tgw-0abc123def456",
					"routes":      []string{"10.0.0.0/8"},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayResource().(*TransitGatewayResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, "tgw-123"),
		"name":                   tftypes.NewValue(tftypes.String, "my-tgw"),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-0abc123def456"),
		"resource_share_id":      tftypes.NewValue(tftypes.String, "share-uuid-123"),
		"resource_share_account": tftypes.NewValue(tftypes.String, "123456789012"),
		"routes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
		}),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"status":           tftypes.NewValue(tftypes.String, "available"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "my-tgw" {
		t.Errorf("Expected Name 'my-tgw', got %s", got.Name.ValueString())
	}
	if got.AwsTransitGatewayID.ValueString() != "tgw-0abc123def456" {
		t.Errorf("Expected AwsTransitGatewayID 'tgw-0abc123def456', got %s", got.AwsTransitGatewayID.ValueString())
	}
	if got.Status.ValueString() != "available" {
		t.Errorf("Expected Status 'available', got %s", got.Status.ValueString())
	}
	if got.ResourceShareID.ValueString() != "share-uuid-123" {
		t.Errorf("Expected ResourceShareID 'share-uuid-123', got %s", got.ResourceShareID.ValueString())
	}
	if got.ResourceShareAccount.ValueString() != "123456789012" {
		t.Errorf("Expected ResourceShareAccount '123456789012', got %s", got.ResourceShareAccount.ValueString())
	}
}

func TestTransitGatewayResource_Read_NotFound(t *testing.T) {
	tgwPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-gone"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwPath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("not found"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayResource().(*TransitGatewayResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, "tgw-gone"),
		"name":                   tftypes.NewValue(tftypes.String, "my-tgw"),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, ""),
		"resource_share_id":      tftypes.NewValue(tftypes.String, "share-uuid-123"),
		"resource_share_account": tftypes.NewValue(tftypes.String, "123456789012"),
		"routes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
		}),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"status":           tftypes.NewValue(tftypes.String, "pending"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() should not have errors on 404, got: %v", resp.Diagnostics.Errors())
	}

	if !resp.State.Raw.IsNull() {
		t.Error("Read() should remove resource from state on 404")
	}
}

func TestTransitGatewayResource_ImportState(t *testing.T) {
	r := NewTransitGatewayResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTransitGatewayResource_ImportState_InvalidID(t *testing.T) {
	res := NewTransitGatewayResource().(*TransitGatewayResource)

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	req := resource.ImportStateRequest{ID: "only-one-part"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	res.ImportState(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ImportState() with invalid ID format should have errors")
	}
}

func TestTransitGatewayResource_ImportState_ValidID(t *testing.T) {
	res := NewTransitGatewayResource().(*TransitGatewayResource)

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	initialState := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, ""),
		"name":                   tftypes.NewValue(tftypes.String, ""),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, ""),
		"resource_share_id":      tftypes.NewValue(tftypes.String, ""),
		"resource_share_account": tftypes.NewValue(tftypes.String, ""),
		"routes":                 tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
		"private_space_id":       tftypes.NewValue(tftypes.String, ""),
		"organization_id":        tftypes.NewValue(tftypes.String, ""),
		"status":                 tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ImportStateRequest{ID: "org-123/ps-456/tgw-789"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: initialState},
	}

	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with valid ID should not have errors: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.OrganizationID.ValueString() != "org-123" {
		t.Errorf("Expected org_id 'org-123', got '%s'", got.OrganizationID.ValueString())
	}
	if got.PrivateSpaceID.ValueString() != "ps-456" {
		t.Errorf("Expected private_space_id 'ps-456', got '%s'", got.PrivateSpaceID.ValueString())
	}
	if got.ID.ValueString() != "tgw-789" {
		t.Errorf("Expected id 'tgw-789', got '%s'", got.ID.ValueString())
	}
}

func TestTransitGatewayResource_Create(t *testing.T) {
	tgwListPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwListPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				w.WriteHeader(http.StatusCreated)
				return
			}
			// GET - list TGWs (called after create to find the new one)
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{
					"id":   "new-tgw-id",
					"name": "test-tgw",
					"spec": map[string]interface{}{
						"resourceShare": map[string]interface{}{
							"id":      "share-uuid",
							"account": "123456789012",
						},
						"region": "us-east-1",
					},
					"status": map[string]interface{}{
						"gateway":     "pending",
						"attachment":  "pending",
						"tgwResource": "tgw-0abc",
						"routes":      []string{"10.0.0.0/8"},
					},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	res := NewTransitGatewayResource().(*TransitGatewayResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, nil),
		"name":                   tftypes.NewValue(tftypes.String, "test-tgw"),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, nil),
		"resource_share_id":      tftypes.NewValue(tftypes.String, "share-uuid"),
		"resource_share_account": tftypes.NewValue(tftypes.String, "123456789012"),
		"routes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
		}),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"status":           tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got TransitGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "new-tgw-id" {
		t.Errorf("Expected ID 'new-tgw-id', got '%s'", got.ID.ValueString())
	}
	if got.Status.ValueString() != "pending" {
		t.Errorf("Expected Status 'pending', got '%s'", got.Status.ValueString())
	}
}

func TestTransitGatewayResource_Delete(t *testing.T) {
	tgwPath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-del"
	deleteCalled := false

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "DELETE" {
				deleteCalled = true
				w.WriteHeader(http.StatusNoContent)
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
	res := NewTransitGatewayResource().(*TransitGatewayResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	stateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, "tgw-del"),
		"name":                   tftypes.NewValue(tftypes.String, "my-tgw"),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-0abc"),
		"resource_share_id":      tftypes.NewValue(tftypes.String, "share-uuid"),
		"resource_share_account": tftypes.NewValue(tftypes.String, "123456789012"),
		"routes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
		}),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"status":           tftypes.NewValue(tftypes.String, "available"),
	})

	req := resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	resp := &resource.DeleteResponse{}
	res.Delete(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() reported errors: %v", resp.Diagnostics.Errors())
	}
	if !deleteCalled {
		t.Error("Delete() should have called the API delete endpoint")
	}
}
