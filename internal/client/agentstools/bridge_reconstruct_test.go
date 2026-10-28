package agentstools

import (
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

func TestBridgeToolName(t *testing.T) {
	cases := []struct {
		method, path, want string
	}{
		{"GET", "/pets", "get_pets"},
		{"GET", "/pets/{petId}", "get_pets_petid"},
		{"post", "/Pets/Orders", "post_pets_orders"},
		{"DELETE", "/", "delete"},
	}
	for _, c := range cases {
		if got := BridgeToolName(c.method, c.path); got != c.want {
			t.Errorf("BridgeToolName(%q,%q) = %q, want %q", c.method, c.path, got, c.want)
		}
	}
}

func toolDef(name, method, path string, q ...string) map[string]interface{} {
	qp := []interface{}{}
	for _, k := range q {
		qp = append(qp, map[string]interface{}{"key": k, "value": "#[vars.params['" + k + "']]"})
	}
	return map[string]interface{}{"name": name, "method": method, "path": path, "queryParams": qp}
}

// TestReconstructBridgeSources_PerLabel verifies that with two source APIs, each source's
// tools are resolved via the router's label->tool-names map (NOT via upstreamIds, which
// the LIST endpoint drops), and that the group defaults to orgID when the connection omits it.
func TestReconstructBridgeSources_PerLabel(t *testing.T) {
	inst := &MCPBridge{
		Routing: []MCPBridgeRoute{
			{Label: "svc-a", Upstreams: []MCPBridgeRouteUpstream{{ID: "u1"}}},
			{Label: "svc-b", Upstreams: []MCPBridgeRouteUpstream{{ID: "u2"}}},
		},
	}
	labelA, labelB := "svc-a", "svc-b"
	ups := []MCPBridgeUpstreamDetail{
		{ID: "u1", URI: "https://a.example.com", Label: &labelA, Connection: &MCPBridgeConnection{AssetID: "a-api", Version: "1.0.0", Label: "svc-a"}},
		{ID: "u2", URI: "https://b.example.com", Label: &labelB, Connection: &MCPBridgeConnection{AssetID: "b-api", GroupID: "grp-b", Version: "2.0.0", Label: "svc-b"}},
	}
	policies := []apimanagement.APIPolicy{
		{AssetID: "mcp-transcoding-router", ConfigurationData: map[string]interface{}{
			"routes": []interface{}{
				map[string]interface{}{"upstreamName": "svc-a", "tools": []interface{}{"get_pets", "post_pets"}},
				map[string]interface{}{"upstreamName": "svc-b", "tools": []interface{}{"get_orders"}},
			},
		}},
		{AssetID: "mcp-transcoding", UpstreamIDs: nil, ConfigurationData: map[string]interface{}{
			"tools": []interface{}{
				toolDef("get_pets", "GET", "/pets", "limit"),
				toolDef("post_pets", "POST", "/pets"),
				toolDef("get_orders", "GET", "/orders"),
			},
		}},
	}

	sources := ReconstructBridgeSources("org-default", inst, ups, policies)
	if len(sources) != 2 {
		t.Fatalf("expected 2 sources, got %d", len(sources))
	}

	a := sources[0]
	if a.Label != "svc-a" || a.AssetID != "a-api" || a.Version != "1.0.0" {
		t.Errorf("source A metadata = %+v", a)
	}
	if a.GroupID != "org-default" {
		t.Errorf("source A group should default to org, got %q", a.GroupID)
	}
	if len(a.Tools) != 2 || a.Tools[0].Name != "get_pets" || a.Tools[1].Name != "post_pets" {
		t.Fatalf("source A tools = %+v", a.Tools)
	}
	if len(a.Tools[0].QueryParams) != 1 || a.Tools[0].QueryParams[0] != "limit" {
		t.Errorf("get_pets query params = %v, want [limit]", a.Tools[0].QueryParams)
	}

	b := sources[1]
	if b.GroupID != "grp-b" {
		t.Errorf("source B group = %q, want grp-b", b.GroupID)
	}
	if len(b.Tools) != 1 || b.Tools[0].Name != "get_orders" {
		t.Fatalf("source B tools = %+v", b.Tools)
	}
}

// TestReconstructBridgeSources_SingleSourceFallback verifies that when the router yields
// no per-label mapping and there is exactly one upstream, all pooled tool defs are assigned
// to it in policy order (order-stable, no post-import churn).
func TestReconstructBridgeSources_SingleSourceFallback(t *testing.T) {
	inst := &MCPBridge{}
	ups := []MCPBridgeUpstreamDetail{
		{ID: "u1", URI: "https://only.example.com", Connection: &MCPBridgeConnection{AssetID: "only-api", Version: "1.0.0", Label: "only"}},
	}
	policies := []apimanagement.APIPolicy{
		{AssetID: "mcp-transcoding", ConfigurationData: map[string]interface{}{
			"tools": []interface{}{
				toolDef("get_a", "GET", "/a"),
				toolDef("get_b", "GET", "/b"),
			},
		}},
	}

	sources := ReconstructBridgeSources("org-default", inst, ups, policies)
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	if got := sources[0].Tools; len(got) != 2 || got[0].Name != "get_a" || got[1].Name != "get_b" {
		t.Fatalf("fallback tools = %+v, want [get_a get_b] in order", got)
	}
	if sources[0].Label != "only" {
		t.Errorf("label should fall back to the connection label, got %q", sources[0].Label)
	}
}
