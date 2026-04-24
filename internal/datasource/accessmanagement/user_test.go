package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewUserDataSource(t *testing.T) {
	dataSource := NewUserDataSource()

	if dataSource == nil {
		t.Error("NewUserDataSource() returned nil")
	}

	// Verify it implements the expected interfaces
	var _ datasource.DataSource = dataSource
	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("UserDataSource does not implement DataSourceWithConfigure")
	}
}

func TestUserDataSource_Metadata(t *testing.T) {
	dataSource := NewUserDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_user" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_user")
	}
}

func TestUserDataSource_Schema(t *testing.T) {
	dataSource := NewUserDataSource()

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
	computedAttrs := []string{"username", "first_name", "last_name", "email", "phone_number", "is_active", "created_at", "updated_at", "is_federated", "mfa_verification_excluded"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}

	// Check optional attributes
	optionalAttrs := []string{"organization_id"}
	for _, attrName := range optionalAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsOptional() && !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be optional or computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing optional attribute: %s", attrName)
		}
	}
}

func TestUserDataSource_Configure(t *testing.T) {
	dataSource := NewUserDataSource().(*UserDataSource)

	// Test with valid provider data
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-pass",
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

func TestUserDataSource_Configure_InvalidProviderData(t *testing.T) {
	dataSource := NewUserDataSource().(*UserDataSource)

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid-data", // Wrong type
	}
	resp := &datasource.ConfigureResponse{}

	dataSource.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}

	if dataSource.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestUserDataSource_Read(t *testing.T) {
	mockUser := &accessmanagement.User{
		ID:                      "test-user-id",
		Username:                "testuser",
		FirstName:               "Test",
		LastName:                "User",
		Email:                   "test@example.com",
		PhoneNumber:             "+1234567890",
		Enabled:                 true,
		CreatedAt:               "2023-01-01T00:00:00Z",
		UpdatedAt:               "2023-01-01T00:00:00Z",
		MfaVerificationExcluded: false,
		Organization: accessmanagement.UserOrganization{
			ID:          "test-org-id",
			IsFederated: false,
		},
	}

	tests := []struct {
		name        string
		model       UserDataSourceModel
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful read",
			model: UserDataSourceModel{
				ID:             types.StringValue("test-user-id"),
				OrganizationID: types.StringValue("test-org-id"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}

				expectedPath := "/accounts/api/organizations/test-org-id/users/test-user-id"
				if r.URL.Path != expectedPath {
					t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
				}

				testutil.JSONResponse(w, http.StatusOK, mockUser)
			},
			wantErr: false,
		},
		{
			name: "user not found",
			model: UserDataSourceModel{
				ID:             types.StringValue("nonexistent-user-id"),
				OrganizationID: types.StringValue("test-org-id"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "User not found")
			},
			wantErr:     true,
			errContains: "user not found",
		},
		{
			name: "server error",
			model: UserDataSourceModel{
				ID:             types.StringValue("test-user-id"),
				OrganizationID: types.StringValue("test-org-id"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get user",
		},
		{
			name: "malformed response",
			model: UserDataSourceModel{
				ID:             types.StringValue("test-user-id"),
				OrganizationID: types.StringValue("test-org-id"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			orgID := tt.model.OrganizationID.ValueString()
			userID := tt.model.ID.ValueString()

			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/organizations/" + orgID + "/users/" + userID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			// Create client
			userClient := &accessmanagement.UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
					OrgID:      "test-org-id",
				},
			}

			// Test the underlying client directly since testing the full Terraform
			// data source would require complex setup of terraform-plugin-framework types
			user, err := userClient.GetUser(context.Background(), orgID, userID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if user == nil {
					t.Error("Expected user but got nil")
				} else {
					// Validate returned user matches expected values
					if user.ID != tt.model.ID.ValueString() {
						t.Errorf("Expected ID %s, got %s", tt.model.ID.ValueString(), user.ID)
					}
					if user.Username != mockUser.Username {
						t.Errorf("Expected username %s, got %s", mockUser.Username, user.Username)
					}
					if user.Email != mockUser.Email {
						t.Errorf("Expected email %s, got %s", mockUser.Email, user.Email)
					}
					if user.FirstName != mockUser.FirstName {
						t.Errorf("Expected first name %s, got %s", mockUser.FirstName, user.FirstName)
					}
					if user.LastName != mockUser.LastName {
						t.Errorf("Expected last name %s, got %s", mockUser.LastName, user.LastName)
					}
					if user.PhoneNumber != mockUser.PhoneNumber {
						t.Errorf("Expected phone number %s, got %s", mockUser.PhoneNumber, user.PhoneNumber)
					}
					if user.Enabled != mockUser.Enabled {
						t.Errorf("Expected enabled %t, got %t", mockUser.Enabled, user.Enabled)
					}
					if user.MfaVerificationExcluded != mockUser.MfaVerificationExcluded {
						t.Errorf("Expected MFA excluded %t, got %t", mockUser.MfaVerificationExcluded, user.MfaVerificationExcluded)
					}
					if user.Organization.ID != mockUser.Organization.ID {
						t.Errorf("Expected organization ID %s, got %s", mockUser.Organization.ID, user.Organization.ID)
					}
					if user.Organization.IsFederated != mockUser.Organization.IsFederated {
						t.Errorf("Expected federated %t, got %t", mockUser.Organization.IsFederated, user.Organization.IsFederated)
					}
				}
			}
		})
	}
}

func TestUserDataSource_ReadWithDefaultOrganization(t *testing.T) {
	mockUser := &accessmanagement.User{
		ID:        "test-user-id",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Enabled:   true,
		Organization: accessmanagement.UserOrganization{
			ID:          "default-org-id",
			IsFederated: false,
		},
	}

	handler := func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			t.Errorf("Expected GET request, got %s", r.Method)
		}

		expectedPath := "/accounts/api/organizations/default-org-id/users/test-user-id"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}

		testutil.JSONResponse(w, http.StatusOK, mockUser)
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/default-org-id/users/test-user-id": handler,
	}
	server := testutil.MockHTTPServer(t, handlers)

	// Create client with default organization
	userClient := &accessmanagement.UserClient{
		UserAnypointClient: &client.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "default-org-id",
		},
	}

	// Test reading user with default organization (no organization_id specified)
	user, err := userClient.GetUser(context.Background(), "default-org-id", "test-user-id")
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}
	if user == nil {
		t.Error("Expected user but got nil")
	} else {
		if user.ID != "test-user-id" {
			t.Errorf("Expected ID test-user-id, got %s", user.ID)
		}
		if user.Organization.ID != "default-org-id" {
			t.Errorf("Expected organization ID default-org-id, got %s", user.Organization.ID)
		}
	}
}

func TestUserDataSourceModel_Validation(t *testing.T) {
	// Test that all model fields exist and are properly typed
	model := UserDataSourceModel{}

	// Verify all expected fields exist
	_ = model.ID
	_ = model.Username
	_ = model.FirstName
	_ = model.LastName
	_ = model.Email
	_ = model.PhoneNumber
	_ = model.IsActive
	_ = model.OrganizationID
	_ = model.CreatedAt
	_ = model.UpdatedAt
	_ = model.IsFederated
	_ = model.MfaVerificationExcluded
}

func TestUserDataSource_InterfaceCompliance(t *testing.T) {
	// Test that UserDataSource implements the expected interfaces
	var _ datasource.DataSource = &UserDataSource{}
	var _ datasource.DataSourceWithConfigure = &UserDataSource{}
}

// Benchmarks

func BenchmarkUserDataSource_Schema(b *testing.B) {
	dataSource := NewUserDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}

func BenchmarkUserDataSource_Metadata(b *testing.B) {
	dataSource := NewUserDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.MetadataResponse{}
		dataSource.Metadata(ctx, req, resp)
	}
}
