package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestConnectedAppResource_Create_PersistsStateWhenScopesFail is the regression for the
// CA-01 orphan bug: POST creates the app, then PUT /scopes fails (e.g. API 400). Before
// the fix, Create returned the error without writing state → live client credentials left
// unmanaged, and every re-apply POSTed another duplicate. Now we persist partial state
// (mirroring anypoint_role / anypoint_team). Terraform still taints on Create error, so a
// bare re-apply replaces; untaint+apply updates scopes on the same id.
func TestConnectedAppResource_Create_PersistsStateWhenScopesFail(t *testing.T) {
	const (
		orgID    = "test-org-id"
		clientID = "created-client-id"
		secret   = "created-client-secret"
	)

	var postCount, putCount int
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/" + orgID + "/connectedApplications": func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "unexpected method "+r.Method)
				return
			}
			postCount++
			testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
				"client_id":      clientID,
				"client_secret":  secret,
				"client_name":    "rf-p2-connected-app",
				"org_id":         orgID,
				"owner_user_id":  "owner-1",
				"grant_types":    []string{"client_credentials"},
				"redirect_uris":  []string{},
				"public_keys":    []string{},
				"scopes":         []string{"profile"},
				"audience":       "internal",
				"client_uri":     "",
				"enabled":        true,
				"created_at":     "2026-07-29T00:00:00Z",
				"updated_at":     "2026-07-29T00:00:00Z",
			})
		},
		"/accounts/api/connectedApplications/" + clientID + "/scopes": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPut:
				putCount++
				testutil.ErrorResponse(w, http.StatusBadRequest,
					"Request failed to validate against RAML definition: 1 != 0")
			case http.MethodGet:
				// Best-effort refresh after the failed PUT — fresh app has only profile.
				testutil.JSONResponse(w, http.StatusOK, accessmanagement.ConnectedAppScopes{
					Scopes: []accessmanagement.Scope{
						{Scope: "profile", ContextParams: map[string]interface{}{}},
					},
					Total: 1,
				})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "unexpected method "+r.Method)
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ac := &client.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      orgID,
	}
	res := &ConnectedAppResource{
		client:       &accessmanagement.ConnectedAppClient{AnypointClient: ac},
		scopesClient: &accessmanagement.ConnectedAppScopesClient{AnypointClient: ac},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() errors: %v", schemaResp.Diagnostics.Errors())
	}
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)

	scopeElemType := objType.AttributeTypes["scopes"].(tftypes.Set).ElementType
	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"name":            tftypes.NewValue(tftypes.String, "rf-p2-connected-app"),
		"client_secret":   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"grant_types":     tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{tftypes.NewValue(tftypes.String, "client_credentials")}),
		"redirect_uris":   tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, tftypes.UnknownValue),
		"public_keys":     tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, tftypes.UnknownValue),
		"audience":        tftypes.NewValue(tftypes.String, "internal"),
		"client_uri":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"enabled":         tftypes.NewValue(tftypes.Bool, true),
		"organization_id": tftypes.NewValue(tftypes.String, orgID),
		"owner_user_id":   tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"created_at":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"updated_at":      tftypes.NewValue(tftypes.String, tftypes.UnknownValue),
		"scopes": tftypes.NewValue(tftypes.Set{ElementType: scopeElemType}, []tftypes.Value{
			tftypes.NewValue(scopeElemType, map[string]tftypes.Value{
				"scope": tftypes.NewValue(tftypes.String, "read:organization"),
				// org present so plan/create validation passes; PUT still returns 400 to
				// exercise the partial-create persist path (API failure, not config).
				"context_params": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
					"org": tftypes.NewValue(tftypes.String, orgID),
				}),
			}),
		}),
	})

	req := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
	resp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)}}
	res.Create(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Create() expected scopes error, got none")
	}
	if postCount != 1 {
		t.Errorf("expected 1 POST (app create), got %d", postCount)
	}
	if putCount != 1 {
		t.Errorf("expected 1 PUT (scopes), got %d", putCount)
	}

	// The bug: state was empty / null after the scopes failure. Assert the app is managed.
	if resp.State.Raw.IsNull() {
		t.Fatal("state is null after scopes failure — app would be orphaned (the CA-01 bug)")
	}
	var got ConnectedAppResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != clientID {
		t.Errorf("state.id = %q, want %q (app must be in state so destroy/re-apply can manage it)", got.ID.ValueString(), clientID)
	}
	if got.ClientSecret.ValueString() != secret {
		t.Errorf("state.client_secret = %q, want %q (secret is only returned at create)", got.ClientSecret.ValueString(), secret)
	}

	// Best-effort refresh should drop desired scopes and reflect live (profile-only → empty set).
	if !got.Scopes.IsNull() && !got.Scopes.IsUnknown() && len(got.Scopes.Elements()) != 0 {
		b, _ := json.Marshal(got.Scopes.Elements())
		t.Errorf("state.scopes should reflect live empty set after failed apply, got %s", b)
	}
}
