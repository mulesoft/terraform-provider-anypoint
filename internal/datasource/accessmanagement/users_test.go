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

func TestNewUsersDataSource(t *testing.T) {
	ds := NewUsersDataSource()

	if ds == nil {
		t.Error("NewUsersDataSource() returned nil")
	}

	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("UsersDataSource does not implement DataSourceWithConfigure")
	}
}

func TestUsersDataSource_Metadata(t *testing.T) {
	ds := NewUsersDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "anypoint",
	}
	resp := &datasource.MetadataResponse{}

	ds.Metadata(ctx, req, resp)

	if resp.TypeName != "anypoint_users" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "anypoint_users")
	}
}

func TestUsersDataSource_Schema(t *testing.T) {
	ds := NewUsersDataSource()

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

	// Check optional organization_id attribute
	if attr, exists := resp.Schema.Attributes["organization_id"]; exists {
		if attr.IsRequired() {
			t.Error("Schema() attribute 'organization_id' should be optional")
		}
	} else {
		t.Error("Schema() missing attribute: organization_id")
	}

	// Check computed users attribute
	if attr, exists := resp.Schema.Attributes["users"]; exists {
		if !attr.IsComputed() {
			t.Error("Schema() attribute 'users' should be computed")
		}
	} else {
		t.Error("Schema() missing computed attribute: users")
	}
}

func TestUsersDataSource_Configure(t *testing.T) {
	ds := NewUsersDataSource().(*UsersDataSource)

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

func TestUsersDataSource_Configure_NilProviderData(t *testing.T) {
	ds := NewUsersDataSource().(*UsersDataSource)

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

func TestUsersDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewUsersDataSource().(*UsersDataSource)

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid",
	}
	resp := &datasource.ConfigureResponse{}

	ds.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid ProviderData should have errors")
	}
}

func TestUsersDataSource_Read(t *testing.T) {
	usersPath := "/accounts/api/organizations/test-org-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		usersPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(accessmanagement.ListOrgUsersResponse{
				Data: []accessmanagement.OrgUser{
					{ID: "user-aaa", Username: "alice", FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", Enabled: true},
					{ID: "user-bbb", Username: "bob", FirstName: "Bob", LastName: "Jones", Email: "bob@example.com", Enabled: true},
					{ID: "user-ccc", Username: "charlie", FirstName: "Charlie", LastName: "Brown", Email: "charlie@example.com", Enabled: false},
				},
				Total: 3,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewUsersDataSource().(*UsersDataSource)
	ds.client = &accessmanagement.RoleUsersClient{
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

	// No filter - should return all users
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"name_filter":     tftypes.NewValue(tftypes.String, nil),
		"users": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"id":         tftypes.String,
				"username":   tftypes.String,
				"first_name": tftypes.String,
				"last_name":  tftypes.String,
				"email":      tftypes.String,
				"enabled":    tftypes.Bool,
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

	var got UsersDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Users.Elements()) != 3 {
		t.Errorf("Expected 3 users, got %d", len(got.Users.Elements()))
	}
}

func TestUsersDataSource_Read_WithFilter(t *testing.T) {
	usersPath := "/accounts/api/organizations/test-org-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		usersPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(accessmanagement.ListOrgUsersResponse{
				Data: []accessmanagement.OrgUser{
					{ID: "user-aaa", Username: "alice", FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", Enabled: true},
					{ID: "user-bbb", Username: "bob", FirstName: "Bob", LastName: "Jones", Email: "bob@example.com", Enabled: true},
					{ID: "user-ccc", Username: "charlie", FirstName: "Charlie", LastName: "Brown", Email: "charlie@example.com", Enabled: false},
				},
				Total: 3,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewUsersDataSource().(*UsersDataSource)
	ds.client = &accessmanagement.RoleUsersClient{
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

	// Filter for "alice" - should match username
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"name_filter":     tftypes.NewValue(tftypes.String, "alice"),
		"users": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"id":         tftypes.String,
				"username":   tftypes.String,
				"first_name": tftypes.String,
				"last_name":  tftypes.String,
				"email":      tftypes.String,
				"enabled":    tftypes.Bool,
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

	var got UsersDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Users.Elements()) != 1 {
		t.Errorf("Expected 1 user matching 'alice', got %d", len(got.Users.Elements()))
	}
}

func TestUsersDataSource_Read_CaseInsensitiveFilter(t *testing.T) {
	usersPath := "/accounts/api/organizations/test-org-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		usersPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(accessmanagement.ListOrgUsersResponse{
				Data: []accessmanagement.OrgUser{
					{ID: "user-aaa", Username: "alice", FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", Enabled: true},
					{ID: "user-bbb", Username: "bob", FirstName: "Bob", LastName: "Jones", Email: "bob@example.com", Enabled: true},
				},
				Total: 2,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewUsersDataSource().(*UsersDataSource)
	ds.client = &accessmanagement.RoleUsersClient{
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

	// Filter "JONES" (uppercase) should match last_name "Jones"
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"name_filter":     tftypes.NewValue(tftypes.String, "JONES"),
		"users": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"id":         tftypes.String,
				"username":   tftypes.String,
				"first_name": tftypes.String,
				"last_name":  tftypes.String,
				"email":      tftypes.String,
				"enabled":    tftypes.Bool,
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

	var got UsersDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Users.Elements()) != 1 {
		t.Errorf("Expected 1 user matching 'JONES' (case-insensitive), got %d", len(got.Users.Elements()))
	}
}

func TestUsersDataSource_Read_FilterByEmail(t *testing.T) {
	usersPath := "/accounts/api/organizations/test-org-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		usersPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(accessmanagement.ListOrgUsersResponse{
				Data: []accessmanagement.OrgUser{
					{ID: "user-aaa", Username: "alice", FirstName: "Alice", LastName: "Smith", Email: "alice@salesforce.com", Enabled: true},
					{ID: "user-bbb", Username: "bob", FirstName: "Bob", LastName: "Jones", Email: "bob@mulesoft.com", Enabled: true},
				},
				Total: 2,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewUsersDataSource().(*UsersDataSource)
	ds.client = &accessmanagement.RoleUsersClient{
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

	// Filter by email domain
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"name_filter":     tftypes.NewValue(tftypes.String, "mulesoft"),
		"users": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"id":         tftypes.String,
				"username":   tftypes.String,
				"first_name": tftypes.String,
				"last_name":  tftypes.String,
				"email":      tftypes.String,
				"enabled":    tftypes.Bool,
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

	var got UsersDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Users.Elements()) != 1 {
		t.Errorf("Expected 1 user matching email 'mulesoft', got %d", len(got.Users.Elements()))
	}
}

func TestUsersDataSource_Read_Error(t *testing.T) {
	usersPath := "/accounts/api/organizations/test-org-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		usersPath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusInternalServerError)
			_, _ = w.Write([]byte("Internal server error"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewUsersDataSource().(*UsersDataSource)
	ds.client = &accessmanagement.RoleUsersClient{
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
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"name_filter":     tftypes.NewValue(tftypes.String, nil),
		"users": tftypes.NewValue(
			tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
				"id":         tftypes.String,
				"username":   tftypes.String,
				"first_name": tftypes.String,
				"last_name":  tftypes.String,
				"email":      tftypes.String,
				"enabled":    tftypes.Bool,
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
