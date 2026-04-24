package secretsmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const testKSID = "test-ks-id"

func TestKeystoreClient_CRUD(t *testing.T) {
	mockKS := &KeystoreResponse{
		Name: "Test Keystore",
		Type: "PEM",
		Meta: SecretGroupMeta{ID: testKSID},
	}

	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/keystores"
	itemPath := basePath + "/" + testKSID

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", basePath)
				testutil.JSONResponse(w, http.StatusCreated, CreateResponse{ID: testKSID, Message: "created"})
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", basePath)
				testutil.JSONResponse(w, http.StatusOK, []KeystoreResponse{*mockKS})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", itemPath)
				testutil.JSONResponse(w, http.StatusOK, mockKS)
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

	c := &KeystoreClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateKeystore(ctx, testOrgID, testEnvID, testSGID, &CreateKeystoreRequest{
		Name:        "Test Keystore",
		Type:        "PEM",
		Certificate: []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
		Key:         []byte("-----BEGIN PRIVATE KEY-----\ntest\n-----END PRIVATE KEY-----"),
	})
	if err != nil {
		t.Fatalf("CreateKeystore() unexpected error: %v", err)
	}
	if created.Meta.ID != testKSID {
		t.Errorf("CreateKeystore() ID = %v, want %v", created.Meta.ID, testKSID)
	}
	if created.Name != "Test Keystore" {
		t.Errorf("CreateKeystore() Name = %v, want 'Test Keystore'", created.Name)
	}

	// Get
	got, err := c.GetKeystore(ctx, testOrgID, testEnvID, testSGID, testKSID)
	if err != nil {
		t.Fatalf("GetKeystore() unexpected error: %v", err)
	}
	if got.Meta.ID != testKSID {
		t.Errorf("GetKeystore() ID = %v, want %v", got.Meta.ID, testKSID)
	}

	// Update
	updated, err := c.UpdateKeystore(ctx, testOrgID, testEnvID, testSGID, testKSID, &CreateKeystoreRequest{
		Name:        "Updated Keystore",
		Type:        "PEM",
		Certificate: []byte("-----BEGIN CERTIFICATE-----\nupdated\n-----END CERTIFICATE-----"),
	})
	if err != nil {
		t.Fatalf("UpdateKeystore() unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateKeystore() returned nil")
	}

	// List
	list, err := c.ListKeystores(ctx, testOrgID, testEnvID, testSGID)
	if err != nil {
		t.Fatalf("ListKeystores() unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListKeystores() returned %d items, want 1", len(list))
	}

	// Delete
	if err := c.DeleteKeystore(ctx, testOrgID, testEnvID, testSGID, testKSID); err != nil {
		t.Fatalf("DeleteKeystore() unexpected error: %v", err)
	}
}

func TestKeystoreClient_Errors(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/keystores"

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
			errContains: "keystore not found",
		},
		{
			name:   "get server error",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to get keystore with status 500",
		},
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Bad request")
			},
			errContains: "failed to create keystore with status 400",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to delete keystore with status 500",
		},
		{
			name:   "list server error",
			action: "list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to list keystores with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath:                tt.handler,
				basePath + "/bad-id":    tt.handler,
				basePath + "/" + testKSID: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &KeystoreClient{
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
				_, err = c.GetKeystore(ctx, testOrgID, testEnvID, testSGID, "bad-id")
			case "create":
				_, err = c.CreateKeystore(ctx, testOrgID, testEnvID, testSGID, &CreateKeystoreRequest{Name: "x", Type: "PEM"})
			case "delete":
				err = c.DeleteKeystore(ctx, testOrgID, testEnvID, testSGID, testKSID)
			case "list":
				_, err = c.ListKeystores(ctx, testOrgID, testEnvID, testSGID)
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
