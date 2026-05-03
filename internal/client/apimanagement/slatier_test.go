package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
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
