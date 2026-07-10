package accessmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTeamDataSource(t *testing.T) {
	dataSource := NewTeamDataSource()

	if dataSource == nil {
		t.Error("NewTeamDataSource() returned nil")
	}

	// Verify it implements the expected interfaces
	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("TeamDataSource does not implement DataSourceWithConfigure")
	}
}

func TestTeamDataSource_Metadata(t *testing.T) {
	dataSource := NewTeamDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_team" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_team")
	}
}

func TestTeamDataSource_Schema(t *testing.T) {
	dataSource := NewTeamDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check required attributes (none for this data source)
	requiredAttrs := []string{}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	// Check computed attributes (id is Required, not Computed)
	computedAttrs := []string{"name", "organization_id"}
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

func TestTeamDataSource_Configure(t *testing.T) {
	dataSource := NewTeamDataSource().(*TeamDataSource)

	// Test with valid provider data
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

	// Verify client is configured
	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestTeamDataSourceModel_Validation(t *testing.T) {
	// Test that all model fields exist and are properly typed
	model := TeamDataSourceModel{}

	// Verify all expected fields exist
	_ = model.ID
	// Add other field validations based on your model
}

func TestTeamDataSource_Read(t *testing.T) {
	tests := []struct {
		name         string
		teamID       string
		orgID        string
		clientOrgID  string
		mockHandler  func(w http.ResponseWriter, r *http.Request)
		wantErr      bool
		errContains  string
		expectedName string
	}{
		{
			name:        "successful read with provided org ID",
			teamID:      "test-team-id",
			orgID:       "test-org-id",
			clientOrgID: "default-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/teams/test-team-id")
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"team_id":      "test-team-id",
					"name":         "Test Team",
					"team_name":    "Test Team",
					"org_id":       "test-org-id",
					"team_type":    "internal",
					"created_date": "2023-01-01T00:00:00Z",
					"updated_date": "2023-01-01T00:00:00Z",
					"member_count": 5,
				})
			},
			wantErr:      false,
			expectedName: "Test Team",
		},
		{
			name:        "successful read with default org ID",
			teamID:      "test-team-id",
			orgID:       "",
			clientOrgID: "default-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/default-org-id/teams/test-team-id")
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"team_id":      "test-team-id",
					"name":         "Default Team",
					"team_name":    "Default Team",
					"org_id":       "default-org-id",
					"team_type":    "internal",
					"created_date": "2023-01-01T00:00:00Z",
					"updated_date": "2023-01-01T00:00:00Z",
					"member_count": 3,
				})
			},
			wantErr:      false,
			expectedName: "Default Team",
		},
		{
			name:        "team not found",
			teamID:      "nonexistent-team-id",
			orgID:       "test-org-id",
			clientOrgID: "default-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
			},
			wantErr:     true,
			errContains: "Could not read team ID nonexistent-team-id",
		},
		{
			name:        "server error",
			teamID:      "test-team-id",
			orgID:       "test-org-id",
			clientOrgID: "default-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "Could not read team ID test-team-id",
		},
		{
			name:        "malformed API response",
			teamID:      "test-team-id",
			orgID:       "test-org-id",
			clientOrgID: "default-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "Could not read team ID test-team-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handlers for different org/team combinations
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/organizations/test-org-id/teams/test-team-id":           tt.mockHandler,
				"/accounts/api/organizations/test-org-id/teams/nonexistent-team-id":    tt.mockHandler,
				"/accounts/api/organizations/default-org-id/teams/test-team-id":        tt.mockHandler,
				"/accounts/api/organizations/default-org-id/teams/nonexistent-team-id": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			// Create client with mock server
			teamClient := &accessmanagement.TeamClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
					OrgID:      tt.clientOrgID,
				},
			}

			// Test the underlying client directly since testing the full Terraform
			// data source would require complex setup of terraform-plugin-framework types
			orgID := tt.orgID
			if orgID == "" {
				orgID = tt.clientOrgID
			}

			team, err := teamClient.GetTeam(context.Background(), orgID, tt.teamID)

			// Verify results
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTeam() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					// For read tests, we check if error contains key parts
					hasExpectedError := strings.Contains(err.Error(), "not found") ||
						strings.Contains(err.Error(), "500") ||
						strings.Contains(err.Error(), "invalid character")
					if !hasExpectedError {
						t.Errorf("GetTeam() error = %v, want error containing patterns for %v", err, tt.errContains)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetTeam() unexpected error = %v", err)
				}
				if team == nil {
					t.Errorf("GetTeam() returned nil team")
				}
				if team != nil && team.TeamName != tt.expectedName {
					t.Errorf("GetTeam() TeamName = %v, want %v", team.TeamName, tt.expectedName)
				}
			}
		})
	}
}

func TestTeamDataSource_Read_Direct(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/teams/test-team-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id":    "test-team-id",
				"team_name":  "Test Team",
				"team_type":  "internal",
				"org_id":     "test-org-id",
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
			})
		},
		// Team role assignments: one normal + one internal (internal must be
		// excluded) + one platform-injected phantom whose role_id is NOT in the
		// catalog (e.g. "Business Group Viewer") — that must be excluded too,
		// otherwise it surfaces with an empty name.
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
						"role_id":        "role-business-group-viewer",
						"name":           "Business Group Viewer",
						"internal":       false,
						"context_params": map[string]string{"org": "test-org-id"},
					},
				},
				"total": 3,
			})
		},
		// Available-roles catalog (role_id -> display name).
		"/accounts/api/roles": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{"role_id": "role-exchange-viewer", "name": "Exchange Viewer", "internal": false},
					{"role_id": "role-internal", "name": "Internal System Role", "internal": true},
				},
				"total": 2,
			})
		},
		// Team members: one normal + one external-group member (both real org users,
		// both surface) + one platform-injected group identity whose id is NOT an org
		// user (e.g. the member the platform adds once a team gains a child team) —
		// that must be excluded, otherwise it surfaces with an empty username.
		basePath + "/members": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{"id": "user-1", "membership_type": "member", "is_assigned_via_external_groups": false},
					{"id": "user-2", "membership_type": "maintainer", "is_assigned_via_external_groups": true},
					{"id": "group-identity-phantom", "membership_type": "member", "is_assigned_via_external_groups": false},
				},
				"total": 3,
			})
		},
		// Org users (user_id -> username).
		"/accounts/api/organizations/test-org-id/users": func(w http.ResponseWriter, r *http.Request) {
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
	ds := NewTeamDataSource().(*TeamDataSource)
	ds.client = &accessmanagement.TeamClient{UserAnypointClient: userClient}
	ds.rolesClient = &accessmanagement.TeamRolesClient{UserAnypointClient: userClient}
	ds.membersClient = &accessmanagement.TeamMembersClient{UserAnypointClient: userClient}
	ds.usersClient = &accessmanagement.RoleUsersClient{UserAnypointClient: userClient}
	ds.catalogClient = &accessmanagement.RolePermissionClient{UserAnypointClient: userClient}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	roleObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	memberObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"username":                        tftypes.String,
		"membership_type":                 tftypes.String,
		"is_assigned_via_external_groups": tftypes.Bool,
	}}
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-team-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"parent_team":     tftypes.NewValue(tftypes.String, nil),
		"team_type":       tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"roles":           tftypes.NewValue(tftypes.List{ElementType: roleObjType}, nil),
		"members":         tftypes.NewValue(tftypes.List{ElementType: memberObjType}, nil),
		"created_at":      tftypes.NewValue(tftypes.String, nil),
		"updated_at":      tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got TeamDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "Test Team" {
		t.Errorf("Expected Name 'Test Team', got %s", got.Name.ValueString())
	}
	// Roles: only the non-internal, in-catalog assignment should surface, labeled
	// by display name. The internal role and the non-catalog "Business Group
	// Viewer" phantom must both be excluded.
	if got.Roles.IsNull() {
		t.Fatalf("Expected roles to be populated, got null")
	}
	if n := len(got.Roles.Elements()); n != 1 {
		t.Errorf("Expected 1 (non-internal, in-catalog) role, got %d", n)
	}
	// The surviving role must be the catalog role — never the empty-named phantom.
	for _, el := range got.Roles.Elements() {
		obj := el.(types.Object)
		name := obj.Attributes()["name"].(types.String).ValueString()
		if name == "" {
			t.Errorf("A role surfaced with an empty name — the Business Group Viewer phantom leaked through the filter")
		}
		if name != "Exchange Viewer" {
			t.Errorf("Expected surviving role 'Exchange Viewer', got %q", name)
		}
	}
	// Members: the two real org users surface (the external-group flag is reported,
	// not filtered); the non-org-user "group identity" phantom must be excluded, so
	// no member may have an empty username.
	if got.Members.IsNull() {
		t.Fatalf("Expected members to be populated, got null")
	}
	if n := len(got.Members.Elements()); n != 2 {
		t.Errorf("Expected 2 members (group-identity phantom excluded), got %d", n)
	}
	for _, el := range got.Members.Elements() {
		obj := el.(types.Object)
		username := obj.Attributes()["username"].(types.String).ValueString()
		if username == "" {
			t.Errorf("A member surfaced with an empty username — the non-org-user group-identity phantom leaked through the filter")
		}
	}
}

func TestTeamDataSource_Read_Direct_Error(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/teams/test-team-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewTeamDataSource().(*TeamDataSource)
	ds.client = &accessmanagement.TeamClient{
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

	roleObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	memberObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"username":                        tftypes.String,
		"membership_type":                 tftypes.String,
		"is_assigned_via_external_groups": tftypes.Bool,
	}}
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-team-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"parent_team":     tftypes.NewValue(tftypes.String, nil),
		"team_type":       tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"roles":           tftypes.NewValue(tftypes.List{ElementType: roleObjType}, nil),
		"members":         tftypes.NewValue(tftypes.List{ElementType: memberObjType}, nil),
		"created_at":      tftypes.NewValue(tftypes.String, nil),
		"updated_at":      tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

// Benchmarks
func BenchmarkTeamDataSource_Schema(b *testing.B) {
	dataSource := NewTeamDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
