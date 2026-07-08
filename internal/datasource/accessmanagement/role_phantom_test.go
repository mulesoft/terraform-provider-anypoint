package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestRoleDataSource_Read_SkipsNonCatalogPhantom is a REPRODUCE-FIRST regression test
// for the Class A phantom leak on the anypoint_role DATA SOURCE read path — the third
// of the three role paths that share this bug class (resource apply, resource read,
// data-source read).
//
// The role-assignments list contains a platform-injected side-effect grant
// ("Business Group Viewer") whose role_id is NOT in the available-roles catalog. The
// user never assigned it and cannot express it. With the pre-fix data source (which
// skipped only internal assignments), it surfaced with name = "" (catalog lookup
// miss), producing a perpetual output diff. The catalog-skip must exclude it exactly
// as the resource reconcile does.
func TestRoleDataSource_Read_SkipsNonCatalogPhantom(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-role-group-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"role_group_id":  "test-role-group-id",
				"name":           "Test Role Group",
				"description":    "A test role group",
				"org_id":         "test-org-id",
				"editable":       true,
				"external_names": []string{},
				"created_at":     "2024-01-01T00:00:00Z",
				"updated_at":     "2024-01-01T00:00:00Z",
			})
		},
		// Assignments: one real in-catalog + one internal + one non-catalog phantom.
		basePath + "/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"role_id":        "role-exchange-viewer",
						"name":           "Exchange Viewer",
						"internal":       false,
						"context_params": map[string]string{"org": "test-org-id"},
					},
					{
						"role_id":        "role-internal",
						"name":           "Internal System Role",
						"internal":       true,
						"context_params": map[string]string{},
					},
					{
						// Platform-injected, org-scoped side-effect: non-internal, NOT in catalog.
						"role_id":        "role-business-group-viewer",
						"name":           "Business Group Viewer",
						"internal":       false,
						"context_params": map[string]string{"org": "test-org-id"},
					},
				},
				"total": 3,
			})
		},
		// Catalog omits "Business Group Viewer" — that is what makes it a phantom.
		"/accounts/api/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{"role_id": "role-exchange-viewer", "name": "Exchange Viewer", "internal": false},
					{"role_id": "role-internal", "name": "Internal System Role", "internal": true},
				},
				"total": 2,
			})
		},
		basePath + "/users": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data":  []map[string]interface{}{{"id": "user-1", "username": "alice"}},
				"total": 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	userClient := &client.UserAnypointClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
		OrgID:      "test-org-id",
	}
	ds := NewRoleDataSource().(*RoleDataSource)
	ds.client = &accessmanagement.RoleClient{UserAnypointClient: userClient}
	ds.permClient = &accessmanagement.RolePermissionClient{UserAnypointClient: userClient}
	ds.usersClient = &accessmanagement.RoleUsersClient{UserAnypointClient: userClient}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	permObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-role-group-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"description":     tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"editable":        tftypes.NewValue(tftypes.Bool, nil),
		"external_names":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"permissions":     tftypes.NewValue(tftypes.List{ElementType: permObjType}, nil),
		"members":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"created_at":      tftypes.NewValue(tftypes.String, nil),
		"updated_at":      tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got RoleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Permissions.IsNull() {
		t.Fatalf("expected permissions populated, got null")
	}
	if n := len(got.Permissions.Elements()); n != 1 {
		t.Errorf("expected exactly 1 permission (internal + non-catalog phantom excluded), got %d", n)
	}
	for _, el := range got.Permissions.Elements() {
		obj := el.(types.Object)
		name := obj.Attributes()["name"].(types.String).ValueString()
		if name == "" {
			t.Error("a permission surfaced with an empty name — the Business Group Viewer phantom leaked through the data-source filter (Class A regression)")
		}
		if name != "Exchange Viewer" {
			t.Errorf("expected surviving permission 'Exchange Viewer', got %q", name)
		}
	}
}
