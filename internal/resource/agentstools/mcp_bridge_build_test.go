package agentstools

import (
	"encoding/json"
	"reflect"
	"testing"
)

// The expected shapes in this file are LOCKED to a live capture of a real bridge on
// the tf-smg-onefile gateway (see .agents/artifacts/mcp-bridge-onefile-capture.md).
// If the platform contract changes, these tests should fail loudly.

func TestBridgeToolName(t *testing.T) {
	cases := []struct{ method, path, want string }{
		{"GET", "/pets", "get_pets"},
		{"POST", "/pets", "post_pets"},
		{"GET", "/pets/{petId}", "get_pets_petid"},
		{"GET", "/pets/{petId}/vaccinations", "get_pets_petid_vaccinations"},
		{"get", "/Pets", "get_pets"}, // lower-cases everything
	}
	for _, c := range cases {
		if got := bridgeToolName(c.method, c.path); got != c.want {
			t.Errorf("bridgeToolName(%q,%q) = %q, want %q", c.method, c.path, got, c.want)
		}
	}
}

func TestPathURIParamKeys(t *testing.T) {
	got := pathURIParamKeys("/pets/{petId}/vaccinations/{vaxId}")
	want := []string{"petId", "vaxId"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("pathURIParamKeys = %v, want %v", got, want)
	}
	if len(pathURIParamKeys("/pets")) != 0 {
		t.Errorf("no braces should yield empty slice")
	}
}

func TestTranscodingToolConfig(t *testing.T) {
	t.Run("GET with path param", func(t *testing.T) {
		got := transcodingToolConfig(bridgeTool{Method: "get", Path: "/pets/{petId}"})
		if got["name"] != "get_pets_petid" {
			t.Errorf("name = %v", got["name"])
		}
		if got["method"] != "GET" {
			t.Errorf("method = %v, want upper-cased GET", got["method"])
		}
		uri := got["uriParams"].([]map[string]string)
		if len(uri) != 1 || uri[0]["key"] != "petId" || uri[0]["value"] != "#[vars.params['petId']]" {
			t.Errorf("uriParams = %v", uri)
		}
		if _, hasBody := got["body"]; hasBody {
			t.Errorf("GET must not carry a body mapping")
		}
		// empty slices, never nil (must serialize as [])
		if got["queryParams"] == nil || got["headers"] == nil {
			t.Errorf("queryParams/headers must be non-nil slices")
		}
	})

	t.Run("POST with body", func(t *testing.T) {
		got := transcodingToolConfig(bridgeTool{Method: "POST", Path: "/pets", HasBody: true})
		if got["body"] != "#[vars.params.body]" {
			t.Errorf("body = %v, want #[vars.params.body]", got["body"])
		}
	})

	t.Run("query + header params wrapped", func(t *testing.T) {
		got := transcodingToolConfig(bridgeTool{Method: "GET", Path: "/inventory", QueryParams: []string{"category"}, HeaderParams: []string{"X-Warehouse-ID"}})
		q := got["queryParams"].([]map[string]string)
		if len(q) != 1 || q[0]["value"] != "#[vars.params['category']]" {
			t.Errorf("queryParams = %v", q)
		}
		h := got["headers"].([]map[string]string)
		if len(h) != 1 || h[0]["value"] != "#[vars.params['X-Warehouse-ID']]" {
			t.Errorf("headers = %v", h)
		}
	})
}

func TestRouterConfig(t *testing.T) {
	sources := []bridgeSource{
		{Label: "tf-fresh-rest-api", Tools: []bridgeTool{
			{Method: "GET", Path: "/pets"}, {Method: "POST", Path: "/pets", HasBody: true}, {Method: "GET", Path: "/pets/{petId}"},
		}},
		{Label: "tf-ref-petstore-mv", Tools: []bridgeTool{
			{Method: "GET", Path: "/pets/{petId}/vaccinations"},
		}},
	}
	cfg := routerConfig(sources)
	if cfg["transcodingPath"] != "/mcp" {
		t.Errorf("transcodingPath = %v, want /mcp", cfg["transcodingPath"])
	}
	routes := cfg["routes"].([]interface{})
	if len(routes) != 2 {
		t.Fatalf("routes len = %d, want 2", len(routes))
	}
	r0 := routes[0].(map[string]interface{})
	if r0["upstreamName"] != "tf-fresh-rest-api" {
		t.Errorf("route0 upstreamName = %v", r0["upstreamName"])
	}
	names := r0["tools"].([]interface{})
	want := []interface{}{"get_pets", "post_pets", "get_pets_petid"}
	if !reflect.DeepEqual(names, want) {
		t.Errorf("route0 tools = %v, want %v", names, want)
	}
}

func TestToolInputSchema(t *testing.T) {
	t.Run("path param required", func(t *testing.T) {
		s := toolInputSchema(bridgeTool{Method: "GET", Path: "/pets/{petId}"})
		req := s["required"].([]string)
		if len(req) != 1 || req[0] != "petId" {
			t.Errorf("required = %v, want [petId]", req)
		}
		props := s["properties"].(map[string]interface{})
		if _, ok := props["petId"]; !ok {
			t.Errorf("properties missing petId: %v", props)
		}
	})
	t.Run("body required, query optional", func(t *testing.T) {
		s := toolInputSchema(bridgeTool{Method: "POST", Path: "/pets", HasBody: true, QueryParams: []string{"dryRun"}})
		req := s["required"].([]string)
		if len(req) != 1 || req[0] != "body" {
			t.Errorf("required = %v, want [body]", req)
		}
		props := s["properties"].(map[string]interface{})
		if _, ok := props["dryRun"]; !ok {
			t.Errorf("query param dryRun should be an (optional) property")
		}
	})
}

func TestBuildBridgeMetadata(t *testing.T) {
	sources := []bridgeSource{{Label: "a", Tools: []bridgeTool{{Method: "GET", Path: "/pets", Description: "Get pets"}}}}
	md := buildBridgeMetadata("http://0.0.0.0:8085/", sources)
	transport := md["transport"].(map[string]interface{})
	if transport["kind"] != "streamableHttp" || transport["path"] != "http://0.0.0.0:8085/" {
		t.Errorf("transport = %v", transport)
	}
	tools := md["tools"].([]interface{})
	if len(tools) != 1 {
		t.Fatalf("tools len = %d, want 1", len(tools))
	}
	tool := tools[0].(map[string]interface{})
	if tool["name"] != "get_pets" || tool["description"] != "Get pets" {
		t.Errorf("tool = %v", tool)
	}
	// Ensure it serializes and has NO _meta / protocolVersion (matches the live plain file).
	b, _ := json.Marshal(md)
	s := string(b)
	if contains(s, "protocolVersion") || contains(s, "_meta") || contains(s, "capabilities") {
		t.Errorf("metadata must be the simple {transport,tools} shape, got %s", s)
	}
}

func TestBridgeProxyURI(t *testing.T) {
	cases := []struct {
		port     int64
		basePath string
		want     string
	}{
		{8085, "", "http://0.0.0.0:8085/"},
		{0, "", "http://0.0.0.0:8081/"}, // default port
		{8081, "/mcp", "http://0.0.0.0:8081/mcp"},
		{9000, "api/v1", "http://0.0.0.0:9000/api/v1"},
	}
	for _, c := range cases {
		if got := bridgeProxyURI(c.port, c.basePath); got != c.want {
			t.Errorf("bridgeProxyURI(%d,%q) = %q, want %q", c.port, c.basePath, got, c.want)
		}
	}
}

func TestBridgeAssetID(t *testing.T) {
	cases := map[string]string{
		"MCP-bridge-test":  "MCP-bridge-test",
		"My Cool Bridge":   "My-Cool-Bridge",
		"weird!@#name.v2":  "weirdname.v2",
		"orders_bridge-01": "orders_bridge-01",
	}
	for in, want := range cases {
		if got := bridgeAssetID(in); got != want {
			t.Errorf("bridgeAssetID(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestBridgeStructuralSignature(t *testing.T) {
	base := []bridgeSource{{Label: "a", UpstreamURI: "u", AssetID: "x", GroupID: "g", Version: "1.0.0", Tools: []bridgeTool{{Method: "GET", Path: "/a"}}}}
	// Same structure, different tools -> same signature (tools update in place).
	toolsChanged := []bridgeSource{{Label: "a", UpstreamURI: "u", AssetID: "x", GroupID: "g", Version: "1.0.0", Tools: []bridgeTool{{Method: "GET", Path: "/a"}, {Method: "POST", Path: "/b"}}}}
	if bridgeStructuralSignature(base) != bridgeStructuralSignature(toolsChanged) {
		t.Errorf("tools-only change must not alter the structural signature")
	}
	// Structural change (label) -> different signature (requires replace).
	labelChanged := []bridgeSource{{Label: "b", UpstreamURI: "u", AssetID: "x", GroupID: "g", Version: "1.0.0"}}
	if bridgeStructuralSignature(base) == bridgeStructuralSignature(labelChanged) {
		t.Errorf("label change must alter the structural signature")
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (indexOf(s, sub) >= 0)
}

func indexOf(s, sub string) int {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return i
		}
	}
	return -1
}
