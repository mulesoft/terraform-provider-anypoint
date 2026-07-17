package cloudhub2

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestTransitGatewayResource_Status_NoUseStateForUnknown guards a deliberate decision: the
// computed `status` attribute must NOT carry UseStateForUnknown. status reflects the LIVE
// gateway attachment state ("Pending", "Available", ...), and Update unconditionally re-reads
// it from the API (plan.Status = types.StringValue(tgw.Status.Gateway)). UseStateForUnknown
// would freeze the plan to the prior status, but apply would write whatever the API returns —
// if the live state changed (e.g. Pending -> Available), that is a different value, so
// Terraform would fail with "Provider produced inconsistent result after apply". Showing
// "(known after apply)" for a genuinely-changing live-state field is correct. (Same class as
// exchange's server-bumped updated_date.)
func TestTransitGatewayResource_Status_NoUseStateForUnknown(t *testing.T) {
	res := NewTransitGatewayResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	attrs := schemaResp.Schema.Attributes

	a, ok := attrs["status"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("status: expected StringAttribute, got %T", attrs["status"])
	}
	if len(a.PlanModifiers) != 0 {
		t.Errorf("status: expected NO plan modifiers (live-state field re-read every Update; "+
			"UseStateForUnknown would cause inconsistent-result at apply), got %d", len(a.PlanModifiers))
	}
}

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
	testutil.TestResourceMetadata(t, r, "_transit_gateway_connection")
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
	tgwPath := "/runtimefabric/api/organizations/org-123/privatespaces/ps-456/transitgateways/tgw-789"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		tgwPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":   "tgw-789",
				"name": "test-tgw",
				"spec": map[string]interface{}{
					"resourceShare": map[string]interface{}{
						"id":      "share-uuid",
						"account": "123456789012",
					},
				},
				"status": map[string]interface{}{
					"gateway":     "available",
					"attachment":  "available",
					"tgwResource": "tgw-0abc",
					"routes":      []string{"10.0.0.0/8", "172.16.0.0/12"},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "org-123",
	}
	res := NewTransitGatewayResource().(*TransitGatewayResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

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

	// Verify routes were seeded from API
	var routes []string
	if diags := got.Routes.ElementsAs(ctx, &routes, false); diags.HasError() {
		t.Fatalf("Routes.ElementsAs errors: %v", diags.Errors())
	}
	if len(routes) != 2 {
		t.Errorf("Expected 2 routes from API, got %d", len(routes))
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

// TestTransitGatewayResource_Update_NameAndRoutes verifies that Update renames via
// the ORG-scoped endpoint (name-only body) AND replaces routes via the PRIVATE-SPACE
// connection PATCH with a {name,routes} object body. These are two DIFFERENT
// endpoints, both confirmed live 2026-07-17:
//   - rename → org-scoped .../transitgateways/{id}, name-only (the private-space
//     connection PATCH silently ignores the name);
//   - routes → private-space connection .../transitgateways/{id}, {name,routes}
//     object (the RAML's /routes sub-resource 404s — it does not exist).
//
// The /routes sub-resource must NEVER be hit.
func TestTransitGatewayResource_Update_NameAndRoutes(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-upd"
	routesPath := basePath + "/routes"
	orgScopedPath := "/runtimefabric/api/organizations/test-org-id/transitgateways/tgw-upd"

	var namePatched bool
	var routesBody []interface{}
	var routesPatched bool

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		// The dedicated /routes sub-resource does not exist — hitting it is a bug.
		routesPath: func(w http.ResponseWriter, r *http.Request) {
			t.Errorf("routes update must NOT hit the non-existent /routes sub-resource %s", routesPath)
			testutil.ErrorResponse(w, http.StatusNotFound, "no such endpoint")
		},
		// Org-scoped rename: name-only body, must NOT include routes.
		orgScopedPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, http.MethodPatch, orgScopedPath)
			namePatched = true
			raw, _ := io.ReadAll(r.Body)
			if strings.Contains(string(raw), "routes") {
				t.Errorf("org-scoped rename body must be name-only, must not mention routes, got %q", string(raw))
			}
			var body map[string]interface{}
			if err := json.Unmarshal(raw, &body); err != nil {
				t.Fatalf("failed to decode rename body: %v", err)
			}
			if body["name"] != "renamed-tgw" {
				t.Errorf("expected rename to 'renamed-tgw', got %v", body["name"])
			}
			// Live shape: the org-scoped PATCH returns a JSON array.
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{
					"id":   "tgw-upd",
					"name": body["name"],
					"status": map[string]interface{}{
						"gateway": "available", "attachment": "available", "tgwResource": "tgw-0abc",
					},
				},
			})
		},
		// Private-space connection: receives BOTH the routes PATCH ({name,routes}
		// object body) AND the step-3 GET re-read. It must NOT be used for the
		// rename (that goes to the org-scoped endpoint), so a name-only PATCH here
		// with no routes key would be wrong; we assert the routes key is present.
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPatch {
				routesPatched = true
				raw, _ := io.ReadAll(r.Body)
				var body struct {
					Name   string        `json:"name"`
					Routes []interface{} `json:"routes"`
				}
				if err := json.Unmarshal(raw, &body); err != nil {
					t.Errorf("routes body must be a {name,routes} object, got %q (err: %v)", string(raw), err)
				}
				if !strings.Contains(string(raw), "\"routes\"") {
					t.Errorf("routes PATCH body must carry a routes key, got %q", string(raw))
				}
				routesBody = body.Routes
				w.WriteHeader(http.StatusOK)
				return
			}
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":   "tgw-upd",
				"name": "renamed-tgw",
				"spec": map[string]interface{}{
					"resourceShare": map[string]interface{}{"id": "share-uuid", "account": "123456789012"},
				},
				"status": map[string]interface{}{
					"gateway": "available", "attachment": "available", "tgwResource": "tgw-0abc",
					"routes": []string{"10.0.0.0/8", "192.168.0.0/16"},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id",
	}
	res := NewTransitGatewayResource().(*TransitGatewayResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	// Prior state: name "old-tgw", routes ["10.0.0.0/8"].
	stateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, "tgw-upd"),
		"name":                   tftypes.NewValue(tftypes.String, "old-tgw"),
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

	// Plan: renamed + a second route added.
	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                     tftypes.NewValue(tftypes.String, "tgw-upd"),
		"name":                   tftypes.NewValue(tftypes.String, "renamed-tgw"),
		"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-0abc"),
		"resource_share_id":      tftypes.NewValue(tftypes.String, "share-uuid"),
		"resource_share_account": tftypes.NewValue(tftypes.String, "123456789012"),
		"routes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
			tftypes.NewValue(tftypes.String, "10.0.0.0/8"),
			tftypes.NewValue(tftypes.String, "192.168.0.0/16"),
		}),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
		"status":           tftypes.NewValue(tftypes.String, "available"),
	})

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	res.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() reported errors: %v", resp.Diagnostics.Errors())
	}
	if !namePatched {
		t.Error("Update() should PATCH the connection name when it changed")
	}
	if !routesPatched {
		t.Error("Update() should PATCH the /routes sub-resource when routes changed")
	}
	if len(routesBody) != 2 {
		t.Errorf("expected 2 routes in the bare-array body, got %d: %v", len(routesBody), routesBody)
	}

	var got TransitGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "renamed-tgw" {
		t.Errorf("Expected Name 'renamed-tgw', got %s", got.Name.ValueString())
	}
	var gotRoutes []string
	if diags := got.Routes.ElementsAs(ctx, &gotRoutes, false); diags.HasError() {
		t.Fatalf("Routes.ElementsAs errors: %v", diags.Errors())
	}
	if len(gotRoutes) != 2 {
		t.Errorf("Expected 2 routes in final state, got %d", len(gotRoutes))
	}
}

// TestTransitGatewayResource_Update_RoutesUnchanged verifies that when neither the
// name nor the route set changes (order-only differences don't count), Update makes
// NO write calls — it only re-reads the connection for computed fields.
func TestTransitGatewayResource_Update_RoutesUnchanged(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/transitgateways/tgw-noop"
	routesPath := basePath + "/routes"

	var wrote bool
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		routesPath: func(w http.ResponseWriter, r *http.Request) {
			wrote = true
			w.WriteHeader(http.StatusOK)
		},
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodPatch {
				wrote = true
			}
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":   "tgw-noop",
				"name": "same-tgw",
				"spec": map[string]interface{}{
					"resourceShare": map[string]interface{}{"id": "share-uuid", "account": "123456789012"},
				},
				"status": map[string]interface{}{
					"gateway": "available", "attachment": "available", "tgwResource": "tgw-0abc",
					"routes": []string{"10.0.0.0/8", "172.16.0.0/12"},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &anypointclient.AnypointClient{
		BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id",
	}
	res := NewTransitGatewayResource().(*TransitGatewayResource)
	res.client = &ch2client.TransitGatewayClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	// State and plan differ ONLY in route ordering — must be treated as no change.
	mk := func(r1, r2 string) tftypes.Value {
		return tftypes.NewValue(stateType, map[string]tftypes.Value{
			"id":                     tftypes.NewValue(tftypes.String, "tgw-noop"),
			"name":                   tftypes.NewValue(tftypes.String, "same-tgw"),
			"aws_transit_gateway_id": tftypes.NewValue(tftypes.String, "tgw-0abc"),
			"resource_share_id":      tftypes.NewValue(tftypes.String, "share-uuid"),
			"resource_share_account": tftypes.NewValue(tftypes.String, "123456789012"),
			"routes": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{
				tftypes.NewValue(tftypes.String, r1),
				tftypes.NewValue(tftypes.String, r2),
			}),
			"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
			"organization_id":  tftypes.NewValue(tftypes.String, "test-org-id"),
			"status":           tftypes.NewValue(tftypes.String, "available"),
		})
	}
	stateRaw := mk("10.0.0.0/8", "172.16.0.0/12")
	planRaw := mk("172.16.0.0/12", "10.0.0.0/8") // reordered

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	res.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() reported errors: %v", resp.Diagnostics.Errors())
	}
	if wrote {
		t.Error("Update() must NOT issue any write when only route ordering changed")
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
