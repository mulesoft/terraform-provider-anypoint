package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRoleDataSource(t *testing.T) {
	dataSource := NewRoleDataSource()

	if dataSource == nil {
		t.Error("NewRoleDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("RoleDataSource does not implement DataSourceWithConfigure")
	}
}

func TestRoleDataSource_Metadata(t *testing.T) {
	dataSource := NewRoleDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_role" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_role")
	}
}

func TestRoleDataSource_Schema(t *testing.T) {
	dataSource := NewRoleDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check required attributes
	requiredAttrs := []string{"id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	// Check computed attributes
	computedAttrs := []string{"name", "description", "editable", "external_names", "created_at", "updated_at"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}
}

func TestRoleDataSource_Configure(t *testing.T) {
	dataSource := NewRoleDataSource().(*RoleDataSource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: providerData,
	}
	resp := &datasource.ConfigureResponse{}

	dataSource.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}

	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestRoleDataSource_Read_Direct(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-role-group-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"role_group_id":  "test-role-group-id",
				"name":           "Test Role Group",
				"description":    "A test role group",
				"org_id":         "test-org-id",
				"editable":       true,
				"external_names": []string{"saml-group-1"},
				"created_at":     "2024-01-01T00:00:00Z",
				"updated_at":     "2024-01-01T00:00:00Z",
			})
		},
		// The data source Read always fetches permissions (assignments + catalog) and members.
		basePath + "/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"role_group_assignment_id": "assign-1",
						"role_group_id":            "test-role-group-id",
						"role_id":                  "role-exchange-viewer",
						"org_id":                   "test-org-id",
						"name":                     "Exchange Viewer",
						"internal":                 false,
						"context_params":           map[string]string{"org": "test-org-id"},
					},
					{
						"role_group_assignment_id": "assign-internal",
						"role_group_id":            "test-role-group-id",
						"role_id":                  "role-internal",
						"org_id":                   "test-org-id",
						"name":                     "Internal System Role",
						"internal":                 true,
						"context_params":           map[string]string{},
					},
				},
				"total": 2,
			})
		},
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
				"data": []map[string]interface{}{
					{"id": "user-1", "username": "alice"},
					{"id": "user-2", "username": "bob"},
				},
				"total": 2,
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
	if got.Name.ValueString() != "Test Role Group" {
		t.Errorf("Expected Name 'Test Role Group', got %s", got.Name.ValueString())
	}
	if got.Description.ValueString() != "A test role group" {
		t.Errorf("Expected Description 'A test role group', got %s", got.Description.ValueString())
	}
	if !got.Editable.ValueBool() {
		t.Errorf("Expected Editable true, got false")
	}

	// Permissions: only the non-internal assignment should surface, labeled by display name.
	if got.Permissions.IsNull() {
		t.Fatalf("Expected permissions to be populated, got null")
	}
	if n := len(got.Permissions.Elements()); n != 1 {
		t.Errorf("Expected 1 (non-internal) permission, got %d", n)
	}
	// Members: both usernames should surface.
	if got.Members.IsNull() {
		t.Fatalf("Expected members to be populated, got null")
	}
	if n := len(got.Members.Elements()); n != 2 {
		t.Errorf("Expected 2 members, got %d", n)
	}
}

func TestRoleDataSource_Read_Error(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/nonexistent-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Role group not found"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewRoleDataSource().(*RoleDataSource)
	ds.client = &accessmanagement.RoleClient{
		UserAnypointClient: &client.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	permObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "nonexistent-id"),
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

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on not found")
	}
}

func TestNewRolesDataSource(t *testing.T) {
	dataSource := NewRolesDataSource()

	if dataSource == nil {
		t.Error("NewRolesDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("RolesDataSource does not implement DataSourceWithConfigure")
	}
}

func TestRolesDataSource_Metadata(t *testing.T) {
	dataSource := NewRolesDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_roles" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_roles")
	}
}

func TestRolesDataSource_Schema(t *testing.T) {
	dataSource := NewRolesDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check roles attribute exists and is computed
	if attr, exists := resp.Schema.Attributes["roles"]; exists {
		if !attr.IsComputed() {
			t.Error("Schema() attribute 'roles' should be computed")
		}
	} else {
		t.Error("Schema() missing 'roles' attribute")
	}
}
