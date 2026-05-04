package secretsmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const testCertID = "test-cert-id"

func TestCertificateClient_CRUD(t *testing.T) {
	mockCert := &CertificateResponse{
		Name: "Test Certificate",
		Type: "PEM",
		Meta: SecretGroupMeta{ID: testCertID},
	}

	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/certificates"
	itemPath := basePath + "/" + testCertID

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", basePath)
				testutil.JSONResponse(w, http.StatusCreated, CreateResponse{ID: testCertID, Message: "created"})
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", basePath)
				testutil.JSONResponse(w, http.StatusOK, []CertificateResponse{*mockCert})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", itemPath)
				testutil.JSONResponse(w, http.StatusOK, mockCert)
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

	c := &CertificateClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateCertificate(ctx, testOrgID, testEnvID, testSGID, &CreateCertificateRequest{
		Name:     "Test Certificate",
		Type:     "PEM",
		CertFile: []byte("-----BEGIN CERTIFICATE-----\ntest\n-----END CERTIFICATE-----"),
	})
	if err != nil {
		t.Fatalf("CreateCertificate() unexpected error: %v", err)
	}
	if created.Meta.ID != testCertID {
		t.Errorf("CreateCertificate() ID = %v, want %v", created.Meta.ID, testCertID)
	}
	if created.Name != "Test Certificate" {
		t.Errorf("CreateCertificate() Name = %v, want 'Test Certificate'", created.Name)
	}

	// Get
	got, err := c.GetCertificate(ctx, testOrgID, testEnvID, testSGID, testCertID)
	if err != nil {
		t.Fatalf("GetCertificate() unexpected error: %v", err)
	}
	if got.Meta.ID != testCertID {
		t.Errorf("GetCertificate() ID = %v, want %v", got.Meta.ID, testCertID)
	}

	// Update
	updated, err := c.UpdateCertificate(ctx, testOrgID, testEnvID, testSGID, testCertID, &CreateCertificateRequest{
		Name:     "Updated Certificate",
		Type:     "PEM",
		CertFile: []byte("-----BEGIN CERTIFICATE-----\nupdated\n-----END CERTIFICATE-----"),
	})
	if err != nil {
		t.Fatalf("UpdateCertificate() unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateCertificate() returned nil")
	}

	// List
	list, err := c.ListCertificates(ctx, testOrgID, testEnvID, testSGID)
	if err != nil {
		t.Fatalf("ListCertificates() unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListCertificates() returned %d items, want 1", len(list))
	}

	// Delete
	if err := c.DeleteCertificate(ctx, testOrgID, testEnvID, testSGID, testCertID); err != nil {
		t.Fatalf("DeleteCertificate() unexpected error: %v", err)
	}
}

func TestCertificateClient_Errors(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups/" + testSGID + "/certificates"

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
			errContains: "certificate not found",
		},
		{
			name:   "get server error",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to get certificate with status 500",
		},
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Bad request")
			},
			errContains: "failed to create certificate with status 400",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to delete certificate with status 500",
		},
		{
			name:   "list server error",
			action: "list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to list certificates with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath:                    tt.handler,
				basePath + "/bad-id":        tt.handler,
				basePath + "/" + testCertID: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &CertificateClient{
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
				_, err = c.GetCertificate(ctx, testOrgID, testEnvID, testSGID, "bad-id")
			case "create":
				_, err = c.CreateCertificate(ctx, testOrgID, testEnvID, testSGID, &CreateCertificateRequest{Name: "x", Type: "PEM"})
			case "delete":
				err = c.DeleteCertificate(ctx, testOrgID, testEnvID, testSGID, testCertID)
			case "list":
				_, err = c.ListCertificates(ctx, testOrgID, testEnvID, testSGID)
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
