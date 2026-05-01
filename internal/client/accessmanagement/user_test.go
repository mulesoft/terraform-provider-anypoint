package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewUserClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *client.UserClientConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &client.UserClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.UserClientConfig{
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.UserClientConfig{
				ClientID: "test-client-id",
				Username: "test-user",
				Password: "test-password",
			},
			wantErr:     true,
			errContains: "client_secret is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())

			if tt.config != nil {
				tt.config.BaseURL = server.URL
			}

			client, err := NewUserClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewUserClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewUserClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewUserClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewUserClient() returned nil client")
				}
			}
		})
	}
}

func TestUserClient_CreateUser(t *testing.T) {
	mockUser := &User{
		ID:        "test-user-id",
		Username:  "testuser",
		FirstName: "Test",
		LastName:  "User",
		Email:     "test@example.com",
		Enabled:   true,
		Organization: UserOrganization{
			ID:          "test-org-id",
			IsFederated: false,
		},
	}

	tests := []struct {
		name         string
		orgID        string
		request      *CreateUserRequest
		mockHandler  func(w http.ResponseWriter, r *http.Request)
		wantErr      bool
		errContains  string
		expectedUser *User
	}{
		{
			name:  "successful creation",
			orgID: "test-org-id",
			request: &CreateUserRequest{
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
				Password:  "password123",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/users")

				body := testutil.AssertJSONBody(t, r, "username", "firstName", "lastName", "email", "password")

				if body["username"] != "testuser" {
					t.Errorf("Expected username 'testuser', got %v", body["username"])
				}
				if body["email"] != "test@example.com" {
					t.Errorf("Expected email 'test@example.com', got %v", body["email"])
				}

				testutil.JSONResponse(w, http.StatusCreated, mockUser)
			},
			wantErr:      false,
			expectedUser: mockUser,
		},
		{
			name:  "user already exists",
			orgID: "test-org-id",
			request: &CreateUserRequest{
				Username:  "existinguser",
				FirstName: "Existing",
				LastName:  "User",
				Email:     "existing@example.com",
				Password:  "password123",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusConflict, "User already exists")
			},
			wantErr:     true,
			errContains: "user already exists",
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			request: &CreateUserRequest{
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
				Password:  "password123",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to create user with status 500",
		},
		{
			name:  "malformed response",
			orgID: "test-org-id",
			request: &CreateUserRequest{
				Username:  "testuser",
				FirstName: "Test",
				LastName:  "User",
				Email:     "test@example.com",
				Password:  "password123",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/users", tt.orgID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			user, err := client.CreateUser(context.Background(), tt.orgID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateUser() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreateUser() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateUser() unexpected error = %v", err)
				}
				if user == nil {
					t.Errorf("CreateUser() returned nil user")
				}

				// Validate returned user
				if user != nil && tt.expectedUser != nil {
					if user.ID != tt.expectedUser.ID {
						t.Errorf("CreateUser() ID = %v, want %v", user.ID, tt.expectedUser.ID)
					}
					if user.Username != tt.expectedUser.Username {
						t.Errorf("CreateUser() Username = %v, want %v", user.Username, tt.expectedUser.Username)
					}
					if user.Email != tt.expectedUser.Email {
						t.Errorf("CreateUser() Email = %v, want %v", user.Email, tt.expectedUser.Email)
					}
				}
			}
		})
	}
}

func TestUserClient_GetUser(t *testing.T) {
	mockUser := &User{
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
		Organization: UserOrganization{
			ID:          "test-org-id",
			IsFederated: false,
		},
	}

	tests := []struct {
		name         string
		orgID        string
		userID       string
		mockHandler  func(w http.ResponseWriter, r *http.Request)
		wantErr      bool
		errContains  string
		expectedUser *User
	}{
		{
			name:   "successful get",
			orgID:  "test-org-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/users/test-user-id")
				testutil.JSONResponse(w, http.StatusOK, mockUser)
			},
			wantErr:      false,
			expectedUser: mockUser,
		},
		{
			name:   "user not found",
			orgID:  "test-org-id",
			userID: "nonexistent-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "User not found")
			},
			wantErr:     true,
			errContains: "user not found",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get user with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/users/%s", tt.orgID, tt.userID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			user, err := client.GetUser(context.Background(), tt.orgID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetUser() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetUser() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetUser() unexpected error = %v", err)
				}
				if user == nil {
					t.Errorf("GetUser() returned nil user")
				}

				// Validate returned user
				if user != nil && tt.expectedUser != nil {
					if user.ID != tt.expectedUser.ID {
						t.Errorf("GetUser() ID = %v, want %v", user.ID, tt.expectedUser.ID)
					}
					if user.Username != tt.expectedUser.Username {
						t.Errorf("GetUser() Username = %v, want %v", user.Username, tt.expectedUser.Username)
					}
					if user.Email != tt.expectedUser.Email {
						t.Errorf("GetUser() Email = %v, want %v", user.Email, tt.expectedUser.Email)
					}
					if user.PhoneNumber != tt.expectedUser.PhoneNumber {
						t.Errorf("GetUser() PhoneNumber = %v, want %v", user.PhoneNumber, tt.expectedUser.PhoneNumber)
					}
				}
			}
		})
	}
}

func TestUserClient_UpdateUser(t *testing.T) {
	mockUser := &User{
		ID:        "test-user-id",
		Username:  "updateduser",
		FirstName: "Updated",
		LastName:  "User",
		Email:     "updated@example.com",
		Enabled:   true,
	}

	tests := []struct {
		name        string
		orgID       string
		userID      string
		request     *UpdateUserRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful update",
			orgID:  "test-org-id",
			userID: "test-user-id",
			request: &UpdateUserRequest{
				Username:  testutil.StringPtr("updateduser"),
				FirstName: testutil.StringPtr("Updated"),
				Email:     testutil.StringPtr("updated@example.com"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/accounts/api/organizations/test-org-id/users/test-user-id")
				testutil.JSONResponse(w, http.StatusOK, mockUser)
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			orgID:  "test-org-id",
			userID: "nonexistent-user-id",
			request: &UpdateUserRequest{
				Username: testutil.StringPtr("updateduser"),
			},
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
				fmt.Sprintf("/accounts/api/organizations/%s/users/%s", tt.orgID, tt.userID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			user, err := client.UpdateUser(context.Background(), tt.orgID, tt.userID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateUser() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateUser() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateUser() unexpected error = %v", err)
				}
				if user == nil {
					t.Errorf("UpdateUser() returned nil user")
				}
			}
		})
	}
}

func TestUserClient_DeleteUser(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		userID      string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful deletion",
			orgID:  "test-org-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/users/test-user-id")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:   "user not found",
			orgID:  "test-org-id",
			userID: "nonexistent-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "User not found")
			},
			wantErr:     true,
			errContains: "User not found",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to delete user with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/users/%s", tt.orgID, tt.userID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &UserClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteUser(context.Background(), tt.orgID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteUser() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteUser() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteUser() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestUser_JSONSerialization(t *testing.T) {
	user := &User{
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
		Organization: UserOrganization{
			ID:          "test-org-id",
			IsFederated: false,
		},
	}

	// Test marshaling
	data, err := json.Marshal(user)
	if err != nil {
		t.Fatalf("Failed to marshal user: %v", err)
	}

	// Test unmarshaling
	var decoded User
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal user: %v", err)
	}

	// Validate key fields
	if decoded.ID != user.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, user.ID)
	}
	if decoded.Username != user.Username {
		t.Errorf("Unmarshaled Username = %v, want %v", decoded.Username, user.Username)
	}
	if decoded.Email != user.Email {
		t.Errorf("Unmarshaled Email = %v, want %v", decoded.Email, user.Email)
	}
	if decoded.Organization.ID != user.Organization.ID {
		t.Errorf("Unmarshaled Organization.ID = %v, want %v", decoded.Organization.ID, user.Organization.ID)
	}
}
