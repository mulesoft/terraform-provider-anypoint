package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"
	"time"

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

func TestOrganizationClient_UpdateOrganization(t *testing.T) {
	updatedOrganization := &Organization{
		ID:        "test-org-id",
		Name:      "Renamed Organization",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-02-02T00:00:00Z",
		OwnerID:   "test-owner-id",
		Entitlements: Entitlements{
			CreateSubOrgs:      false,
			CreateEnvironments: true,
			GlobalDeployment:   false,
		},
	}

	tests := []struct {
		name        string
		orgID       string
		request     *UpdateOrganizationRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
		expectedOrg *Organization
	}{
		{
			name:  "successful update",
			orgID: "test-org-id",
			request: &UpdateOrganizationRequest{
				ID:         "test-org-id",
				Name:       "Renamed Organization",
				OwnerID:    "test-owner-id",
				Properties: map[string]interface{}{"flow_designer": map[string]interface{}{}},
				Entitlements: Entitlements{
					CreateSubOrgs:      false,
					CreateEnvironments: true,
					GlobalDeployment:   false,
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/accounts/api/organizations/test-org-id")
				body := testutil.AssertJSONBody(t, r, "id", "name", "ownerId", "properties", "entitlements")

				if body["name"] != "Renamed Organization" {
					t.Errorf("Expected name 'Renamed Organization', got %v", body["name"])
				}
				if body["id"] != "test-org-id" {
					t.Errorf("Expected id 'test-org-id', got %v", body["id"])
				}
				if body["ownerId"] != "test-owner-id" {
					t.Errorf("Expected ownerId 'test-owner-id', got %v", body["ownerId"])
				}
				if _, ok := body["parentOrganizationId"]; ok {
					t.Errorf("parentOrganizationId must not be present in update payload")
				}
				ent, ok := body["entitlements"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected entitlements to be an object, got %T", body["entitlements"])
				}
				if ent["createSubOrgs"] != false {
					t.Errorf("Expected entitlements.createSubOrgs=false, got %v", ent["createSubOrgs"])
				}
				if ent["createEnvironments"] != true {
					t.Errorf("Expected entitlements.createEnvironments=true, got %v", ent["createEnvironments"])
				}
				if ent["globalDeployment"] != false {
					t.Errorf("Expected entitlements.globalDeployment=false, got %v", ent["globalDeployment"])
				}

				testutil.JSONResponse(w, http.StatusOK, updatedOrganization)
			},
			wantErr:     false,
			expectedOrg: updatedOrganization,
		},
		{
			name:  "nil properties is sent as empty object",
			orgID: "test-org-id",
			request: &UpdateOrganizationRequest{
				ID:      "test-org-id",
				Name:    "Still-a-name",
				OwnerID: "test-owner-id",
				// Properties intentionally nil
				Entitlements: Entitlements{
					CreateSubOrgs:      true,
					CreateEnvironments: true,
					GlobalDeployment:   false,
				},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/accounts/api/organizations/test-org-id")
				body := testutil.AssertJSONBody(t, r, "properties")
				props, ok := body["properties"].(map[string]interface{})
				if !ok {
					t.Fatalf("Expected properties to be an object, got %T (%v)", body["properties"], body["properties"])
				}
				if len(props) != 0 {
					t.Errorf("Expected properties to be an empty object, got %v", props)
				}

				testutil.JSONResponse(w, http.StatusOK, updatedOrganization)
			},
			wantErr:     false,
			expectedOrg: updatedOrganization,
		},
		{
			name:  "organization not found",
			orgID: "missing-org-id",
			request: &UpdateOrganizationRequest{
				ID: "missing-org-id", Name: "x", OwnerID: "o",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Organization not found")
			},
			wantErr:     true,
			errContains: "organization not found",
		},
		{
			name:  "server rejects with 400",
			orgID: "test-org-id",
			request: &UpdateOrganizationRequest{
				ID: "test-org-id", Name: "bad", OwnerID: "o",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid entitlements")
			},
			wantErr:     true,
			errContains: "failed to update organization with status 400",
		},
		{
			name:    "empty organization ID",
			orgID:   "",
			request: &UpdateOrganizationRequest{Name: "x"},
			mockHandler: func(_ http.ResponseWriter, _ *http.Request) {
				t.Errorf("request should not be dispatched when organization ID is empty")
			},
			wantErr:     true,
			errContains: "organization ID is required",
		},
		{
			name:        "nil request body",
			orgID:       "test-org-id",
			request:     nil,
			mockHandler: func(_ http.ResponseWriter, _ *http.Request) {},
			wantErr:     true,
			errContains: "update request cannot be nil",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){}
			if tt.orgID != "" {
				handlers[fmt.Sprintf("/accounts/api/organizations/%s", tt.orgID)] = tt.mockHandler
			}
			server := testutil.MockHTTPServer(t, handlers)

			oc := &OrganizationClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			org, err := oc.UpdateOrganization(context.Background(), tt.orgID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateOrganization() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateOrganization() error = %v, want error containing %v", err, tt.errContains)
				}
				return
			}

			if err != nil {
				t.Errorf("UpdateOrganization() unexpected error = %v", err)
			}
			if org == nil {
				t.Fatalf("UpdateOrganization() returned nil organization")
			}
			if tt.expectedOrg != nil {
				if org.ID != tt.expectedOrg.ID {
					t.Errorf("UpdateOrganization() ID = %v, want %v", org.ID, tt.expectedOrg.ID)
				}
				if org.Name != tt.expectedOrg.Name {
					t.Errorf("UpdateOrganization() Name = %v, want %v", org.Name, tt.expectedOrg.Name)
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
				Assigned: 5,
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

// TestOrganizationClient_WaitForOrganizationDeletion covers the three
// outcomes of the post-DELETE polling loop:
//
//  1. Soft-delete (Anypoint's actual behaviour): GET keeps returning 200 but
//     the `deletedAt` field becomes non-nil. The helper MUST return nil so
//     the provider doesn't fire a misleading "Deletion Timeout" warning on
//     every successful destroy.
//  2. Hard-delete: GET returns 404. The helper MUST return nil.
//  3. Neither signal arrives within `maxRetries`: the helper MUST return
//     a timeout error so the provider can surface the warning for real.
func TestOrganizationClient_WaitForOrganizationDeletion(t *testing.T) {
	ctx := context.Background()
	const orgID = "org-to-delete"

	t.Run("soft delete surfaces deletedAt", func(t *testing.T) {
		var calls int
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/organizations/" + orgID: func(w http.ResponseWriter, r *http.Request) {
				calls++
				if calls < 2 {
					// First poll: not marked deleted yet.
					testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
						"id":        orgID,
						"name":      "org-to-delete",
						"deletedAt": nil,
					})
					return
				}
				// Subsequent poll: deletedAt populated, hasn't 404ed yet.
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"id":        orgID,
					"name":      "org-to-delete",
					"deletedAt": "2026-04-30T07:30:00.000Z",
				})
			},
			"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
			"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &OrganizationClient{
			UserAnypointClient: &client.UserAnypointClient{
				BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{},
			},
		}
		err := c.WaitForOrganizationDeletion(ctx, orgID, 5, 1*time.Millisecond)
		if err != nil {
			t.Fatalf("soft-deleted org should resolve cleanly; got error: %v", err)
		}
		if calls < 2 {
			t.Errorf("expected at least 2 polls before deletedAt appeared, got %d", calls)
		}
	})

	t.Run("hard delete returns 404", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/organizations/" + orgID: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "organization not found")
			},
			"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
			"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &OrganizationClient{
			UserAnypointClient: &client.UserAnypointClient{
				BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{},
			},
		}
		if err := c.WaitForOrganizationDeletion(ctx, orgID, 3, 1*time.Millisecond); err != nil {
			t.Errorf("404 should resolve cleanly; got error: %v", err)
		}
	})

	t.Run("genuine timeout still errors", func(t *testing.T) {
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/accounts/api/organizations/" + orgID: func(w http.ResponseWriter, r *http.Request) {
				// Always return a live, non-deleted org so the loop exhausts retries.
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"id":        orgID,
					"name":      "org-to-delete",
					"deletedAt": nil,
				})
			},
			"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
			"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
		}
		server := testutil.MockHTTPServer(t, handlers)

		c := &OrganizationClient{
			UserAnypointClient: &client.UserAnypointClient{
				BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{},
			},
		}
		err := c.WaitForOrganizationDeletion(ctx, orgID, 3, 1*time.Millisecond)
		if err == nil {
			t.Fatal("expected timeout error when neither 404 nor deletedAt materialise")
		}
		if !strings.Contains(err.Error(), "timeout") {
			t.Errorf("expected timeout error, got: %v", err)
		}
	})
}
