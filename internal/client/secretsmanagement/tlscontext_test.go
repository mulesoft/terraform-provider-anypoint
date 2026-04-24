package secretsmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const testTLSID = "test-tls-id"

func TestTLSContextClient_CRUD(t *testing.T) {
	mockTLS := &TLSContextResponse{
		Name:   "Test TLS Context",
		Target: "Mule",
		Meta:   SecretGroupMeta{ID: testTLSID},
		Keystore: &TLSContextPathRef{
			Path: "/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/keystores/ks-1",
		},
		MinTLSVersion: "TLSv1.2",
		MaxTLSVersion: "TLSv1.3",
	}

	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/tlsContexts"
	itemPath := basePath + "/" + testTLSID

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", basePath)
				body := testutil.AssertJSONBody(t, r, "name", "target")
				if body["name"] != "Test TLS Context" {
					t.Errorf("Expected name 'Test TLS Context', got %v", body["name"])
				}
				testutil.JSONResponse(w, http.StatusCreated, CreateResponse{ID: testTLSID, Message: "created"})
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", basePath)
				testutil.JSONResponse(w, http.StatusOK, []TLSContextResponse{*mockTLS})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", itemPath)
				testutil.JSONResponse(w, http.StatusOK, mockTLS)
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

	c := &TLSContextClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateTLSContext(ctx, testOrgID, testEnvID, testSGID, &TLSContext{
		Name:   "Test TLS Context",
		Target: "Mule",
		Keystore: &TLSContextPathRef{
			Path: "/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/keystores/ks-1",
		},
		MinTLSVersion: "TLSv1.2",
		MaxTLSVersion: "TLSv1.3",
	})
	if err != nil {
		t.Fatalf("CreateTLSContext() unexpected error: %v", err)
	}
	if created.Meta.ID != testTLSID {
		t.Errorf("CreateTLSContext() ID = %v, want %v", created.Meta.ID, testTLSID)
	}
	if created.Name != "Test TLS Context" {
		t.Errorf("CreateTLSContext() Name = %v, want 'Test TLS Context'", created.Name)
	}
	if created.Target != "Mule" {
		t.Errorf("CreateTLSContext() Target = %v, want 'Mule'", created.Target)
	}

	// Get
	got, err := c.GetTLSContext(ctx, testOrgID, testEnvID, testSGID, testTLSID)
	if err != nil {
		t.Fatalf("GetTLSContext() unexpected error: %v", err)
	}
	if got.Meta.ID != testTLSID {
		t.Errorf("GetTLSContext() ID = %v, want %v", got.Meta.ID, testTLSID)
	}
	if got.MinTLSVersion != "TLSv1.2" {
		t.Errorf("GetTLSContext() MinTLSVersion = %v, want 'TLSv1.2'", got.MinTLSVersion)
	}

	// Update
	updated, err := c.UpdateTLSContext(ctx, testOrgID, testEnvID, testSGID, testTLSID, &TLSContext{
		Name:          "Updated TLS Context",
		Target:        "Mule",
		MinTLSVersion: "TLSv1.2",
		MaxTLSVersion: "TLSv1.3",
	})
	if err != nil {
		t.Fatalf("UpdateTLSContext() unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateTLSContext() returned nil")
	}

	// List
	list, err := c.ListTLSContexts(ctx, testOrgID, testEnvID, testSGID)
	if err != nil {
		t.Fatalf("ListTLSContexts() unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListTLSContexts() returned %d items, want 1", len(list))
	}

	// Delete
	if err := c.DeleteTLSContext(ctx, testOrgID, testEnvID, testSGID, testTLSID); err != nil {
		t.Fatalf("DeleteTLSContext() unexpected error: %v", err)
	}
}

func TestTLSContextClient_Errors(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/tlsContexts"

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
			errContains: "TLS context not found",
		},
		{
			name:   "get server error",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to get TLS context with status 500",
		},
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Bad request")
			},
			errContains: "failed to create TLS context with status 400",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to delete TLS context with status 500",
		},
		{
			name:   "list server error",
			action: "list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to list TLS contexts with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath:                  tt.handler,
				basePath + "/bad-id":      tt.handler,
				basePath + "/" + testTLSID: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TLSContextClient{
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
				_, err = c.GetTLSContext(ctx, testOrgID, testEnvID, testSGID, "bad-id")
			case "create":
				_, err = c.CreateTLSContext(ctx, testOrgID, testEnvID, testSGID, &TLSContext{Name: "x", Target: "Mule"})
			case "delete":
				err = c.DeleteTLSContext(ctx, testOrgID, testEnvID, testSGID, testTLSID)
			case "list":
				_, err = c.ListTLSContexts(ctx, testOrgID, testEnvID, testSGID)
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
