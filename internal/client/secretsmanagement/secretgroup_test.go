package secretsmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const (
	testOrgID = "test-org-id"
	testEnvID = "test-env-id"
	testSGID  = "test-sg-id"
)

func TestSecretGroupClient_CRUD(t *testing.T) {
	mockSG := &SecretGroupResponse{
		Name:         "Test Secret Group",
		Downloadable: true,
		Meta:         SecretGroupMeta{ID: testSGID},
		CurrentState: "ACTIVE",
	}

	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups"
	itemPath := basePath + "/" + testSGID

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.AssertHTTPRequest(t, r, "POST", basePath)
				body := testutil.AssertJSONBody(t, r, "name")
				if body["name"] != "Test Secret Group" {
					t.Errorf("Expected name 'Test Secret Group', got %v", body["name"])
				}
				testutil.JSONResponse(w, http.StatusCreated, CreateResponse{ID: testSGID, Message: "created"})
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", basePath)
				testutil.JSONResponse(w, http.StatusOK, []SecretGroupResponse{*mockSG})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "Method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "GET":
				testutil.AssertHTTPRequest(t, r, "GET", itemPath)
				testutil.JSONResponse(w, http.StatusOK, mockSG)
			case "PATCH":
				testutil.AssertHTTPRequest(t, r, "PATCH", itemPath)
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

	c := &SecretGroupClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateSecretGroup(ctx, testOrgID, testEnvID, &CreateSecretGroupRequest{
		Name:         "Test Secret Group",
		Downloadable: true,
	})
	if err != nil {
		t.Fatalf("CreateSecretGroup() unexpected error: %v", err)
	}
	if created.Meta.ID != testSGID {
		t.Errorf("CreateSecretGroup() ID = %v, want %v", created.Meta.ID, testSGID)
	}
	if created.Name != "Test Secret Group" {
		t.Errorf("CreateSecretGroup() Name = %v, want 'Test Secret Group'", created.Name)
	}

	// Get
	got, err := c.GetSecretGroup(ctx, testOrgID, testEnvID, testSGID)
	if err != nil {
		t.Fatalf("GetSecretGroup() unexpected error: %v", err)
	}
	if got.Meta.ID != testSGID {
		t.Errorf("GetSecretGroup() ID = %v, want %v", got.Meta.ID, testSGID)
	}

	// Update
	updated, err := c.UpdateSecretGroup(ctx, testOrgID, testEnvID, testSGID, &UpdateSecretGroupRequest{
		Name: "Updated Group",
	})
	if err != nil {
		t.Fatalf("UpdateSecretGroup() unexpected error: %v", err)
	}
	if updated == nil {
		t.Fatal("UpdateSecretGroup() returned nil")
	}

	// List
	list, err := c.ListSecretGroups(ctx, testOrgID, testEnvID)
	if err != nil {
		t.Fatalf("ListSecretGroups() unexpected error: %v", err)
	}
	if len(list) != 1 {
		t.Errorf("ListSecretGroups() returned %d items, want 1", len(list))
	}

	// Delete
	if err := c.DeleteSecretGroup(ctx, testOrgID, testEnvID, testSGID); err != nil {
		t.Fatalf("DeleteSecretGroup() unexpected error: %v", err)
	}
}

func TestSecretGroupClient_Errors(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/secretGroups"

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
			errContains: "secret group not found",
		},
		{
			name:   "get server error",
			action: "get",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to get secret group with status 500",
		},
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Bad request")
			},
			errContains: "failed to create secret group with status 400",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to delete secret group with status 500",
		},
		{
			name:   "list server error",
			action: "list",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal error")
			},
			errContains: "failed to list secret groups with status 500",
		},
		{
			name:   "update not found",
			action: "update",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Not found")
			},
			errContains: "secret group not found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath:                  tt.handler,
				basePath + "/bad-id":      tt.handler,
				basePath + "/" + testSGID: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &SecretGroupClient{
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
				_, err = c.GetSecretGroup(ctx, testOrgID, testEnvID, "bad-id")
			case "create":
				_, err = c.CreateSecretGroup(ctx, testOrgID, testEnvID, &CreateSecretGroupRequest{Name: "x"})
			case "delete":
				err = c.DeleteSecretGroup(ctx, testOrgID, testEnvID, testSGID)
			case "list":
				_, err = c.ListSecretGroups(ctx, testOrgID, testEnvID)
			case "update":
				_, err = c.UpdateSecretGroup(ctx, testOrgID, testEnvID, testSGID, &UpdateSecretGroupRequest{Name: "x"})
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
