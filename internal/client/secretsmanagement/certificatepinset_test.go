package secretsmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const testPinID = "test-pin-id"

func TestCertificatePinsetClient_CRUD(t *testing.T) {
	mockPin := &CertificatePinsetResponse{
		Name: "Test Pinset",
		Meta: SecretGroupMeta{ID: testPinID},
	}

	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/certificatePinsets"
	itemPath := basePath + "/" + testPinID

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", basePath)
				testutil.JSONResponse(w, http.StatusCreated, CreateResponse{ID: testPinID, Message: "created"})
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", basePath)
				testutil.JSONResponse(w, http.StatusOK, []CertificatePinsetResponse{*mockPin})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", itemPath)
				testutil.JSONResponse(w, http.StatusOK, mockPin)
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

	c := &CertificatePinsetClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateCertificatePinset(ctx, testOrgID, testEnvID, testSGID, &CreateCertificatePinsetRequest{
		Name:    "Test Pinset",
		PinFile: []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
	})
	if err != nil {
		t.Fatalf("CreateCertificatePinset() unexpected error: %v", err)
	}
	if created.Meta.ID != testPinID {
		t.Errorf("CreateCertificatePinset() ID = %v, want %v", created.Meta.ID, testPinID)
	}
	if created.Name != "Test Pinset" {
		t.Errorf("CreateCertificatePinset() Name = %v, want 'Test Pinset'", created.Name)
	}

	// Get
	got, err := c.GetCertificatePinset(ctx, testOrgID, testEnvID, testSGID, testPinID)
	if err != nil {
		t.Fatalf("GetCertificatePinset() unexpected error: %v", err)
	}
	if got.Meta.ID != testPinID {
		t.Errorf("GetCertificatePinset() ID = %v, want %v", got.Meta.ID, testPinID)
	}

	// Update
	updated, err := c.UpdateCertificatePinset(ctx, testOrgID, testEnvID, testSGID, testPinID, &CreateCertificatePinsetRequest{
		Name:    "Updated Pinset",
		PinFile: []byte("-----BEGIN CERTIFICATE-----\nupdated\n-----END CERTIFICATE-----"),
	})
	if err != nil {
		t.Fatalf("UpdateCertificatePinset() unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateCertificatePinset() returned nil")
	}

	// List
	list, err := c.ListCertificatePinsets(ctx, testOrgID, testEnvID, testSGID)
	if err != nil {
		t.Fatalf("ListCertificatePinsets() unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListCertificatePinsets() returned %d items, want 1", len(list))
	}

	// Delete
	if err := c.DeleteCertificatePinset(ctx, testOrgID, testEnvID, testSGID, testPinID); err != nil {
		t.Fatalf("DeleteCertificatePinset() unexpected error: %v", err)
	}
}

func TestCertificatePinsetClient_Errors(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/certificatePinsets"

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
			errContains: "certificate pinset not found",
		},
		{
			name:   "get server error",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to get certificate pinset with status 500",
		},
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Bad request")
			},
			errContains: "failed to create certificate pinset with status 400",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to delete certificate pinset with status 500",
		},
		{
			name:   "list server error",
			action: "list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to list certificate pinsets with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath:                   tt.handler,
				basePath + "/bad-id":       tt.handler,
				basePath + "/" + testPinID: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &CertificatePinsetClient{
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
				_, err = c.GetCertificatePinset(ctx, testOrgID, testEnvID, testSGID, "bad-id")
			case "create":
				_, err = c.CreateCertificatePinset(ctx, testOrgID, testEnvID, testSGID, &CreateCertificatePinsetRequest{Name: "x"})
			case "delete":
				err = c.DeleteCertificatePinset(ctx, testOrgID, testEnvID, testSGID, testPinID)
			case "list":
				_, err = c.ListCertificatePinsets(ctx, testOrgID, testEnvID, testSGID)
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
