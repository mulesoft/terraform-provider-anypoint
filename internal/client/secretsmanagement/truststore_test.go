package secretsmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const testTSID = "test-ts-id"

func TestTruststoreClient_CRUD(t *testing.T) {
	mockTS := &TruststoreResponse{
		Name: "Test Truststore",
		Type: "PEM",
		Meta: SecretGroupMeta{ID: testTSID},
	}

	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/truststores"
	itemPath := basePath + "/" + testTSID

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", basePath)
				testutil.JSONResponse(w, http.StatusCreated, CreateResponse{ID: testTSID, Message: "created"})
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", basePath)
				testutil.JSONResponse(w, http.StatusOK, []TruststoreResponse{*mockTS})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", itemPath)
				testutil.JSONResponse(w, http.StatusOK, mockTS)
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

	c := &TruststoreClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateTruststore(ctx, testOrgID, testEnvID, testSGID, &CreateTruststoreRequest{
		Name:       "Test Truststore",
		Type:       "PEM",
		TrustStore: []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
	})
	if err != nil {
		t.Fatalf("CreateTruststore() unexpected error: %v", err)
	}
	if created.Meta.ID != testTSID {
		t.Errorf("CreateTruststore() ID = %v, want %v", created.Meta.ID, testTSID)
	}
	if created.Name != "Test Truststore" {
		t.Errorf("CreateTruststore() Name = %v, want 'Test Truststore'", created.Name)
	}

	// Get
	got, err := c.GetTruststore(ctx, testOrgID, testEnvID, testSGID, testTSID)
	if err != nil {
		t.Fatalf("GetTruststore() unexpected error: %v", err)
	}
	if got.Meta.ID != testTSID {
		t.Errorf("GetTruststore() ID = %v, want %v", got.Meta.ID, testTSID)
	}

	// Update
	updated, err := c.UpdateTruststore(ctx, testOrgID, testEnvID, testSGID, testTSID, &CreateTruststoreRequest{
		Name:       "Updated Truststore",
		Type:       "PEM",
		TrustStore: []byte("-----BEGIN CERTIFICATE-----\nupdated\n-----END CERTIFICATE-----"),
	})
	if err != nil {
		t.Fatalf("UpdateTruststore() unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateTruststore() returned nil")
	}

	// List
	list, err := c.ListTruststores(ctx, testOrgID, testEnvID, testSGID)
	if err != nil {
		t.Fatalf("ListTruststores() unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListTruststores() returned %d items, want 1", len(list))
	}

	// Delete
	if err := c.DeleteTruststore(ctx, testOrgID, testEnvID, testSGID, testTSID); err != nil {
		t.Fatalf("DeleteTruststore() unexpected error: %v", err)
	}
}

func TestTruststoreClient_Errors(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/truststores"

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
			errContains: "truststore not found",
		},
		{
			name:   "get server error",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to get truststore with status 500",
		},
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Bad request")
			},
			errContains: "failed to create truststore with status 400",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to delete truststore with status 500",
		},
		{
			name:   "list server error",
			action: "list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to list truststores with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath:                tt.handler,
				basePath + "/bad-id":    tt.handler,
				basePath + "/" + testTSID: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TruststoreClient{
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
				_, err = c.GetTruststore(ctx, testOrgID, testEnvID, testSGID, "bad-id")
			case "create":
				_, err = c.CreateTruststore(ctx, testOrgID, testEnvID, testSGID, &CreateTruststoreRequest{Name: "x", Type: "PEM"})
			case "delete":
				err = c.DeleteTruststore(ctx, testOrgID, testEnvID, testSGID, testTSID)
			case "list":
				_, err = c.ListTruststores(ctx, testOrgID, testEnvID, testSGID)
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
