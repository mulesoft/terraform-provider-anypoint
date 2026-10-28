package agentstools

import (
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// transcodingPolicy wraps a tools[] definition the way the live policy carries it.
func transcodingPolicy(tools ...map[string]interface{}) apimanagement.APIPolicy {
	items := make([]interface{}, 0, len(tools))
	for _, t := range tools {
		items = append(items, t)
	}
	return apimanagement.APIPolicy{
		AssetID:           bridgeTranscodingAsset,
		ConfigurationData: map[string]interface{}{"tools": items},
	}
}

func paramEntries(pairs ...[2]string) []interface{} {
	out := make([]interface{}, 0, len(pairs))
	for _, p := range pairs {
		out = append(out, map[string]interface{}{"key": p[0], "value": p[1]})
	}
	return out
}

// A plain identity mapping must be reported as derived, so import emits the concise
// query_params form rather than a wall of boilerplate identical to the default.
func TestHasCustomHTTPMapping_IdentityMappingIsNotCustom(t *testing.T) {
	policies := []apimanagement.APIPolicy{transcodingPolicy(map[string]interface{}{
		"name":        "get_pet",
		"method":      "GET",
		"path":        "/pets/{petId}",
		"queryParams": paramEntries([2]string{"verbose", "#[vars.params['verbose']]"}),
		"uriParams":   paramEntries([2]string{"petId", "#[vars.params['petId']]"}),
		"headers":     paramEntries(),
	})}

	tools := reconstructToolsFromPolicies(t, policies)
	if tools[0].HasCustomHTTPMapping() {
		t.Error("an identity mapping is exactly what the parameter lists generate; it must not be reported as custom")
	}
}

// A renamed parameter cannot be expressed by the concise form, so it must be reported as
// custom or the value would be silently lost on import.
func TestHasCustomHTTPMapping_RenamedParameterIsCustom(t *testing.T) {
	policies := []apimanagement.APIPolicy{transcodingPolicy(map[string]interface{}{
		"name":        "search",
		"method":      "GET",
		"path":        "/search",
		"queryParams": paramEntries([2]string{"query", "#[vars.params['q']]"}),
		"uriParams":   paramEntries(),
		"headers":     paramEntries(),
	})}

	tools := reconstructToolsFromPolicies(t, policies)
	if !tools[0].HasCustomHTTPMapping() {
		t.Error("an HTTP name fed by a differently-named tool parameter must be reported as custom")
	}
	if len(tools[0].QueryMapping) != 1 || tools[0].QueryMapping[0].Value != "#[vars.params['q']]" {
		t.Errorf("the DataWeave expression must be preserved, got %v", tools[0].QueryMapping)
	}
}

func TestHasCustomHTTPMapping_CustomBodyIsCustom(t *testing.T) {
	policies := []apimanagement.APIPolicy{transcodingPolicy(map[string]interface{}{
		"name": "create_pet", "method": "POST", "path": "/pets",
		"queryParams": paramEntries(), "uriParams": paramEntries(), "headers": paramEntries(),
		"body": "#[vars.params.payload]",
	})}

	tools := reconstructToolsFromPolicies(t, policies)
	if !tools[0].HasCustomHTTPMapping() {
		t.Error("a body expression other than the default must be reported as custom")
	}
}

func TestHasCustomHTTPMapping_DefaultBodyIsNotCustom(t *testing.T) {
	policies := []apimanagement.APIPolicy{transcodingPolicy(map[string]interface{}{
		"name": "create_pet", "method": "POST", "path": "/pets",
		"queryParams": paramEntries(), "uriParams": paramEntries(), "headers": paramEntries(),
		"body": "#[vars.params.body]",
	})}

	tools := reconstructToolsFromPolicies(t, policies)
	if tools[0].HasCustomHTTPMapping() {
		t.Error("the default body expression is what has_body generates; it must not be reported as custom")
	}
}

// uriParams are derived from the path, so a mapping whose keys no longer match the
// placeholders cannot be reproduced from the path alone.
func TestHasCustomHTTPMapping_URIKeysNotMatchingPathIsCustom(t *testing.T) {
	policies := []apimanagement.APIPolicy{transcodingPolicy(map[string]interface{}{
		"name": "get_pet", "method": "GET", "path": "/pets/{petId}",
		"queryParams": paramEntries(), "headers": paramEntries(),
		"uriParams": paramEntries([2]string{"id", "#[vars.params['id']]"}),
	})}

	tools := reconstructToolsFromPolicies(t, policies)
	if !tools[0].HasCustomHTTPMapping() {
		t.Error("uri params whose keys do not match the path placeholders must be reported as custom")
	}
}

func reconstructToolsFromPolicies(t *testing.T, policies []apimanagement.APIPolicy) []ReconstructedTool {
	t.Helper()
	inst := &MCPBridge{Routing: []MCPBridgeRoute{{Label: "src", Upstreams: []MCPBridgeRouteUpstream{{ID: "u1"}}}}}
	ups := []MCPBridgeUpstreamDetail{{ID: "u1", URI: "https://backend.example.com"}}

	sources := ReconstructBridgeSources("org-1", inst, ups, policies)
	if len(sources) != 1 || len(sources[0].Tools) == 0 {
		t.Fatalf("expected one source with tools, got %+v", sources)
	}
	return sources[0].Tools
}
