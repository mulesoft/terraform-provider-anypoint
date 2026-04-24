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

func TestNewEnvironmentClient(t *testing.T) {
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
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.UserClientConfig{
				ClientID: "test-client-id",
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

			client, err := NewEnvironmentClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewEnvironmentClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewEnvironmentClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewEnvironmentClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewEnvironmentClient() returned nil client")
				}
			}
		})
	}
}


func TestEnvironmentClient_CreateEnvironment(t *testing.T) {
	mockEnvironment := &Environment{
		ID:             "test-environment-id",
		Name:           "Test Environment",
		Type:           "sandbox",
		IsProduction:   false,
		OrganizationID: "test-org-id",
		ClientID:       "test-client-id",
	}

	tests := []struct {
		name                string
		orgID               string
		request             *CreateEnvironmentRequest
		mockHandler         func(w http.ResponseWriter, r *http.Request)
		wantErr             bool
		errContains         string
		expectedEnvironment *Environment
	}{
		{
			name: "successful creation",
			orgID: "test-org-id",
			request: &CreateEnvironmentRequest{
				Name:         "Test Environment",
				Type:         "sandbox",
				IsProduction: false,
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/environments")
				
				body := testutil.AssertJSONBody(t, r, "name", "type")
				if body["name"] != "Test Environment" {
					t.Errorf("Expected name 'Test Environment', got %v", body["name"])
				}
				
				testutil.JSONResponse(w, http.StatusCreated, mockEnvironment)
			},
			wantErr: false,
			expectedEnvironment: mockEnvironment,
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			request: &CreateEnvironmentRequest{
				Name:         "Test Environment",
				Type:         "sandbox",
				IsProduction: false,
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to create environment with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/environments", tt.orgID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &EnvironmentClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.CreateEnvironment(context.Background(), tt.orgID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateEnvironment() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("CreateEnvironment() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("CreateEnvironment() returned nil environment")
				}
				
				// Validate returned environment
				if result != nil && tt.expectedEnvironment != nil {
					if result.ID != tt.expectedEnvironment.ID {
						t.Errorf("CreateEnvironment() ID = %v, want %v", result.ID, tt.expectedEnvironment.ID)
					}
					if result.Name != tt.expectedEnvironment.Name {
						t.Errorf("CreateEnvironment() Name = %v, want %v", result.Name, tt.expectedEnvironment.Name)
					}
					if result.Type != tt.expectedEnvironment.Type {
						t.Errorf("CreateEnvironment() Type = %v, want %v", result.Type, tt.expectedEnvironment.Type)
					}
				}
			}
		})
	}
}



func TestEnvironmentClient_GetEnvironment(t *testing.T) {
	mockEnvironment := &Environment{
		ID:             "test-environment-id",
		Name:           "Test Environment",
		Type:           "sandbox",
		IsProduction:   false,
		OrganizationID: "test-org-id",
		ClientID:       "test-client-id",
	}

	tests := []struct {
		name                string
		orgID               string
		environmentID       string
		mockHandler         func(w http.ResponseWriter, r *http.Request)
		wantErr             bool
		errContains         string
		expectedEnvironment *Environment
	}{
		{
			name:          "successful get",
			orgID:         "test-org-id",
			environmentID: "test-environment-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/environments/test-environment-id")
				testutil.JSONResponse(w, http.StatusOK, mockEnvironment)
			},
			wantErr:             false,
			expectedEnvironment: mockEnvironment,
		},
		{
			name:          "environment not found",
			orgID:         "test-org-id",
			environmentID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Environment not found")
			},
			wantErr:     true,
			errContains: "environment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/environments/%s", tt.orgID, tt.environmentID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &EnvironmentClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.GetEnvironment(context.Background(), tt.orgID, tt.environmentID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetEnvironment() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("GetEnvironment() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetEnvironment() returned nil environment")
				}
				
				// Validate returned environment
				if result != nil && tt.expectedEnvironment != nil {
					if result.ID != tt.expectedEnvironment.ID {
						t.Errorf("GetEnvironment() ID = %v, want %v", result.ID, tt.expectedEnvironment.ID)
					}
					if result.Name != tt.expectedEnvironment.Name {
						t.Errorf("GetEnvironment() Name = %v, want %v", result.Name, tt.expectedEnvironment.Name)
					}
					if result.Type != tt.expectedEnvironment.Type {
						t.Errorf("GetEnvironment() Type = %v, want %v", result.Type, tt.expectedEnvironment.Type)
					}
					if result.IsProduction != tt.expectedEnvironment.IsProduction {
						t.Errorf("GetEnvironment() IsProduction = %v, want %v", result.IsProduction, tt.expectedEnvironment.IsProduction)
					}
				}
			}
		})
	}
}



func TestEnvironmentClient_UpdateEnvironment(t *testing.T) {
	mockEnvironment := &Environment{
		ID:             "test-environment-id",
		Name:           "Updated Environment",
		Type:           "production",
		IsProduction:   true,
		OrganizationID: "test-org-id",
		ClientID:       "test-client-id",
	}

	tests := []struct {
		name          string
		orgID         string
		environmentID string
		request       *UpdateEnvironmentRequest
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errContains   string
	}{
		{
			name:          "successful update",
			orgID:         "test-org-id",
			environmentID: "test-environment-id",
			request: &UpdateEnvironmentRequest{
				Name:         testutil.StringPtr("Updated Environment"),
				Type:         testutil.StringPtr("production"),
				IsProduction: testutil.BoolPtr(true),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/accounts/api/organizations/test-org-id/environments/test-environment-id")
				
				body := testutil.AssertJSONBody(t, r, "name", "type")
				if body["name"] != "Updated Environment" {
					t.Errorf("Expected name 'Updated Environment', got %v", body["name"])
				}
				
				testutil.JSONResponse(w, http.StatusOK, mockEnvironment)
			},
			wantErr: false,
		},
		{
			name:          "environment not found",
			orgID:         "test-org-id",
			environmentID: "nonexistent-id",
			request: &UpdateEnvironmentRequest{
				Name: testutil.StringPtr("Updated Environment"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Environment not found")
			},
			wantErr:     true,
			errContains: "environment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/environments/%s", tt.orgID, tt.environmentID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &EnvironmentClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := client.UpdateEnvironment(context.Background(), tt.orgID, tt.environmentID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateEnvironment() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("UpdateEnvironment() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("UpdateEnvironment() returned nil")
				}
			}
		})
	}
}



func TestEnvironmentClient_DeleteEnvironment(t *testing.T) {
	tests := []struct {
		name          string
		orgID         string
		environmentID string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errContains   string
	}{
		{
			name:          "successful deletion",
			orgID:         "test-org-id",
			environmentID: "test-environment-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/environments/test-environment-id")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:          "environment not found",
			orgID:         "test-org-id",
			environmentID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Environment not found")
			},
			wantErr:     true,
			errContains: "environment not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/environments/%s", tt.orgID, tt.environmentID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &EnvironmentClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteEnvironment(context.Background(), tt.orgID, tt.environmentID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteEnvironment() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeleteEnvironment() unexpected error = %v", err)
				}
			}
		})
	}
}


// JSON serialization test
func TestEnvironment_JSONSerialization(t *testing.T) {
	env := &Environment{
		ID:             "test-environment-id",
		Name:           "Test Environment",
		Type:           "sandbox",
		IsProduction:   false,
		OrganizationID: "test-org-id",
		ClientID:       "test-client-id",
	}

	// Test marshaling
	data, err := json.Marshal(env)
	if err != nil {
		t.Fatalf("Failed to marshal Environment: %v", err)
	}

	// Test unmarshaling
	var decoded Environment
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal Environment: %v", err)
	}

	// Validate key fields
	if decoded.ID != env.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, env.ID)
	}
	if decoded.Name != env.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", decoded.Name, env.Name)
	}
	if decoded.Type != env.Type {
		t.Errorf("Unmarshaled Type = %v, want %v", decoded.Type, env.Type)
	}
	if decoded.IsProduction != env.IsProduction {
		t.Errorf("Unmarshaled IsProduction = %v, want %v", decoded.IsProduction, env.IsProduction)
	}
}

