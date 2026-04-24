package secretsmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const testSSID = "test-ss-id"

func TestSharedSecretClient_CRUD(t *testing.T) {
	mockSS := &SharedSecretResponse{
		Name:     "Test Shared Secret",
		Type:     "usernamePassword",
		Meta:     SecretGroupMeta{ID: testSSID},
		Username: "admin",
	}

	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/sharedSecrets"
	itemPath := basePath + "/" + testSSID

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", basePath)
				body := testutil.AssertJSONBody(t, r, "name", "type")
				if body["name"] != "Test Shared Secret" {
					t.Errorf("Expected name 'Test Shared Secret', got %v", body["name"])
				}
				testutil.JSONResponse(w, http.StatusCreated, CreateResponse{ID: testSSID, Message: "created"})
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", basePath)
				testutil.JSONResponse(w, http.StatusOK, []SharedSecretResponse{*mockSS})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", itemPath)
				testutil.JSONResponse(w, http.StatusOK, mockSS)
			case "PUT":
				testutil.AssertHTTPRequest(t, r, "PUT", itemPath)
				testutil.JSONResponse(w, http.StatusOK, nil)
			case "DELETE":
				testutil.AssertHTTPRequest(t, r, "DELETE", itemPath)
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &SharedSecretClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateSharedSecret(ctx, testOrgID, testEnvID, testSGID, &SharedSecret{
		Name:     "Test Shared Secret",
		Type:     "usernamePassword",
		Username: "admin",
		Password: "secret123",
	})
	if err != nil {
		t.Fatalf("CreateSharedSecret() unexpected error: %v", err)
	}
	if created.Meta.ID != testSSID {
		t.Errorf("CreateSharedSecret() ID = %v, want %v", created.Meta.ID, testSSID)
	}
	if created.Name != "Test Shared Secret" {
		t.Errorf("CreateSharedSecret() Name = %v, want 'Test Shared Secret'", created.Name)
	}
	if created.Username != "admin" {
		t.Errorf("CreateSharedSecret() Username = %v, want 'admin'", created.Username)
	}

	// Get
	got, err := c.GetSharedSecret(ctx, testOrgID, testEnvID, testSGID, testSSID)
	if err != nil {
		t.Fatalf("GetSharedSecret() unexpected error: %v", err)
	}
	if got.Meta.ID != testSSID {
		t.Errorf("GetSharedSecret() ID = %v, want %v", got.Meta.ID, testSSID)
	}

	// Update
	updated, err := c.UpdateSharedSecret(ctx, testOrgID, testEnvID, testSGID, testSSID, &SharedSecret{
		Name:     "Updated Shared Secret",
		Type:     "usernamePassword",
		Username: "admin",
		Password: "newpassword",
	})
	if err != nil {
		t.Fatalf("UpdateSharedSecret() unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateSharedSecret() returned nil")
	}

	// List
	list, err := c.ListSharedSecrets(ctx, testOrgID, testEnvID, testSGID)
	if err != nil {
		t.Fatalf("ListSharedSecrets() unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListSharedSecrets() returned %d items, want 1", len(list))
	}

	// Delete
	if err := c.DeleteSharedSecret(ctx, testOrgID, testEnvID, testSGID, testSSID); err != nil {
		t.Fatalf("DeleteSharedSecret() unexpected error: %v", err)
	}
}

func TestSharedSecretClient_Errors(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/sharedSecrets"

	tests := []struct {
		name        string
		action      string
		handler     func(w http.ResponseWriter, r *http.Request)
		errContains string
	}{
		{
			name:   "get not found",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Not found")
			},
			errContains: "shared secret not found",
		},
		{
			name:   "get server error",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to get shared secret with status 500",
		},
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Bad request")
			},
			errContains: "failed to create shared secret with status 400",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to delete shared secret with status 500",
		},
		{
			name:   "list server error",
			action: "list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to list shared secrets with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath:                tt.handler,
				basePath + "/bad-id":    tt.handler,
				basePath + "/" + testSSID: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &SharedSecretClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}
			ctx := context.Background()

			var err error
			switch tt.action {
			case "get":
				_, err = c.GetSharedSecret(ctx, testOrgID, testEnvID, testSGID, "bad-id")
			case "create":
				_, err = c.CreateSharedSecret(ctx, testOrgID, testEnvID, testSGID, &SharedSecret{Name: "x", Type: "usernamePassword"})
			case "delete":
				err = c.DeleteSharedSecret(ctx, testOrgID, testEnvID, testSGID, testSSID)
			case "list":
				_, err = c.ListSharedSecrets(ctx, testOrgID, testEnvID, testSGID)
			}

			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Error = %v, want error containing %q", err, tt.errContains)
			}
		})
	}
}
