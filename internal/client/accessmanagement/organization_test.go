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

func TestNewOrganizationClient(t *testing.T) {
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
			// Create mock server for user authentication
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/v2/oauth2/token": func(w http.ResponseWriter, r *http.Request) {
					testutil.JSONResponse(w, http.StatusOK, testutil.MockAuthResponse())
				},
				"/accounts/api/me": func(w http.ResponseWriter, r *http.Request) {
					testutil.JSONResponse(w, http.StatusOK, testutil.MockMeResponse())
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			if tt.config != nil {
				tt.config.BaseURL = server.URL
			}

			client, err := NewOrganizationClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewOrganizationClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewOrganizationClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewOrganizationClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewOrganizationClient() returned nil client")
				}
			}
		})
	}
}

func TestOrganizationClient_CreateOrganization(t *testing.T) {
	mockOrganization := &Organization{
		ID:        "test-org-id",
		Name:      "Test Organization",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-01-01T00:00:00Z",
		OwnerID:   "test-owner-id",
	}

	tests := []struct {
		name        string
		request     *CreateOrganizationRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		expectedOrg *Organization
	}{
		{
			name: "successful creation",
			request: &CreateOrganizationRequest{
				Name:                 "Test Organization",
				ParentOrganizationID: "parent-org-id",
				OwnerID:              "test-owner-id",
				Entitlements: Entitlements{
					CreateSubOrgs:      true,
					CreateEnvironments: true,
					GlobalDeployment:   true,
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations")
				
				// Validate request body
				body := testutil.AssertJSONBody(t, r, "name", "parentOrganizationId", "ownerId", "entitlements")
				
				if body["name"] != "Test Organization" {
					t.Errorf("Expected name 'Test Organization', got %v", body["name"])
				}
				
				testutil.JSONResponse(w, http.StatusCreated, mockOrganization)
			},
			wantErr:     false,
			expectedOrg: mockOrganization,
		},
		{
			name: "server returns error",
			request: &CreateOrganizationRequest{
				Name:                 "Test Organization",
				ParentOrganizationID: "parent-org-id",
				OwnerID:              "test-owner-id",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid request")
			},
			wantErr:     true,
			errContains: "failed to create organization with status 400",
		},
		{
			name: "malformed response",
			request: &CreateOrganizationRequest{
				Name:                 "Test Organization",
				ParentOrganizationID: "parent-org-id",
				OwnerID:              "test-owner-id",
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
				"/accounts/api/organizations": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &OrganizationClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			org, err := client.CreateOrganization(context.Background(), tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateOrganization() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreateOrganization() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateOrganization() unexpected error = %v", err)
				}
				if org == nil {
					t.Errorf("CreateOrganization() returned nil organization")
				}
				
				// Validate returned organization
				if org != nil && tt.expectedOrg != nil {
					if org.ID != tt.expectedOrg.ID {
						t.Errorf("CreateOrganization() ID = %v, want %v", org.ID, tt.expectedOrg.ID)
					}
					if org.Name != tt.expectedOrg.Name {
						t.Errorf("CreateOrganization() Name = %v, want %v", org.Name, tt.expectedOrg.Name)
					}
				}
			}
		})
	}
}

func TestOrganizationClient_GetOrganization(t *testing.T) {
	mockOrganization := &Organization{
		ID:        "test-org-id",
		Name:      "Test Organization",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-01-01T00:00:00Z",
		OwnerID:   "test-owner-id",
		Owner: Owner{
			ID:        "test-owner-id",
			FirstName: "Test",
			LastName:  "Owner",
			Email:     "test@example.com",
			Username:  "testowner",
		},
		Entitlements: Entitlements{
			CreateSubOrgs:      true,
			CreateEnvironments: true,
			GlobalDeployment:   true,
		},
		Subscription: Subscription{
			Category:   "Trial",
			Type:       "Trial",
			Expiration: "2024-01-01T00:00:00Z",
		},
	}

	tests := []struct {
		name        string
		orgID       string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		expectedOrg *Organization
	}{
		{
			name:  "successful get",
			orgID: "test-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id")
				testutil.JSONResponse(w, http.StatusOK, mockOrganization)
			},
			wantErr:     false,
			expectedOrg: mockOrganization,
		},
		{
			name:  "organization not found",
			orgID: "nonexistent-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/nonexistent-org-id")
				testutil.ErrorResponse(w, http.StatusNotFound, "Organization not found")
			},
			wantErr:     true,
			errContains: "organization not found",
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get organization with status 500",
		},
		{
			name:  "malformed response",
			orgID: "test-org-id",
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
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s", tt.orgID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &OrganizationClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			org, err := client.GetOrganization(context.Background(), tt.orgID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetOrganization() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetOrganization() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetOrganization() unexpected error = %v", err)
				}
				if org == nil {
					t.Errorf("GetOrganization() returned nil organization")
				}
				
				// Validate returned organization
				if org != nil && tt.expectedOrg != nil {
					if org.ID != tt.expectedOrg.ID {
						t.Errorf("GetOrganization() ID = %v, want %v", org.ID, tt.expectedOrg.ID)
					}
					if org.Name != tt.expectedOrg.Name {
						t.Errorf("GetOrganization() Name = %v, want %v", org.Name, tt.expectedOrg.Name)
					}
					if org.Owner.ID != tt.expectedOrg.Owner.ID {
						t.Errorf("GetOrganization() Owner.ID = %v, want %v", org.Owner.ID, tt.expectedOrg.Owner.ID)
					}
					if org.Entitlements.CreateSubOrgs != tt.expectedOrg.Entitlements.CreateSubOrgs {
						t.Errorf("GetOrganization() Entitlements.CreateSubOrgs = %v, want %v", 
							org.Entitlements.CreateSubOrgs, tt.expectedOrg.Entitlements.CreateSubOrgs)
					}
				}
			}
		})
	}
}

// Test data structures marshaling/unmarshaling
func TestOrganization_JSONSerialization(t *testing.T) {
	org := &Organization{
		ID:        "test-org-id",
		Name:      "Test Organization",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-01-01T00:00:00Z",
		OwnerID:   "test-owner-id",
		Entitlements: Entitlements{
			CreateSubOrgs:      true,
			CreateEnvironments: true,
			GlobalDeployment:   false,
			VCoresProduction: &VCoreEntitlement{
				Assigned:   10,
				Reassigned: 5,
			},
			VCoresSandbox: &VCoreEntitlement{
				Assigned:   5,
			},
		},
		Subscription: Subscription{
			Category:   "Trial",
			Type:       "Trial",
			Expiration: "2024-01-01T00:00:00Z",
		},
	}

	// Test marshaling
	data, err := json.Marshal(org)
	if err != nil {
		t.Fatalf("Failed to marshal organization: %v", err)
	}

	// Test unmarshaling
	var decoded Organization
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal organization: %v", err)
	}

	// Validate key fields
	if decoded.ID != org.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, org.ID)
	}
	if decoded.Name != org.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", decoded.Name, org.Name)
	}
	if decoded.Entitlements.CreateSubOrgs != org.Entitlements.CreateSubOrgs {
		t.Errorf("Unmarshaled Entitlements.CreateSubOrgs = %v, want %v", 
			decoded.Entitlements.CreateSubOrgs, org.Entitlements.CreateSubOrgs)
	}
	if decoded.Entitlements.VCoresProduction == nil || decoded.Entitlements.VCoresProduction.Assigned != org.Entitlements.VCoresProduction.Assigned {
		t.Errorf("Unmarshaled VCoresProduction.Assigned = %v, want %v",
			decoded.Entitlements.VCoresProduction, org.Entitlements.VCoresProduction)
	}
}

func TestCreateOrganizationRequest_JSONSerialization(t *testing.T) {
	req := &CreateOrganizationRequest{
		Name:                 "Test Organization",
		ParentOrganizationID: "parent-org-id",
		OwnerID:              "owner-id",
		Entitlements: Entitlements{
			CreateSubOrgs:      true,
			CreateEnvironments: false,
			GlobalDeployment:   true,
		},
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal create request: %v", err)
	}

	// Test unmarshaling
	var decoded CreateOrganizationRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal create request: %v", err)
	}

	// Validate fields
	if decoded.Name != req.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", decoded.Name, req.Name)
	}
	if decoded.ParentOrganizationID != req.ParentOrganizationID {
		t.Errorf("Unmarshaled ParentOrganizationID = %v, want %v", 
			decoded.ParentOrganizationID, req.ParentOrganizationID)
	}
	if decoded.Entitlements.CreateSubOrgs != req.Entitlements.CreateSubOrgs {
		t.Errorf("Unmarshaled Entitlements.CreateSubOrgs = %v, want %v", 
			decoded.Entitlements.CreateSubOrgs, req.Entitlements.CreateSubOrgs)
	}
}

// Test edge cases and error scenarios
func TestOrganizationClient_ErrorHandling(t *testing.T) {
	tests := []struct {
		name        string
		setupClient func() *OrganizationClient
		operation   func(client *OrganizationClient) error
		wantErr     bool
	}{
		{
			name: "nil request to CreateOrganization",
			setupClient: func() *OrganizationClient {
				return &OrganizationClient{
					UserAnypointClient: &client.UserAnypointClient{
						BaseURL:    "http://test.com",
						Token:      "mock-token",
						HTTPClient: &http.Client{},
					},
				}
			},
			operation: func(client *OrganizationClient) error {
				_, err := client.CreateOrganization(context.Background(), nil)
				return err
			},
			wantErr: true,
		},
		{
			name: "empty organization ID to GetOrganization",
			setupClient: func() *OrganizationClient {
				return &OrganizationClient{
					UserAnypointClient: &client.UserAnypointClient{
						BaseURL:    "http://test.com",
						Token:      "mock-token",
						HTTPClient: &http.Client{},
					},
				}
			},
			operation: func(client *OrganizationClient) error {
				_, err := client.GetOrganization(context.Background(), "")
				return err
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			client := tt.setupClient()
			err := tt.operation(client)
			
			if tt.wantErr && err == nil {
				t.Errorf("%s expected error, got nil", tt.name)
			} else if !tt.wantErr && err != nil {
				t.Errorf("%s unexpected error = %v", tt.name, err)
			}
		})
	}
}