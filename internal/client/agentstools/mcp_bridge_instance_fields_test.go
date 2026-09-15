package agentstools

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// A nil pointer means "leave this field alone" and must not appear in the request body,
// otherwise an unrelated update would silently clear a label the practitioner set.
func TestBridgeInstanceFields_OmitsUntouchedFields(t *testing.T) {
	label := "prod-bridge"
	body, err := json.Marshal(BridgeInstanceFields{InstanceLabel: &label}.payload())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(body), `{"instanceLabel":"prod-bridge"}`; got != want {
		t.Errorf("payload = %s, want %s (approvalMethod/endpointUri must not be sent)", got, want)
	}
}

// Clearing a value has to reach the wire as JSON null. Sending "" would store an empty
// string, which the platform reports back as a blank label rather than no label at all.
func TestBridgeInstanceFields_ClearsWithExplicitNull(t *testing.T) {
	empty := ""
	body, err := json.Marshal(BridgeInstanceFields{ApprovalMethod: &empty}.payload())
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if got, want := string(body), `{"approvalMethod":null}`; got != want {
		t.Errorf("payload = %s, want %s", got, want)
	}
}

func TestBridgeInstanceFields_PayloadCarriesEveryField(t *testing.T) {
	label, approval, endpoint, provider := "lbl", "manual", "https://gw.example.com/mcp", "prov-1"
	got := BridgeInstanceFields{
		InstanceLabel:  &label,
		ApprovalMethod: &approval,
		EndpointURI:    &endpoint,
		ProviderID:     &provider,
	}.payload()

	want := map[string]interface{}{
		"instanceLabel":  "lbl",
		"approvalMethod": "manual",
		"endpointUri":    "https://gw.example.com/mcp",
		"providerId":     "prov-1",
	}
	for k, v := range want {
		if got[k] != v {
			t.Errorf("payload[%q] = %v, want %v", k, got[k], v)
		}
	}
	if len(got) != len(want) {
		t.Errorf("payload has %d keys, want %d: %v", len(got), len(want), got)
	}
}

// The live contract: both api/v1 and xapi/v1 accept these fields, and the provider uses
// api/v1. Guard the method and path so a refactor cannot quietly move it to an endpoint
// that does not persist them.
func TestUpdateBridgeInstanceFields_PatchesInstance(t *testing.T) {
	var gotMethod, gotPath string
	var gotBody map[string]interface{}

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotMethod, gotPath = r.Method, r.URL.Path
		raw, _ := io.ReadAll(r.Body)
		_ = json.Unmarshal(raw, &gotBody)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42,"instanceLabel":"prod-bridge"}`))
	}))
	defer srv.Close()

	c := &MCPBridgeClient{AnypointClient: &client.AnypointClient{BaseURL: srv.URL, HTTPClient: srv.Client()}}

	label := "prod-bridge"
	bridge, err := c.UpdateBridgeInstanceFields(context.Background(), "org-1", "env-1", 42,
		BridgeInstanceFields{InstanceLabel: &label})
	if err != nil {
		t.Fatalf("UpdateBridgeInstanceFields: %v", err)
	}

	if gotMethod != http.MethodPatch {
		t.Errorf("method = %s, want PATCH", gotMethod)
	}
	if want := "/apimanager/api/v1/organizations/org-1/environments/env-1/apis/42"; gotPath != want {
		t.Errorf("path = %s, want %s", gotPath, want)
	}
	if gotBody["instanceLabel"] != "prod-bridge" {
		t.Errorf("body = %v, want instanceLabel prod-bridge", gotBody)
	}
	if bridge.InstanceLabel != "prod-bridge" {
		t.Errorf("returned label = %q, want prod-bridge", bridge.InstanceLabel)
	}
}

// With nothing to change the client must not issue a PATCH at all — an empty body would
// be a pointless write against the instance on every no-op apply.
func TestUpdateBridgeInstanceFields_SkipsPatchWhenEmpty(t *testing.T) {
	var methods []string

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		methods = append(methods, r.Method)
		w.Header().Set("Content-Type", "application/json")
		_, _ = w.Write([]byte(`{"id":42}`))
	}))
	defer srv.Close()

	c := &MCPBridgeClient{AnypointClient: &client.AnypointClient{BaseURL: srv.URL, HTTPClient: srv.Client()}}

	if _, err := c.UpdateBridgeInstanceFields(context.Background(), "org-1", "env-1", 42,
		BridgeInstanceFields{}); err != nil {
		t.Fatalf("UpdateBridgeInstanceFields: %v", err)
	}

	for _, m := range methods {
		if m == http.MethodPatch {
			t.Error("no fields changed, so the client must fall back to a read instead of PATCHing")
		}
	}
}
