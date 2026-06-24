package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRolePermissionsDataSource(t *testing.T) {
	ds := NewRolePermissionsDataSource()

	if ds == nil {
		t.Error("NewRolePermissionsDataSource() returned nil")
	}

	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("RolePermissionsDataSource does not implement DataSourceWithConfigure")
	}
}

func TestRolePermissionsDataSource_Metadata(t *testing.T) {
	ds := NewRolePermissionsDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(ctx, req, resp)

	if resp.TypeName != "test_role_permissions" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_role_permissions")
	}
}

func TestRolePermissionsDataSource_Schema(t *testing.T) {
	ds := NewRolePermissionsDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check required attributes
	if attr, exists := resp.Schema.Attributes["role_group_id"]; exists {
		if !attr.IsRequired() {
			t.Error("Schema() attribute 'role_group_id' should be required")
		}
	} else {
		t.Error("Schema() missing required attribute: role_group_id")
	}

	// Check computed attributes
	if attr, exists := resp.Schema.Attributes["permissions"]; exists {
		if !attr.IsComputed() {
			t.Error("Schema() attribute 'permissions' should be computed")
		}
	} else {
		t.Error("Schema() missing computed attribute: permissions")
	}
}

func TestRolePermissionsDataSource_Configure(t *testing.T) {
	ds := NewRolePermissionsDataSource().(*RolePermissionsDataSource)

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

	ds.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}

	if ds.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestRolePermissionsDataSource_Read(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"role_group_assignment_id": "assign-1",
						"role_group_id":            "test-rg-id",
						"role_id":                  "role-aaa",
						"org_id":                   "test-org-id",
						"name":                     "Audit Log Viewer",
						"description":              "View audit logs",
						"internal":                 false,
						"context_params":           map[string]string{"org": "test-org-id"},
						"created_at":               "2024-01-01T00:00:00Z",
					},
					{
						"role_group_assignment_id": "assign-2",
						"role_group_id":            "test-rg-id",
						"role_id":                  "role-bbb",
						"org_id":                   "test-org-id",
						"name":                     "Read Applications",
						"description":              "Read apps in env",
						"internal":                 false,
						"context_params":           map[string]string{"org": "test-org-id", "envId": "env-1"},
						"created_at":               "2024-01-02T00:00:00Z",
					},
				},
				"total": 2,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewRolePermissionsDataSource().(*RolePermissionsDataSource)
	ds.client = &accessmanagement.RolePermissionClient{
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

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"permissions": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"id":             tftypes.String,
				"role_id":        tftypes.String,
				"name":           tftypes.String,
				"description":    tftypes.String,
				"context_params": tftypes.Map{ElementType: tftypes.String},
				"created_at":     tftypes.String,
			}}},
			nil,
		),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got RolePermissionsDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.OrganizationID.ValueString() != "test-org-id" {
		t.Errorf("Expected OrganizationID 'test-org-id', got %s", got.OrganizationID.ValueString())
	}
	if len(got.Permissions.Elements()) != 2 {
		t.Errorf("Expected 2 permissions, got %d", len(got.Permissions.Elements()))
	}
}

func TestRolePermissionsDataSource_Read_Error(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/nonexistent/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			w.Write([]byte("Role group not found"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewRolePermissionsDataSource().(*RolePermissionsDataSource)
	ds.client = &accessmanagement.RolePermissionClient{
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

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"role_group_id":   tftypes.NewValue(tftypes.String, "nonexistent"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"permissions": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"id":             tftypes.String,
				"role_id":        tftypes.String,
				"name":           tftypes.String,
				"description":    tftypes.String,
				"context_params": tftypes.Map{ElementType: tftypes.String},
				"created_at":     tftypes.String,
			}}},
			nil,
		),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on not found")
	}
}
