package agentstools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	agentsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

// --- technologyToAPI / technologyFromAPI (agent_instance.go) ---

func TestAgentInstance_technologyToAPI(t *testing.T) {
	tests := []struct{ in, want string }{
		{"omniGateway", "flexGateway"},
		{"flexGateway", "flexGateway"},
		{"mule4", "mule4"},
		{"", ""},
	}
	for _, tt := range tests {
		got := technologyToAPI(tt.in)
		if got != tt.want {
			t.Errorf("technologyToAPI(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

func TestAgentInstance_technologyFromAPI(t *testing.T) {
	tests := []struct{ in, want string }{
		{"flexGateway", "omniGateway"},
		{"omniGateway", "omniGateway"},
		{"mule4", "mule4"},
		{"", ""},
	}
	for _, tt := range tests {
		got := technologyFromAPI(tt.in)
		if got != tt.want {
			t.Errorf("technologyFromAPI(%q) = %q, want %q", tt.in, got, tt.want)
		}
	}
}

// --- AgentInstanceResource.mergeUpstreamIDs ---

func TestAgentInstanceResource_mergeUpstreamIDs(t *testing.T) {
	r := &AgentInstanceResource{}

	t.Run("copies IDs by URI match", func(t *testing.T) {
		current := []agentsclient.AgentInstanceRoute{
			{Upstreams: []agentsclient.AgentInstanceUpstream{
				{ID: "srv-id-1", URI: "https://a.example.com"},
				{ID: "srv-id-2", URI: "https://b.example.com"},
			}},
		}
		update := []agentsclient.AgentInstanceRoute{
			{Upstreams: []agentsclient.AgentInstanceUpstream{
				{URI: "https://a.example.com"},
				{URI: "https://b.example.com"},
			}},
		}
		r.mergeUpstreamIDs(current, update)
		if update[0].Upstreams[0].ID != "srv-id-1" {
			t.Errorf("Upstream[0].ID = %q, want srv-id-1", update[0].Upstreams[0].ID)
		}
		if update[0].Upstreams[0].URI != "" {
			t.Errorf("URI should be cleared after ID merge, got %q", update[0].Upstreams[0].URI)
		}
		if update[0].Upstreams[1].ID != "srv-id-2" {
			t.Errorf("Upstream[1].ID = %q, want srv-id-2", update[0].Upstreams[1].ID)
		}
	})

	t.Run("falls back to positional merge when URI not found", func(t *testing.T) {
		current := []agentsclient.AgentInstanceRoute{
			{Upstreams: []agentsclient.AgentInstanceUpstream{
				{ID: "pos-id", URI: "https://old.example.com"},
			}},
		}
		update := []agentsclient.AgentInstanceRoute{
			{Upstreams: []agentsclient.AgentInstanceUpstream{
				{URI: "https://new.example.com"},
			}},
		}
		r.mergeUpstreamIDs(current, update)
		// URI fallback: positional match sets the ID and clears URI
		if update[0].Upstreams[0].ID != "pos-id" {
			t.Errorf("Upstream[0].ID = %q, want pos-id (positional fallback)", update[0].Upstreams[0].ID)
		}
	})

	t.Run("empty current does nothing", func(t *testing.T) {
		update := []agentsclient.AgentInstanceRoute{
			{Upstreams: []agentsclient.AgentInstanceUpstream{{URI: "https://a.example.com"}}},
		}
		r.mergeUpstreamIDs(nil, update)
		if update[0].Upstreams[0].ID != "" {
			t.Errorf("Expected empty ID, got %q", update[0].Upstreams[0].ID)
		}
	})

	t.Run("update has more routes than current, no panic", func(t *testing.T) {
		current := []agentsclient.AgentInstanceRoute{
			{Upstreams: []agentsclient.AgentInstanceUpstream{{ID: "id-1", URI: "https://a.example.com"}}},
		}
		update := []agentsclient.AgentInstanceRoute{
			{Upstreams: []agentsclient.AgentInstanceUpstream{{URI: "https://a.example.com"}}},
			{Upstreams: []agentsclient.AgentInstanceUpstream{{URI: "https://b.example.com"}}},
		}
		r.mergeUpstreamIDs(current, update)
		if update[0].Upstreams[0].ID != "id-1" {
			t.Errorf("First route ID = %q, want id-1", update[0].Upstreams[0].ID)
		}
		if update[1].Upstreams[0].ID != "" {
			t.Error("Extra route should not get an ID")
		}
	})
}

// --- AgentInstanceResource.expandRouting ---

func TestAgentInstanceResource_expandRouting(t *testing.T) {
	r := &AgentInstanceResource{}
	ctx := context.Background()

	t.Run("null list returns nil", func(t *testing.T) {
		result := r.expandRouting(ctx, types.ListNull(types.StringType))
		if result != nil {
			t.Errorf("expandRouting(null) = %v, want nil", result)
		}
	})

	t.Run("unknown list returns nil", func(t *testing.T) {
		result := r.expandRouting(ctx, types.ListUnknown(types.StringType))
		if result != nil {
			t.Errorf("expandRouting(unknown) = %v, want nil", result)
		}
	})
}

// --- AgentInstanceResource.expandCreateRequest ---

func TestAgentInstanceResource_expandCreateRequest(t *testing.T) {
	r := &AgentInstanceResource{}
	ctx := context.Background()

	t.Run("technology omniGateway is mapped to flexGateway", func(t *testing.T) {
		data := AgentInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(agentRouteListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
			UpstreamURI:      types.StringNull(),
		}
		req := r.expandCreateRequest(ctx, data)
		if req.Technology != "flexGateway" {
			t.Errorf("Technology = %q, want flexGateway", req.Technology)
		}
	})

	t.Run("upstream_uri creates single-route routing with weight 100", func(t *testing.T) {
		data := AgentInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringValue("https://agent-backend.example.com"),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(agentRouteListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandCreateRequest(ctx, data)
		if len(req.Routing) != 1 {
			t.Fatalf("Routing len = %d, want 1", len(req.Routing))
		}
		if len(req.Routing[0].Upstreams) != 1 {
			t.Fatalf("Upstreams len = %d, want 1", len(req.Routing[0].Upstreams))
		}
		if req.Routing[0].Upstreams[0].URI != "https://agent-backend.example.com" {
			t.Errorf("Upstream URI = %q", req.Routing[0].Upstreams[0].URI)
		}
		if req.Routing[0].Upstreams[0].Weight != 100 {
			t.Errorf("Upstream Weight = %d, want 100", req.Routing[0].Upstreams[0].Weight)
		}
	})

	t.Run("endpoint with base_path generates a2a type and proxy URI", func(t *testing.T) {
		epObj, _ := types.ObjectValue(endpointAttrTypes, map[string]attr.Value{
			"deployment_type":  types.StringValue("HY"),
			"type":             types.StringValue("http"),
			"base_path":        types.StringValue("/a2a/v1"),
			"uri":              types.StringNull(),
			"response_timeout": types.Int64Null(),
		})
		data := AgentInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         epObj,
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(agentRouteListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandCreateRequest(ctx, data)
		if req.Endpoint == nil {
			t.Fatal("Endpoint should not be nil")
		}
		if req.Endpoint.Type != "a2a" {
			t.Errorf("Endpoint.Type = %q, want a2a", req.Endpoint.Type)
		}
		if req.Endpoint.ProxyURI == nil || *req.Endpoint.ProxyURI != "http://0.0.0.0:8081/a2a/v1" {
			t.Errorf("ProxyURI = %v, want http://0.0.0.0:8081/a2a/v1", req.Endpoint.ProxyURI)
		}
	})

	t.Run("spec is included when set", func(t *testing.T) {
		data := AgentInstanceResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(agentRouteListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
			Spec: &SpecModel{
				AssetID: types.StringValue("my-agent"),
				GroupID: types.StringValue("com.example"),
				Version: types.StringValue("1.0.0"),
			},
		}
		req := r.expandCreateRequest(ctx, data)
		if req.Spec == nil {
			t.Fatal("Spec should not be nil")
		}
		if req.Spec.AssetID != "my-agent" {
			t.Errorf("Spec.AssetID = %q, want my-agent", req.Spec.AssetID)
		}
	})
}

// agentRouteListElemType mirrors the schema-level route object type for null list construction.
var agentRouteListElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"label":     types.StringType,
		"rules":     types.ObjectType{AttrTypes: agentRulesAttrTypes},
		"upstreams": types.ListType{ElemType: agentUpstreamObjType},
	},
}

var agentRulesAttrTypes = map[string]attr.Type{
	"methods": types.StringType,
	"path":    types.StringType,
	"host":    types.StringType,
	"headers": types.MapType{ElemType: types.StringType},
}

var agentUpstreamObjType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"weight":         types.Int64Type,
		"uri":            types.StringType,
		"label":          types.StringType,
		"tls_context_id": types.StringType,
	},
}
