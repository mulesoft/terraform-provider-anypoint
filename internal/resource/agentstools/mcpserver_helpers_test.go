package agentstools

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	agentsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
)

// --- MCPServerResource.mergeUpstreamIDs ---

func TestMCPServerResource_mergeUpstreamIDs(t *testing.T) {
	r := &MCPServerResource{}

	t.Run("copies IDs by URI match and clears URI", func(t *testing.T) {
		current := []agentsclient.MCPServerRoute{
			{Upstreams: []agentsclient.MCPServerUpstream{
				{ID: "mcp-id-1", URI: "https://mcp-a.example.com"},
			}},
		}
		update := []agentsclient.MCPServerRoute{
			{Upstreams: []agentsclient.MCPServerUpstream{
				{URI: "https://mcp-a.example.com"},
			}},
		}
		r.mergeUpstreamIDs(current, update)
		if update[0].Upstreams[0].ID != "mcp-id-1" {
			t.Errorf("Upstream[0].ID = %q, want mcp-id-1", update[0].Upstreams[0].ID)
		}
		if update[0].Upstreams[0].URI != "" {
			t.Errorf("URI should be cleared after ID merge, got %q", update[0].Upstreams[0].URI)
		}
	})

	t.Run("positional fallback when URI not found", func(t *testing.T) {
		current := []agentsclient.MCPServerRoute{
			{Upstreams: []agentsclient.MCPServerUpstream{
				{ID: "pos-id", URI: "https://old.example.com"},
			}},
		}
		update := []agentsclient.MCPServerRoute{
			{Upstreams: []agentsclient.MCPServerUpstream{
				{URI: "https://new.example.com"},
			}},
		}
		r.mergeUpstreamIDs(current, update)
		if update[0].Upstreams[0].ID != "pos-id" {
			t.Errorf("positional fallback: ID = %q, want pos-id", update[0].Upstreams[0].ID)
		}
	})

	t.Run("empty current does nothing", func(t *testing.T) {
		update := []agentsclient.MCPServerRoute{
			{Upstreams: []agentsclient.MCPServerUpstream{{URI: "https://mcp.example.com"}}},
		}
		r.mergeUpstreamIDs(nil, update)
		if update[0].Upstreams[0].ID != "" {
			t.Errorf("Expected empty ID, got %q", update[0].Upstreams[0].ID)
		}
	})
}

// --- MCPServerResource.expandRouting ---

func TestMCPServerResource_expandRouting(t *testing.T) {
	r := &MCPServerResource{}
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

// --- MCPServerResource.expandCreateRequest ---

func TestMCPServerResource_expandCreateRequest(t *testing.T) {
	r := &MCPServerResource{}
	ctx := context.Background()

	t.Run("technology omniGateway is mapped to flexGateway", func(t *testing.T) {
		data := MCPServerResourceModel{
			Technology:       types.StringValue("omniGateway"),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(mcpRouteListElemType),
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

	t.Run("upstream_uri creates single-route routing", func(t *testing.T) {
		data := MCPServerResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringValue("https://mcp-backend.example.com"),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(mcpRouteListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandCreateRequest(ctx, data)
		if len(req.Routing) != 1 {
			t.Fatalf("Routing len = %d, want 1", len(req.Routing))
		}
		if req.Routing[0].Upstreams[0].URI != "https://mcp-backend.example.com" {
			t.Errorf("Upstream URI = %q", req.Routing[0].Upstreams[0].URI)
		}
		if req.Routing[0].Upstreams[0].Weight != 100 {
			t.Errorf("Weight = %d, want 100", req.Routing[0].Upstreams[0].Weight)
		}
	})

	t.Run("endpoint generates mcp type", func(t *testing.T) {
		epObj, _ := types.ObjectValue(endpointAttrTypes, map[string]attr.Value{
			"deployment_type":  types.StringValue("HY"),
			"type":             types.StringValue("http"),
			"base_path":        types.StringValue("/mcp/v1"),
			"uri":              types.StringNull(),
			"response_timeout": types.Int64Null(),
		})
		data := MCPServerResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         epObj,
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(mcpRouteListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandCreateRequest(ctx, data)
		if req.Endpoint == nil {
			t.Fatal("Endpoint should not be nil")
		}
		if req.Endpoint.Type != "mcp" {
			t.Errorf("Endpoint.Type = %q, want mcp", req.Endpoint.Type)
		}
		if req.Endpoint.ProxyURI == nil || *req.Endpoint.ProxyURI != "http://0.0.0.0:8081/mcp/v1" {
			t.Errorf("ProxyURI = %v, want http://0.0.0.0:8081/mcp/v1", req.Endpoint.ProxyURI)
		}
	})
}

// --- MCPServerResource.expandUpdateRequest ---

func TestMCPServerResource_expandUpdateRequest(t *testing.T) {
	r := &MCPServerResource{}
	ctx := context.Background()

	t.Run("technology is mapped correctly", func(t *testing.T) {
		data := MCPServerResourceModel{
			Technology:       types.StringValue("omniGateway"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(mcpRouteListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandUpdateRequest(ctx, data)
		if req.Technology == nil || *req.Technology != "flexGateway" {
			t.Errorf("Technology = %v, want flexGateway", req.Technology)
		}
	})
}

// mcpRouteListElemType mirrors the route list element type for null list construction.
var mcpRouteListElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"label":     types.StringType,
		"rules":     types.ObjectType{AttrTypes: agentRulesAttrTypes},
		"upstreams": types.ListType{ElemType: agentUpstreamObjType},
	},
}
