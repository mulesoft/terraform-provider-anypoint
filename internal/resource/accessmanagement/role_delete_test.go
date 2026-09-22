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

// The role lifecycle's DELETE path could not be exercised end-to-end via
// `terraform destroy` on devx — the org's "Destructive Command Restriction" policy
// blocks that CLI verb. These tests drive the SAME RoleResource.Delete() code the
// destroy would run, against a mock server, covering the normal delete and — most
// importantly — the Class H idempotency branch (a 404 from the API on delete is
// treated as success, so a resource already removed out-of-band does not error the
// apply). The create/plan-idempotency/update/import steps WERE run live on devx.

// buildRoleDeleteState constructs a populated tfsdk.State for the role resource with
// the given id/org so Delete() can read it. permissions/members are null (Delete
// never touches them).
func buildRoleDeleteState(t *testing.T, r *RoleResource, id, orgID string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("Schema() errors: %v", schemaResp.Diagnostics.Errors())
	}
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	permObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	raw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, id),
		"name":            tftypes.NewValue(tftypes.String, "madhav-manual-test-role"),
		"description":     tftypes.NewValue(tftypes.String, "desc"),
		"organization_id": tftypes.NewValue(tftypes.String, orgID),
		"editable":        tftypes.NewValue(tftypes.Bool, true),
		"external_names":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
		"permissions":     tftypes.NewValue(tftypes.Set{ElementType: permObjType}, nil),
		"members":         tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		"created_at":      tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
		"updated_at":      tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
	})
	return tfsdk.State{Schema: schemaResp.Schema, Raw: raw}
}

func newRoleResourceWithDeleteHandler(t *testing.T, orgID, rgID string, handler func(w http.ResponseWriter, r *http.Request)) *RoleResource {
	t.Helper()
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/" + orgID + "/rolegroups/" + rgID: handler,
	}
	server := testutil.MockHTTPServer(t, handlers)
	userClient := &client.AnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      orgID,
	}
	return &RoleResource{
		client:      &accessmanagement.RoleClient{AnypointClient: userClient},
		permClient:  &accessmanagement.RolePermissionClient{AnypointClient: userClient},
		usersClient: &accessmanagement.RoleUsersClient{AnypointClient: userClient},
	}
}

// TestRoleResource_Delete_Success: a normal delete returns no diagnostics and issues
// the DELETE to the API.
func TestRoleResource_Delete_Success(t *testing.T) {
	const orgID, rgID = "test-org-id", "test-rg-id"
	var deleteCalled bool
	r := newRoleResourceWithDeleteHandler(t, orgID, rgID, func(w http.ResponseWriter, req *http.Request) {
		if req.Method == http.MethodDelete {
			deleteCalled = true
			// Role group DELETE returns 200 with an org-id array (not 204).
			testutil.JSONResponse(w, http.StatusOK, []string{orgID})
			return
		}
		w.WriteHeader(http.StatusOK)
	})

	ctx := context.Background()
	req := resource.DeleteRequest{State: buildRoleDeleteState(t, r, rgID, orgID)}
	resp := &resource.DeleteResponse{State: buildRoleDeleteState(t, r, rgID, orgID)}
	r.Delete(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() reported errors: %v", resp.Diagnostics.Errors())
	}
	if !deleteCalled {
		t.Error("expected DELETE to be issued to the API")
	}
}

// TestRoleResource_Delete_NotFoundIsIdempotent is the Class H regression: if the role
// group is already gone (API returns 404), Delete must succeed silently rather than
// erroring the apply. This is the branch `if client.IsNotFound(err) { return }`.
func TestRoleResource_Delete_NotFoundIsIdempotent(t *testing.T) {
	const orgID, rgID = "test-org-id", "test-rg-id"
	r := newRoleResourceWithDeleteHandler(t, orgID, rgID, func(w http.ResponseWriter, req *http.Request) {
		// Simulate a role group deleted out-of-band: every call 404s.
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("Role group not found"))
	})

	ctx := context.Background()
	req := resource.DeleteRequest{State: buildRoleDeleteState(t, r, rgID, orgID)}
	resp := &resource.DeleteResponse{State: buildRoleDeleteState(t, r, rgID, orgID)}
	r.Delete(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Delete() must treat a 404 (already-deleted) as success, but reported: %v (Class H regression)", resp.Diagnostics.Errors())
	}
}
