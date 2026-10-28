package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// newTestSelfManagedResource wires a resource to a client pointed at the mock server.
func newTestSelfManagedResource(baseURL string) *SelfManagedGatewayResource {
	res := NewSelfManagedGatewayResource().(*SelfManagedGatewayResource)
	res.client = &apimgmtclient.SelfManagedGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    baseURL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}
	return res
}

// smgStateType returns the tftypes state type for the resource schema.
func smgStateType(t *testing.T, res *SelfManagedGatewayResource) (resource.SchemaResponse, tftypes.Type) {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	return schemaResp, schemaResp.Schema.Type().TerraformType(ctx)
}

// smgPlanValues builds a full raw-state/plan value for the resource object type. Every
// attribute must be present (the framework requires the object to match the schema); pass
// the computed fields as unknown for a plan, or as concrete values for a prior state.
func smgPlanValues(stateType tftypes.Type, m map[string]tftypes.Value) tftypes.Value {
	return tftypes.NewValue(stateType, m)
}

// tokenPath / gatewaysPath / gatewayItemPath are the mock routes for org/env "test-*".
const (
	smgTokenPath    = "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gatewaytokens"
	smgGatewaysPath = "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways"
)

// --- Interface / Metadata / Schema / Configure ---

func TestNewSelfManagedGatewayResource(t *testing.T) {
	r := NewSelfManagedGatewayResource()
	if r == nil {
		t.Fatal("NewSelfManagedGatewayResource() returned nil")
	}
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("SelfManagedGatewayResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("SelfManagedGatewayResource should implement ResourceWithImportState")
	}
}

func TestSelfManagedGatewayResource_Metadata(t *testing.T) {
	r := NewSelfManagedGatewayResource()
	testutil.TestResourceMetadata(t, r, "_self_managed_gateway")
}

func TestSelfManagedGatewayResource_Schema(t *testing.T) {
	r := NewSelfManagedGatewayResource()
	requiredAttrs := []string{"name", "environment_id"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id", "organization_id", "registration_token", "gateway_id", "status", "last_update"}
	testutil.TestResourceSchema(t, r, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestSelfManagedGatewayResource_Configure(t *testing.T) {
	res := NewSelfManagedGatewayResource().(*SelfManagedGatewayResource)
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

func TestSelfManagedGatewayResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewSelfManagedGatewayResource().(*SelfManagedGatewayResource)
	ctx := context.Background()
	req := resource.ConfigureRequest{ProviderData: "invalid"}
	resp := &resource.ConfigureResponse{}
	res.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

// --- Create ---

// Create mints a token and, when the runtime has NOT yet self-registered (the normal
// state right after apply), keeps the resource with an empty gateway_id. The crux
// assertions: (1) registration_token is populated, (2) id == org/env/name, (3) the
// organization_id fell back to the provider-credentials org, and (4) the post-apply
// state is fully known — the framework rejects any leftover unknown Computed attribute.
func TestSelfManagedGatewayResource_Create_Unregistered(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgTokenPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				w.WriteHeader(http.StatusMethodNotAllowed)
				return
			}
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"registrationToken": "opaque-token-value",
			})
		},
		smgGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			// No runtime has registered yet.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content":       []map[string]interface{}{},
				"totalElements": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	// organization_id is unknown (Optional+Computed, omitted from config) so Create must
	// fall back to the provider-credentials org and resolve it to a known value.
	planRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"gateway_id":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"status":             tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"last_update":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: planRaw}}
	res.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SelfManagedGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.RegistrationToken.ValueString() != "opaque-token-value" {
		t.Errorf("registration_token = %q, want opaque-token-value", got.RegistrationToken.ValueString())
	}
	if got.OrganizationID.ValueString() != "test-org-id" {
		t.Errorf("organization_id = %q, want test-org-id (creds fallback)", got.OrganizationID.ValueString())
	}
	if got.ID.ValueString() != "test-org-id/test-env-id/test-flex-gw" {
		t.Errorf("id = %q, want test-org-id/test-env-id/test-flex-gw", got.ID.ValueString())
	}
	if got.GatewayID.ValueString() != "" {
		t.Errorf("gateway_id = %q, want empty (runtime not registered yet)", got.GatewayID.ValueString())
	}

	// Post-apply invariant: the Terraform framework rejects any state whose Computed
	// attributes are still unknown after apply. gateway_id/status/last_update must have been
	// resolved to known (empty) values by setEmptyIfUnknown.
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatal("state has unknown values after Create — framework would reject with " +
			"\"invalid result object after apply\"")
	}
}

// When a runtime with the same name has already registered, Create resolves and surfaces
// the platform gateway id / status / last_update alongside the freshly minted token.
func TestSelfManagedGatewayResource_Create_AlreadyRegistered(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgTokenPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"registrationToken": "opaque-token-value",
			})
		},
		smgGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					{"id": "gw-1", "name": "test-flex-gw", "status": "CONNECTED", "lastUpdate": "2026-07-21T14:29:07.69Z"},
				},
				"totalElements": 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	planRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"gateway_id":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"status":             tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"last_update":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: planRaw}}
	res.Create(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SelfManagedGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.GatewayID.ValueString() != "gw-1" {
		t.Errorf("gateway_id = %q, want gw-1", got.GatewayID.ValueString())
	}
	if got.Status.ValueString() != "CONNECTED" {
		t.Errorf("status = %q, want CONNECTED", got.Status.ValueString())
	}
	if got.LastUpdate.ValueString() != "2026-07-21T14:29:07.69Z" {
		t.Errorf("last_update = %q, want 2026-07-21T14:29:07.69Z", got.LastUpdate.ValueString())
	}
	if !resp.State.Raw.IsFullyKnown() {
		t.Fatal("state has unknown values after Create")
	}
}

// A failed token mint must fail Create (the token is the whole point of the resource).
func TestSelfManagedGatewayResource_Create_MintError(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgTokenPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "boom")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	planRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"gateway_id":         tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"status":             tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"last_update":        tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: planRaw}}
	res.Create(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Create() should error when the token mint fails")
	}
}

// --- Read ---

// Read resolves a now-registered gateway and surfaces its platform fields.
func TestSelfManagedGatewayResource_Read_Resolves(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					{"id": "gw-1", "name": "test-flex-gw", "status": "CONNECTED", "lastUpdate": "2026-07-21T15:00:00Z"},
				},
				"totalElements": 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	priorStateRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-env-id/test-flex-gw"),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, "opaque-token-value"),
		"gateway_id":         tftypes.NewValue(tftypes.String, ""),
		"status":             tftypes.NewValue(tftypes.String, ""),
		"last_update":        tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SelfManagedGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.GatewayID.ValueString() != "gw-1" {
		t.Errorf("gateway_id = %q, want gw-1", got.GatewayID.ValueString())
	}
	if got.Status.ValueString() != "CONNECTED" {
		t.Errorf("status = %q, want CONNECTED", got.Status.ValueString())
	}
	if got.LastUpdate.ValueString() != "2026-07-21T15:00:00Z" {
		t.Errorf("last_update = %q, want 2026-07-21T15:00:00Z", got.LastUpdate.ValueString())
	}
	// The minted token must survive a Read (it is not recoverable from the API).
	if got.RegistrationToken.ValueString() != "opaque-token-value" {
		t.Errorf("registration_token = %q, want it preserved across Read", got.RegistrationToken.ValueString())
	}
}

// Read of an unregistered gateway keeps the resource in state (does NOT remove it), so a
// re-mint is not forced on the next apply.
func TestSelfManagedGatewayResource_Read_UnregisteredStaysInState(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content":       []map[string]interface{}{},
				"totalElements": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	priorStateRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-env-id/test-flex-gw"),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, "opaque-token-value"),
		"gateway_id":         tftypes.NewValue(tftypes.String, ""),
		"status":             tftypes.NewValue(tftypes.String, ""),
		"last_update":        tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	if resp.State.Raw.IsNull() {
		t.Fatal("Read() must NOT remove an unregistered self-managed gateway from state")
	}
	var got SelfManagedGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.GatewayID.ValueString() != "" {
		t.Errorf("gateway_id = %q, want empty (still unregistered)", got.GatewayID.ValueString())
	}
	if got.RegistrationToken.ValueString() != "opaque-token-value" {
		t.Errorf("registration_token = %q, want it preserved", got.RegistrationToken.ValueString())
	}
}

// Bug B regression: a soft-deleted (DELETED) gateway is a tombstone that lingers in the list
// forever. Read must NOT re-bind gateway_id/status to it — a `terraform destroy` followed by
// a refresh would otherwise report spurious drift. The resolved fields must fall through to
// known+empty exactly as if the gateway had never registered.
func TestSelfManagedGatewayResource_Read_SkipsDeletedTombstone(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					// Same name, but soft-deleted — must be skipped.
					{"id": "gw-dead", "name": "test-flex-gw", "status": "DELETED", "lastUpdate": "2026-07-21T14:00:00Z"},
				},
				"totalElements": 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	// Prior state simulates a gateway that WAS bound to gw-dead before it was deleted.
	priorStateRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-env-id/test-flex-gw"),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, "opaque-token-value"),
		"gateway_id":         tftypes.NewValue(tftypes.String, "gw-dead"),
		"status":             tftypes.NewValue(tftypes.String, "CONNECTED"),
		"last_update":        tftypes.NewValue(tftypes.String, "2026-07-21T13:00:00Z"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SelfManagedGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.GatewayID.ValueString() != "" {
		t.Errorf("gateway_id = %q, want empty — a DELETED tombstone must not re-bind", got.GatewayID.ValueString())
	}
	if got.Status.ValueString() != "" {
		t.Errorf("status = %q, want empty — a DELETED tombstone must not re-bind", got.Status.ValueString())
	}
}

// Bug B regression (re-registration): when a tombstone AND a live gateway share the same
// name (the runtime was re-registered after a delete), Read must bind to the LIVE one
// regardless of the order the list returns them.
func TestSelfManagedGatewayResource_Read_PrefersLiveOverTombstone(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					// Tombstone listed first, live second — the live one must win.
					{"id": "gw-dead", "name": "test-flex-gw", "status": "DELETED", "lastUpdate": "2026-07-21T14:00:00Z"},
					{"id": "gw-live", "name": "test-flex-gw", "status": "CONNECTED", "lastUpdate": "2026-07-21T16:00:00Z"},
				},
				"totalElements": 2,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	priorStateRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-env-id/test-flex-gw"),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, "opaque-token-value"),
		"gateway_id":         tftypes.NewValue(tftypes.String, ""),
		"status":             tftypes.NewValue(tftypes.String, ""),
		"last_update":        tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SelfManagedGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.GatewayID.ValueString() != "gw-live" {
		t.Errorf("gateway_id = %q, want gw-live — must bind to the live gateway, not the tombstone", got.GatewayID.ValueString())
	}
	if got.Status.ValueString() != "CONNECTED" {
		t.Errorf("status = %q, want CONNECTED", got.Status.ValueString())
	}
}

// --- Delete ---

// When the runtime never registered, gateway_id is empty and Delete is a no-op: it must
// NOT issue a platform DELETE (there is no object to delete).
func TestSelfManagedGatewayResource_Delete_NoOpWhenUnregistered(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			// Any call to the gateways endpoint during delete is a bug for the no-op case.
			t.Errorf("Delete must NOT call the gateways endpoint when gateway_id is empty; got %s %s", r.Method, r.URL.Path)
			w.WriteHeader(http.StatusInternalServerError)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	stateRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-env-id/test-flex-gw"),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, "opaque-token-value"),
		"gateway_id":         tftypes.NewValue(tftypes.String, ""),
		"status":             tftypes.NewValue(tftypes.String, ""),
		"last_update":        tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	res.Delete(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() (no-op) reported errors: %v", resp.Diagnostics.Errors())
	}
}

// When the gateway has registered (gateway_id set), Delete issues the platform DELETE. The
// real API answers the soft-delete with HTTP 202 Accepted.
func TestSelfManagedGatewayResource_Delete_CallsDeleteWhenRegistered(t *testing.T) {
	deletePath := smgGatewaysPath + "/gw-1"
	var gotMethod string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		deletePath: func(w http.ResponseWriter, r *http.Request) {
			gotMethod = r.Method
			w.WriteHeader(http.StatusAccepted)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestSelfManagedResource(server.URL)

	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, res)

	stateRaw := smgPlanValues(stateType, map[string]tftypes.Value{
		"id":                 tftypes.NewValue(tftypes.String, "test-org-id/test-env-id/test-flex-gw"),
		"name":               tftypes.NewValue(tftypes.String, "test-flex-gw"),
		"organization_id":    tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":     tftypes.NewValue(tftypes.String, "test-env-id"),
		"registration_token": tftypes.NewValue(tftypes.String, "opaque-token-value"),
		"gateway_id":         tftypes.NewValue(tftypes.String, "gw-1"),
		"status":             tftypes.NewValue(tftypes.String, "CONNECTED"),
		"last_update":        tftypes.NewValue(tftypes.String, "2026-07-21T15:00:00Z"),
	})

	req := resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	resp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	res.Delete(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() reported errors: %v", resp.Diagnostics.Errors())
	}
	if gotMethod != http.MethodDelete {
		t.Errorf("Delete() issued method %q, want DELETE", gotMethod)
	}
}

// --- ImportState ---

func TestSelfManagedGatewayResource_ImportState(t *testing.T) {
	r := NewSelfManagedGatewayResource().(*SelfManagedGatewayResource)
	ctx := context.Background()
	schemaResp, stateType := smgStateType(t, r)
	nullRaw := nullSMGState(stateType)

	t.Run("2-part env/name falls back to creds org", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "env-2/my-gw"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: nullRaw}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got SelfManagedGatewayResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("environment_id = %q, want env-2", got.EnvironmentID.ValueString())
		}
		if got.Name.ValueString() != "my-gw" {
			t.Errorf("name = %q, want my-gw", got.Name.ValueString())
		}
	})

	t.Run("3-part org/env/name sets all three", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1/env-2/my-gw"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: nullRaw}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got SelfManagedGatewayResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("organization_id = %q, want org-1", got.OrganizationID.ValueString())
		}
		if got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("environment_id = %q, want env-2", got.EnvironmentID.ValueString())
		}
		if got.Name.ValueString() != "my-gw" {
			t.Errorf("name = %q, want my-gw", got.Name.ValueString())
		}
	})

	t.Run("1-part ID errors", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "just-a-name"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: nullRaw}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for a 1-part ID")
		}
	})

	t.Run("4-part ID errors", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "a/b/c/d"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: nullRaw}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for a 4-part ID")
		}
	})
}

// nullSMGState builds an all-null state value for the resource's object type.
func nullSMGState(stateType tftypes.Type) tftypes.Value {
	obj := stateType.(tftypes.Object)
	vals := make(map[string]tftypes.Value, len(obj.AttributeTypes))
	for k, at := range obj.AttributeTypes {
		vals[k] = tftypes.NewValue(at, nil)
	}
	return tftypes.NewValue(obj, vals)
}
