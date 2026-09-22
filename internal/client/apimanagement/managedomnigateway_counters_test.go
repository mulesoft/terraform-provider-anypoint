package apimanagement

import (
	"context"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// The xapi/v1 gateway view returns null for apiLimit and desiredStatus, so a data
// source reading only that endpoint reports 0 and "" for every gateway. The values live
// on api/v1, which must be consulted to fill them in.
func TestGetManagedOmniGatewayWithCounters_FillsFieldsMissingFromXAPI(t *testing.T) {
	var paths []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		paths = append(paths, r.URL.Path)
		w.Header().Set("Content-Type", "application/json")
		if strings.Contains(r.URL.Path, "/xapi/v1/") {
			_, _ = w.Write([]byte(`{"id":"gw-1","status":"RUNNING","apiLimit":null,"desiredStatus":null}`))
			return
		}
		_, _ = w.Write([]byte(`{"id":"gw-1","status":"APPLIED","apiLimit":50,"desiredStatus":"STARTED"}`))
	}))
	defer srv.Close()

	c := &ManagedOmniGatewayClient{AnypointClient: &client.AnypointClient{BaseURL: srv.URL, HTTPClient: srv.Client()}}

	gw, err := c.GetManagedOmniGatewayWithCounters(context.Background(), "org-1", "env-1", "gw-1")
	if err != nil {
		t.Fatalf("GetManagedOmniGatewayWithCounters: %v", err)
	}

	if gw.APILimit != 50 {
		t.Errorf("api_limit = %d, want 50 (taken from the api/v1 view)", gw.APILimit)
	}
	if gw.DesiredStatus != "STARTED" {
		t.Errorf("desired_status = %q, want STARTED (taken from the api/v1 view)", gw.DesiredStatus)
	}

	// The two endpoints report different things under `status`; hydrating the counters
	// must not drag api/v1's APPLIED over xapi/v1's RUNNING.
	if gw.Status != "RUNNING" {
		t.Errorf("status = %q, want RUNNING — status must stay on the xapi/v1 value", gw.Status)
	}

	if len(paths) != 2 {
		t.Fatalf("expected an xapi/v1 read plus an api/v1 read, got %v", paths)
	}
}

// A gateway that reads fine from xapi/v1 must still be returned when the supplementary
// call fails; losing the whole read over two cosmetic fields would be a regression.
func TestGetManagedOmniGatewayWithCounters_ToleratesCounterFailure(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if strings.Contains(r.URL.Path, "/xapi/v1/") {
			w.Header().Set("Content-Type", "application/json")
			_, _ = w.Write([]byte(`{"id":"gw-1","status":"RUNNING","name":"test"}`))
			return
		}
		w.WriteHeader(http.StatusForbidden)
	}))
	defer srv.Close()

	c := &ManagedOmniGatewayClient{AnypointClient: &client.AnypointClient{BaseURL: srv.URL, HTTPClient: srv.Client()}}

	gw, err := c.GetManagedOmniGatewayWithCounters(context.Background(), "org-1", "env-1", "gw-1")
	if err != nil {
		t.Fatalf("a 403 on the counters call must not fail the read: %v", err)
	}
	if gw.Name != "test" || gw.Status != "RUNNING" {
		t.Errorf("primary gateway fields lost: name=%q status=%q", gw.Name, gw.Status)
	}
}

// A not-found gateway must still surface as not-found rather than being masked by the
// best-effort second call.
func TestGetManagedOmniGatewayWithCounters_PropagatesNotFound(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()

	c := &ManagedOmniGatewayClient{AnypointClient: &client.AnypointClient{BaseURL: srv.URL, HTTPClient: srv.Client()}}

	if _, err := c.GetManagedOmniGatewayWithCounters(context.Background(), "org-1", "env-1", "gw-1"); err == nil {
		t.Fatal("expected a not-found error")
	}
}
