package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewUserResource(t *testing.T) {
	r := NewUserResource()

	if r == nil {
		t.Error("NewUserResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("UserResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("UserResource should implement ResourceWithImportState")
	}
}

func TestUserResource_Metadata(t *testing.T) {
	r := NewUserResource()
	testutil.TestResourceMetadata(t, r, "_user")
}

func TestUserResource_Schema(t *testing.T) {
	res := NewUserResource()

	requiredAttrs := []string{"username", "first_name", "last_name", "email", "password"}
	optionalAttrs := []string{"phone_number", "organization_id", "mfa_verification_excluded"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestUserResource_Configure(t *testing.T) {
	res := NewUserResource().(*UserResource)

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

func TestUserResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewUserResource().(*UserResource)

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

func TestUserResource_Create(t *testing.T) {
	mockUser := &accessmanagement.User{
		ID:                      "test-user-id",
		Username:                "testuser",
		FirstName:               "Test",
		LastName:                "User",
		Email:                   "test@example.com",
		PhoneNumber:             "+1234567890",
		Enabled:                 true,
		MfaVerificationExcluded: false,
		Organization: accessmanagement.UserOrganization{
			ID:          "test-org-id",
			IsFederated: false,
		},
	}

	tests := []struct {
		name        string
		model       UserResourceModel
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful creation",
			model: UserResourceModel{
				Username:                types.StringValue("testuser"),
				FirstName:               types.StringValue("Test"),
				LastName:                types.StringValue("User"),
				Email:                   types.StringValue("test@example.com"),
				Password:                types.StringValue("password123"),
				OrganizationID:          types.StringValue("test-org-id"),
				MfaVerificationExcluded: types.BoolValue(false),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusCreated, mockUser)
			},
			wantErr: false,
		},
		{
			name: "creation failure - user already exists",
			model: UserResourceModel{
				Username:  types.StringValue("existinguser"),
				FirstName: types.StringValue("Existing"),
				LastName:  types.StringValue("User"),
				Email:     types.StringValue("existing@example.com"),
				Password:  types.StringValue("password123"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusConflict, "User already exists")
			},
			wantErr:     true,
			errContains: "failed to create user",
		},
		{
			name: "creation failure - invalid email",
			model: UserResourceModel{
				Username:  types.StringValue("testuser"),
				FirstName: types.StringValue("Test"),
				LastName:  types.StringValue("User"),
				Email:     types.StringValue("invalid-email"),
				Password:  types.StringValue("password123"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid email format")
			},
			wantErr:     true,
			errContains: "failed to create user",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/organizations/test-org-id/users": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			userClient := &accessmanagement.UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
					OrgID:      "test-org-id",
				},
			}

			_ = &UserResource{
				client: userClient,
			}

			ctx := context.Background()

			createReq := &accessmanagement.CreateUserRequest{
				Username:                tt.model.Username.ValueString(),
				FirstName:               tt.model.FirstName.ValueString(),
				LastName:                tt.model.LastName.ValueString(),
				Email:                   tt.model.Email.ValueString(),
				Password:                tt.model.Password.ValueString(),
				MfaVerificationExcluded: tt.model.MfaVerificationExcluded.ValueBool(),
			}

			orgID := "test-org-id"
			if !tt.model.OrganizationID.IsNull() {
				orgID = tt.model.OrganizationID.ValueString()
			}

			user, err := userClient.CreateUser(ctx, orgID, createReq)

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
					if user.Username != tt.model.Username.ValueString() {
						t.Errorf("Expected username %s, got %s", tt.model.Username.ValueString(), user.Username)
					}
					if user.Email != tt.model.Email.ValueString() {
						t.Errorf("Expected email %s, got %s", tt.model.Email.ValueString(), user.Email)
					}
				}
			}
		})
	}
}

func TestUserResource_Read(t *testing.T) {
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
	}

	tests := []struct {
		name        string
		orgID       string
		userID      string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful read",
			orgID:  "test-org-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockUser)
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			orgID:  "test-org-id",
			userID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "User not found")
			},
			wantErr:     true,
			errContains: "user not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/organizations/" + tt.orgID + "/users/" + tt.userID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			userClient := &accessmanagement.UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			user, err := userClient.GetUser(context.Background(), tt.orgID, tt.userID)

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
					if user.ID != tt.userID {
						t.Errorf("Expected ID %s, got %s", tt.userID, user.ID)
					}
					if user.Username != mockUser.Username {
						t.Errorf("Expected username %s, got %s", mockUser.Username, user.Username)
					}
				}
			}
		})
	}
}

func TestUserResource_Update(t *testing.T) {
	mockUser := &accessmanagement.User{
		ID:        "test-user-id",
		Username:  "testuser",
		FirstName: "Updated",
		LastName:  "User",
		Email:     "updated@example.com",
		Enabled:   true,
	}

	tests := []struct {
		name        string
		orgID       string
		userID      string
		updateReq   *accessmanagement.UpdateUserRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:   "successful update",
			orgID:  "test-org-id",
			userID: "test-user-id",
			updateReq: &accessmanagement.UpdateUserRequest{
				FirstName: stringPtr("Updated"),
				Email:     stringPtr("updated@example.com"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("Expected PUT request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockUser)
			},
			wantErr: false,
		},
		{
			name:   "update failure - not found",
			orgID:  "test-org-id",
			userID: "nonexistent-id",
			updateReq: &accessmanagement.UpdateUserRequest{
				FirstName: stringPtr("Updated"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "User not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/organizations/" + tt.orgID + "/users/" + tt.userID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			userClient := &accessmanagement.UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			user, err := userClient.UpdateUser(context.Background(), tt.orgID, tt.userID, tt.updateReq)

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
				} else if user.FirstName != "Updated" {
					t.Errorf("Expected first name Updated, got %s", user.FirstName)
				}
			}
		})
	}
}

func TestUserResource_Delete(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		userID      string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:   "successful deletion",
			orgID:  "test-org-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:   "deletion failure - not found",
			orgID:  "test-org-id",
			userID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "User not found")
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/organizations/" + tt.orgID + "/users/" + tt.userID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			userClient := &accessmanagement.UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := userClient.DeleteUser(context.Background(), tt.orgID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestUserResource_ImportState(t *testing.T) {
	r := NewUserResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestUserResourceModel_Validation(t *testing.T) {
	model := UserResourceModel{}
	_ = model.ID
	_ = model.Username
	_ = model.FirstName
	_ = model.LastName
	_ = model.Email
	_ = model.PhoneNumber
	_ = model.Password
	_ = model.OrganizationID
	_ = model.MfaVerificationExcluded
}

func stringPtr(s string) *string {
	return &s
}

func BenchmarkUserResource_Schema(b *testing.B) {
	res := NewUserResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}

func BenchmarkUserResource_Metadata(b *testing.B) {
	res := NewUserResource()
	ctx := context.Background()
	req := resource.MetadataRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.MetadataResponse{}
		res.Metadata(ctx, req, resp)
	}
}
