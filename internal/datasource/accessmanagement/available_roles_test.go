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

func TestNewAvailableRolesDataSource(t *testing.T) {
	ds := NewAvailableRolesDataSource()

	if ds == nil {
		t.Error("NewAvailableRolesDataSource() returned nil")
	}

	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("AvailableRolesDataSource does not implement DataSourceWithConfigure")
	}
}

func TestAvailableRolesDataSource_Metadata(t *testing.T) {
	ds := NewAvailableRolesDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "anypoint",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(ctx, req, resp)

	if resp.TypeName != "anypoint_available_permissions" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "anypoint_available_permissions")
	}
}

func TestAvailableRolesDataSource_Schema(t *testing.T) {
	ds := NewAvailableRolesDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	ds.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check optional name_filter attribute
	if attr, exists := resp.Schema.Attributes["name_filter"]; exists {
		if attr.IsRequired() {
			t.Error("Schema() attribute 'name_filter' should be optional, not required")
		}
	} else {
		t.Error("Schema() missing attribute: name_filter")
	}

	// Check computed permissions attribute
	if attr, exists := resp.Schema.Attributes["permissions"]; exists {
		if !attr.IsComputed() {
			t.Error("Schema() attribute 'permissions' should be computed")
		}
	} else {
		t.Error("Schema() missing computed attribute: permissions")
	}
}

func TestAvailableRolesDataSource_Configure(t *testing.T) {
	ds := NewAvailableRolesDataSource().(*AvailableRolesDataSource)

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

func TestAvailableRolesDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewAvailableRolesDataSource().(*AvailableRolesDataSource)

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: nil,
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Error("Configure() with nil ProviderData should not have errors")
	}
}

func TestAvailableRolesDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewAvailableRolesDataSource().(*AvailableRolesDataSource)

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid", // not *client.Config
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid ProviderData should have errors")
	}
}

func TestAvailableRolesDataSource_Read(t *testing.T) {
	rolesPath := "/accounts/api/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		rolesPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"role_id":     "role-aaa",
						"name":        "Audit Log Viewer",
						"description": "View audit logs",
						"internal":    false,
					},
					{
						"role_id":     "role-bbb",
						"name":        "Read Applications",
						"description": "Read applications in environment",
						"internal":    false,
					},
					{
						"role_id":     "role-ccc",
						"name":        "Create Applications",
						"description": "Create new applications",
						"internal":    false,
					},
				},
				"total": 3,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAvailableRolesDataSource().(*AvailableRolesDataSource)
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

	// No filter - should return all roles
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"name_filter": tftypes.NewValue(tftypes.String, nil), // null = no filter
		"permissions": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"role_id":     tftypes.String,
				"name":        tftypes.String,
				"description": tftypes.String,
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

	var got AvailableRolesDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Permissions.Elements()) != 3 {
		t.Errorf("Expected 3 roles, got %d", len(got.Permissions.Elements()))
	}
}

func TestAvailableRolesDataSource_Read_WithFilter(t *testing.T) {
	rolesPath := "/accounts/api/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		rolesPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"role_id":     "role-aaa",
						"name":        "Audit Log Viewer",
						"description": "View audit logs",
						"internal":    false,
					},
					{
						"role_id":     "role-bbb",
						"name":        "Read Applications",
						"description": "Read applications in environment",
						"internal":    false,
					},
					{
						"role_id":     "role-ccc",
						"name":        "Create Applications",
						"description": "Create new applications",
						"internal":    false,
					},
				},
				"total": 3,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAvailableRolesDataSource().(*AvailableRolesDataSource)
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

	// Filter for "Applications" - should match "Read Applications" and "Create Applications"
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"name_filter": tftypes.NewValue(tftypes.String, "Applications"),
		"permissions": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"role_id":     tftypes.String,
				"name":        tftypes.String,
				"description": tftypes.String,
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

	var got AvailableRolesDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Permissions.Elements()) != 2 {
		t.Errorf("Expected 2 roles matching 'Applications', got %d", len(got.Permissions.Elements()))
	}
}

func TestAvailableRolesDataSource_Read_CaseInsensitiveFilter(t *testing.T) {
	rolesPath := "/accounts/api/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		rolesPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"role_id":     "role-aaa",
						"name":        "Audit Log Viewer",
						"description": "View audit logs",
						"internal":    false,
					},
					{
						"role_id":     "role-bbb",
						"name":        "Read Applications",
						"description": "Read applications in environment",
						"internal":    false,
					},
				},
				"total": 2,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAvailableRolesDataSource().(*AvailableRolesDataSource)
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

	// Filter using lowercase "audit" - should match "Audit Log Viewer"
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"name_filter": tftypes.NewValue(tftypes.String, "audit"),
		"permissions": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"role_id":     tftypes.String,
				"name":        tftypes.String,
				"description": tftypes.String,
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

	var got AvailableRolesDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Permissions.Elements()) != 1 {
		t.Errorf("Expected 1 role matching 'audit' (case-insensitive), got %d", len(got.Permissions.Elements()))
	}
}

func TestAvailableRolesDataSource_Read_Error(t *testing.T) {
	rolesPath := "/accounts/api/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		rolesPath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAvailableRolesDataSource().(*AvailableRolesDataSource)
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
		"name_filter": tftypes.NewValue(tftypes.String, nil),
		"permissions": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"role_id":     tftypes.String,
				"name":        tftypes.String,
				"description": tftypes.String,
			}}},
			nil,
		),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func TestContainsIgnoreCase(t *testing.T) {
	tests := []struct {
		s      string
		substr string
		want   bool
	}{
		{"Read Applications", "applications", true},
		{"Read Applications", "APPLICATIONS", true},
		{"Read Applications", "Read", true},
		{"Read Applications", "write", false},
		{"Audit Log Viewer", "audit", true},
		{"Audit Log Viewer", "AUDIT LOG", true},
		{"", "test", false},
		{"test", "", true},
	}

	for _, tt := range tests {
		got := containsIgnoreCase(tt.s, tt.substr)
		if got != tt.want {
			t.Errorf("containsIgnoreCase(%q, %q) = %v, want %v", tt.s, tt.substr, got, tt.want)
		}
	}
}
