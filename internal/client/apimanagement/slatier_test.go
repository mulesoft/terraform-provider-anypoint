package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestSLATierClient_ListNotFound verifies that a 404 from the SLA tier list
// endpoint (e.g. when the parent API instance has been deleted out-of-band)
// is surfaced as a NotFoundError. Resource Read handlers use this signal to
// silently drop the resource from state instead of erroring.
func TestSLATierClient_ListNotFound(t *testing.T) {
	handlers := testutil.StandardMockHandlers()

	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/9999/tiers"] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}

	server := testutil.MockHTTPServer(t, handlers)

	c := &SLATierClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	t.Run("List returns NotFound on 404", func(t *testing.T) {
		_, err := c.ListSLATiers(context.Background(), "org-123", "env-456", 9999)
		if err == nil {
			t.Fatal("expected error for missing parent API, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("expected IsNotFound==true, got err=%v", err)
		}
	})

	t.Run("Get returns NotFound on 404", func(t *testing.T) {
		_, err := c.GetSLATier(context.Background(), "org-123", "env-456", 9999, 1234)
		if err == nil {
			t.Fatal("expected error for missing parent API, got nil")
		}
		if !client.IsNotFound(err) {
			t.Errorf("expected IsNotFound==true, got err=%v", err)
		}
	})
}

// TestSLATierClient_ListSurfacesOtherErrors verifies that non-404 list
// failures are still propagated as generic errors (not NotFound).
func TestSLATierClient_ListSurfacesOtherErrors(t *testing.T) {
	handlers := testutil.StandardMockHandlers()

	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/tiers"] = func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}

	server := testutil.MockHTTPServer(t, handlers)

	c := &SLATierClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	_, err := c.ListSLATiers(context.Background(), "org-123", "env-456", 100)
	if err == nil {
		t.Fatal("expected error on 500, got nil")
	}
	if client.IsNotFound(err) {
		t.Errorf("did not expect NotFound for 500 response, got %v", err)
	}
}

// TestSLATierClient_GetReturnsNotFoundWhenAbsentFromList preserves the
// existing behaviour: a successful list that simply doesn't contain the
// requested tier ID still yields NotFound.
func TestSLATierClient_GetReturnsNotFoundWhenAbsentFromList(t *testing.T) {
	handlers := testutil.StandardMockHandlers()

	handlers["GET /apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/tiers"] = func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(struct {
			Tiers []SLATier `json:"tiers"`
		}{Tiers: []SLATier{{ID: 1, Name: "Bronze"}}})
	}

	server := testutil.MockHTTPServer(t, handlers)

	c := &SLATierClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: server.Client(),
		},
	}

	_, err := c.GetSLATier(context.Background(), "org-123", "env-456", 100, 9999)
	if err == nil {
		t.Fatal("expected NotFound when tier id absent from list")
	}
	if !client.IsNotFound(err) {
		t.Errorf("expected IsNotFound==true, got %v", err)
	}
}

func TestSLATierClient_CRUD(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/tiers"
	itemPath := basePath + "/42"

	mockTier := &SLATier{
		ID:          42,
		Name:        "Gold",
		Description: "Gold tier",
		AutoApprove: true,
		Status:      "ACTIVE",
		Limits: []SLALimit{
			{TimePeriodInMilliseconds: 1000, MaximumRequests: 100, Visible: true},
		},
	}
	updatedTier := *mockTier
	updatedTier.Name = "Platinum"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "POST":
				testutil.JSONResponse(w, http.StatusCreated, mockTier)
			case "GET":
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(struct {
					Tiers []SLATier `json:"tiers"`
				}{Tiers: []SLATier{*mockTier}})
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		},
		itemPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case "PUT":
				testutil.JSONResponse(w, http.StatusOK, &updatedTier)
			case "DELETE":
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "method not allowed")
			}
		},
	}

	server := testutil.MockHTTPServer(t, handlers)
	c := &SLATierClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "test-token",
			HTTPClient: &http.Client{},
		},
	}
	ctx := context.Background()

	// Create
	created, err := c.CreateSLATier(ctx, "org-123", "env-456", 100, &CreateSLATierRequest{
		Name:        "Gold",
		Description: "Gold tier",
		AutoApprove: true,
		Limits:      []SLALimit{{TimePeriodInMilliseconds: 1000, MaximumRequests: 100, Visible: true}},
	})
	if err != nil {
		t.Fatalf("CreateSLATier() unexpected error: %v", err)
	}
	if created.ID != 42 {
		t.Errorf("CreateSLATier() ID = %d, want 42", created.ID)
	}
	if created.Name != "Gold" {
		t.Errorf("CreateSLATier() Name = %q, want Gold", created.Name)
	}

	// Get (via List+filter)
	got, err := c.GetSLATier(ctx, "org-123", "env-456", 100, 42)
	if err != nil {
		t.Fatalf("GetSLATier() unexpected error: %v", err)
	}
	if got.ID != 42 {
		t.Errorf("GetSLATier() ID = %d, want 42", got.ID)
	}

	// Update
	autoApprove := false
	updated, err := c.UpdateSLATier(ctx, "org-123", "env-456", 100, 42, &UpdateSLATierRequest{
		Name:        "Platinum",
		AutoApprove: &autoApprove,
	})
	if err != nil {
		t.Fatalf("UpdateSLATier() unexpected error: %v", err)
	}
	if updated.Name != "Platinum" {
		t.Errorf("UpdateSLATier() Name = %q, want Platinum", updated.Name)
	}

	// Delete
	if err := c.DeleteSLATier(ctx, "org-123", "env-456", 100, 42); err != nil {
		t.Fatalf("DeleteSLATier() unexpected error: %v", err)
	}
}

func TestSLATierClient_Errors(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/tiers"
	itemPath := basePath + "/42"

	tests := []struct {
		name        string
		action      string
		handler     func(w http.ResponseWriter, r *http.Request)
		errContains string
	}{
		{
			name:   "create server error",
			action: "create",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "bad request")
			},
			errContains: "failed to create SLA tier with status 400",
		},
		{
			name:   "update not found",
			action: "update",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
			errContains: "SLA tier not found",
		},
		{
			name:   "update server error",
			action: "update",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
			},
			errContains: "failed to update SLA tier with status 500",
		},
		{
			name:   "delete server error",
			action: "delete",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
			},
			errContains: "failed to delete SLA tier with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath: tt.handler,
				itemPath: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)
			c := &SLATierClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "test-token",
					HTTPClient: &http.Client{},
				},
			}
			ctx := context.Background()

			var err error
			autoApprove := true
			switch tt.action {
			case "create":
				_, err = c.CreateSLATier(ctx, "org-123", "env-456", 100, &CreateSLATierRequest{Name: "x"})
			case "update":
				_, err = c.UpdateSLATier(ctx, "org-123", "env-456", 100, 42, &UpdateSLATierRequest{Name: "x", AutoApprove: &autoApprove})
			case "delete":
				err = c.DeleteSLATier(ctx, "org-123", "env-456", 100, 42)
			}

			if err == nil {
				t.Fatal("Expected error, got nil")
			}
			if !strings.Contains(err.Error(), tt.errContains) {
				t.Errorf("Error = %v, want containing %q", err, tt.errContains)
			}
		})
	}
}

