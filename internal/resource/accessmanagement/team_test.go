package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestTeamResource_ManagedSignal_NoUseStateForUnknown guards the same invariant as the role
// test: permissions and members must NOT carry UseStateForUnknown. Team Update uses
// plan.X.IsUnknown() as the "is this attribute config-managed?" signal (managePermissions :=
// !plan.Permissions.IsNull() && !plan.Permissions.IsUnknown()), so config-omit -> unknown ->
// reconcile read-only from the API. UseStateForUnknown would make omit -> known ->
// applyTeamPermissions enforces the last-applied set, reverting out-of-band permission/member
// changes and breaking the documented "Omit the attribute entirely to leave ... unmanaged"
// contract.
func TestTeamResource_ManagedSignal_NoUseStateForUnknown(t *testing.T) {
	res := NewTeamResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	attrs := schemaResp.Schema.Attributes

	for _, name := range []string{"permissions", "members"} {
		a, ok := attrs[name].(schema.SetNestedAttribute)
		if !ok {
			t.Fatalf("%s: expected SetNestedAttribute, got %T", name, attrs[name])
		}
		if len(a.PlanModifiers) != 0 {
			t.Errorf("%s: expected NO plan modifiers (IsUnknown is the managed-signal; "+
				"UseStateForUnknown would flip unmanaged->managed and revert out-of-band changes), got %d",
				name, len(a.PlanModifiers))
		}
	}
}

// teamStateType returns the resource's Terraform type for building raw plan/state.
func teamStateType(t *testing.T, res *TeamResource) tftypes.Type {
	t.Helper()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(context.Background(), resource.SchemaRequest{}, schemaResp)
	return schemaResp.Schema.Type().TerraformType(context.Background())
}

// teamRawValue builds a raw team object with null roles/members (so Update's
// role/member reconcile is skipped) and the given id/name/type/parent.
// parentID sets parent_team_id (null if empty string); parentUnknown makes it unknown.
func teamRawValue(stateType tftypes.Type, id, name, teamType, _ string, parentUnknown bool, parentID string) tftypes.Value {
	roleObj := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	memberObj := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"username":        tftypes.String,
		"membership_type": tftypes.String,
	}}
	var parentTeamVal tftypes.Value
	if parentUnknown {
		parentTeamVal = tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	} else if parentID == "" {
		parentTeamVal = tftypes.NewValue(tftypes.String, nil)
	} else {
		parentTeamVal = tftypes.NewValue(tftypes.String, parentID)
	}
	return tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, id),
		"name":            tftypes.NewValue(tftypes.String, name),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"team_type":       tftypes.NewValue(tftypes.String, teamType),
		"parent_team_id":  parentTeamVal,
		"permissions":     tftypes.NewValue(tftypes.Set{ElementType: roleObj}, nil),
		"members":         tftypes.NewValue(tftypes.Set{ElementType: memberObj}, nil),
		"created_at":      tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
		"updated_at":      tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
	})
}

// TestTeamResource_Update_UnknownParentDoesNotMove reproduces the "400 failed to
// validate against RAML" bug on the UPDATE path: parent_team_id was Optional+Computed
// WITHOUT UseStateForUnknown, so on any later plan the already-resolved parent went
// "(known after apply)". Update then saw plan.parent (unknown) != state.parent and
// fired UpdateTeamParent with an EMPTY parent (unknown.ValueString() == ""), which the
// platform rejects with a 400. The guard in Update must NOT move the team when the
// planned parent is unknown. This test fails (unknown -> empty PUT) without the guard.
func TestTeamResource_Update_UnknownParentDoesNotMove(t *testing.T) {
	teamID := "team-1"
	basePath := "/accounts/api/organizations/test-org-id/teams/" + teamID
	parentCalled := false
	var sentParent string

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath + "/parent": func(w http.ResponseWriter, r *http.Request) {
			parentCalled = true
			var body accessmanagement.UpdateTeamParentRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			sentParent = body.ParentTeamID
			// Mimic the platform: an empty parent_team_id is a 400 RAML failure.
			if body.ParentTeamID == "" {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Request failed to validate against RAML definition")
				return
			}
			w.WriteHeader(http.StatusOK)
		},
		// ListTeams — resolveTeamIDToName needs this after mapTeamToState.
		"/accounts/api/organizations/test-org-id/teams": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{"team_id": "root-team-id", "team_name": "Root Team", "org_id": "test-org-id", "ancestor_team_ids": []string{}},
				},
				"total": 1,
			})
		},
		basePath: func(w http.ResponseWriter, r *http.Request) {
			// GetTeam readback (Update reads the team when name/type are unchanged).
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id":           teamID,
				"team_name":         "madhav-manual-test-team",
				"team_type":         "internal",
				"org_id":            "test-org-id",
				"ancestor_team_ids": []string{"root-team-id"},
				"created_at":        "2024-01-01T00:00:00Z",
				"updated_at":        "2024-01-02T00:00:00Z",
			})
		},
		// Reconcile endpoints for unmanaged roles/members (null in plan → !manageRoles/!manageMembers).
		basePath + "/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		"/accounts/api/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		basePath + "/members": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		"/accounts/api/organizations/test-org-id/users": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestTeamResource(server.URL)

	ctx := context.Background()
	stateType := teamStateType(t, res)
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	// State: parent_team_id already resolved to the root. Plan: parent UNKNOWN (the bug trigger).
	stateRaw := teamRawValue(stateType, teamID, "madhav-manual-test-team", "internal", "root-team-id", false, "root-team-id")
	planRaw := teamRawValue(stateType, teamID, "madhav-manual-test-team", "internal", "", true, "")

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	res.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() reported errors (bug: unknown parent triggered an empty-parent move): %v", resp.Diagnostics.Errors())
	}
	if parentCalled {
		t.Errorf("UpdateTeamParent was called with %q; it must NOT be called when the planned parent is unknown", sentParent)
	}
}

// TestTeamResource_Update_RealParentChangeMoves confirms the guard doesn't over-block:
// a genuine parent change (known, non-empty, different ID) still moves the team.
func TestTeamResource_Update_RealParentChangeMoves(t *testing.T) {
	teamID := "team-1"
	basePath := "/accounts/api/organizations/test-org-id/teams/" + teamID
	parentCalled := false
	var sentParent string

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath + "/parent": func(w http.ResponseWriter, r *http.Request) {
			parentCalled = true
			var body accessmanagement.UpdateTeamParentRequest
			_ = json.NewDecoder(r.Body).Decode(&body)
			sentParent = body.ParentTeamID
			w.WriteHeader(http.StatusOK)
		},
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id":           teamID,
				"team_name":         "madhav-manual-test-team",
				"team_type":         "internal",
				"org_id":            "test-org-id",
				"ancestor_team_ids": []string{"new-parent-id"},
				"created_at":        "2024-01-01T00:00:00Z",
				"updated_at":        "2024-01-02T00:00:00Z",
			})
		},
		// Reconcile endpoints for unmanaged roles/members.
		basePath + "/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		"/accounts/api/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		basePath + "/members": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		"/accounts/api/organizations/test-org-id/users": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestTeamResource(server.URL)

	ctx := context.Background()
	stateType := teamStateType(t, res)
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	// State: current parent is root-team-id. Plan: user changed parent_team_id to "new-parent-id".
	stateRaw := teamRawValue(stateType, teamID, "madhav-manual-test-team", "internal", "root-team-id", false, "root-team-id")
	planRaw := teamRawValue(stateType, teamID, "madhav-manual-test-team", "internal", "root-team-id", false, "new-parent-id")

	req := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	resp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw}}
	res.Update(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Update() reported errors on a real parent change: %v", resp.Diagnostics.Errors())
	}
	if !parentCalled {
		t.Error("UpdateTeamParent should have been called for a genuine parent change")
	}
	if sentParent != "new-parent-id" {
		t.Errorf("UpdateTeamParent sent parent %q, want %q", sentParent, "new-parent-id")
	}
}

func newTestTeamResource(serverURL string) *TeamResource {
	mockClient := &client.AnypointClient{
		BaseURL:    serverURL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
		Cache:      client.NewResponseCache(),
	}
	res := NewTeamResource().(*TeamResource)
	res.client = &accessmanagement.TeamClient{AnypointClient: mockClient}
	res.rolesClient = &accessmanagement.TeamRolesClient{AnypointClient: mockClient}
	res.membersClient = &accessmanagement.TeamMembersClient{AnypointClient: mockClient}
	res.usersClient = &accessmanagement.RoleUsersClient{AnypointClient: mockClient}
	res.catalogClient = &accessmanagement.RolePermissionClient{AnypointClient: mockClient}
	return res
}

// TestTeamResource_resolveRootTeamID proves that when parent_team_id is omitted,
// the provider can find the org's root team (the one with no ancestors) to use as
// the required parent — the fix for the "400 failed to validate against RAML"
// error that a missing parent_team_id caused.
func TestTeamResource_resolveRootTeamID(t *testing.T) {
	listPath := "/accounts/api/organizations/test-org-id/teams"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{"team_id": "child-1", "team_name": "Child A", "ancestor_team_ids": []string{"root-team-id"}},
					{"team_id": "root-team-id", "team_name": "Everyone", "ancestor_team_ids": []string{}},
					{"team_id": "child-2", "team_name": "Child B", "ancestor_team_ids": []string{"root-team-id", "child-1"}},
				},
				"total": 3,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestTeamResource(server.URL)

	got, err := res.resolveRootTeamID(context.Background(), "test-org-id")
	if err != nil {
		t.Fatalf("resolveRootTeamID() unexpected error: %v", err)
	}
	if got != "root-team-id" {
		t.Errorf("resolveRootTeamID() = %q, want %q (the ancestor-less team)", got, "root-team-id")
	}
}

// TestTeamResource_resolveRootTeamID_NoRoot ensures we return a clear error (rather
// than silently sending an empty parent) if no ancestor-less team exists.
func TestTeamResource_resolveRootTeamID_NoRoot(t *testing.T) {
	listPath := "/accounts/api/organizations/test-org-id/teams"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{"team_id": "child-1", "team_name": "Child A", "ancestor_team_ids": []string{"some-parent"}},
				},
				"total": 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	res := newTestTeamResource(server.URL)

	if _, err := res.resolveRootTeamID(context.Background(), "test-org-id"); err == nil {
		t.Error("resolveRootTeamID() expected an error when no root team exists, got nil")
	}
}

func TestNewTeamResource(t *testing.T) {
	r := NewTeamResource()

	if r == nil {
		t.Error("NewTeamResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("TeamResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("TeamResource should implement ResourceWithImportState")
	}
}

func TestTeamResource_Metadata(t *testing.T) {
	r := NewTeamResource()
	testutil.TestResourceMetadata(t, r, "_team")
}

func TestTeamResource_Schema(t *testing.T) {
	res := NewTeamResource()

	requiredAttrs := []string{"name"}
	optionalAttrs := []string{"team_type", "organization_id", "parent_team_id"}
	computedAttrs := []string{"id", "created_at", "updated_at"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTeamResource_Configure(t *testing.T) {
	res := NewTeamResource().(*TeamResource)

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

func TestTeamResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTeamResource().(*TeamResource)

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

func TestTeamResource_ImportState(t *testing.T) {
	r := NewTeamResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTeamResourceModel_Validation(t *testing.T) {
	model := TeamResourceModel{}
	_ = model.ID
}

func TestTeamResource_Read(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/teams/test-team-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id":    "test-team-id",
				"team_name":  "My Team",
				"team_type":  "internal",
				"org_id":     "test-org-id",
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
			})
		},
		basePath + "/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		"/accounts/api/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		basePath + "/members": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
		"/accounts/api/organizations/test-org-id/users": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []interface{}{},
				"total": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	mockClient := &client.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
		Cache:      client.NewResponseCache(),
	}
	res := NewTeamResource().(*TeamResource)
	res.client = &accessmanagement.TeamClient{AnypointClient: mockClient}
	res.rolesClient = &accessmanagement.TeamRolesClient{AnypointClient: mockClient}
	res.membersClient = &accessmanagement.TeamMembersClient{AnypointClient: mockClient}
	res.usersClient = &accessmanagement.RoleUsersClient{AnypointClient: mockClient}
	res.catalogClient = &accessmanagement.RolePermissionClient{AnypointClient: mockClient}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-team-id"),
		"name":            tftypes.NewValue(tftypes.String, "My Team"),
		"parent_team_id":  tftypes.NewValue(tftypes.String, nil),
		"team_type":       tftypes.NewValue(tftypes.String, "internal"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"created_at":      tftypes.NewValue(tftypes.String, ""),
		"updated_at":      tftypes.NewValue(tftypes.String, ""),
		// roles/members null — Read populates from API (empty sets returned by mock).
		"permissions": tftypes.NewValue(tftypes.Set{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"name":           tftypes.String,
			"context_params": tftypes.Map{ElementType: tftypes.String},
		}}}, nil),
		"members": tftypes.NewValue(tftypes.Set{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"username":        tftypes.String,
			"membership_type": tftypes.String,
		}}}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got TeamResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "My Team" {
		t.Errorf("Expected Name 'My Team', got %s", got.Name.ValueString())
	}
}

func TestTeamResource_Read_NotFound(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/teams/test-team-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTeamResource().(*TeamResource)
	res.client = &accessmanagement.TeamClient{
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-team-id"),
		"name":            tftypes.NewValue(tftypes.String, "My Team"),
		"parent_team_id":  tftypes.NewValue(tftypes.String, nil),
		"team_type":       tftypes.NewValue(tftypes.String, "internal"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"created_at":      tftypes.NewValue(tftypes.String, ""),
		"updated_at":      tftypes.NewValue(tftypes.String, ""),
		// roles/members null — Read populates from API (empty sets returned by mock).
		"permissions": tftypes.NewValue(tftypes.Set{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"name":           tftypes.String,
			"context_params": tftypes.Map{ElementType: tftypes.String},
		}}}, nil),
		"members": tftypes.NewValue(tftypes.Set{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"username":        tftypes.String,
			"membership_type": tftypes.String,
		}}}, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

func BenchmarkTeamResource_Schema(b *testing.B) {
	res := NewTeamResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
