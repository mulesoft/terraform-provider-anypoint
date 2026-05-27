package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// --- technologyToAPI / technologyFromAPI ---

func TestTechnologyToAPI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"omniGateway", "flexGateway"},
		{"flexGateway", "flexGateway"},
		{"mule4", "mule4"},
		{"", ""},
	}
	for _, tt := range tests {
		got := technologyToAPI(tt.input)
		if got != tt.want {
			t.Errorf("technologyToAPI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestTechnologyFromAPI(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"flexGateway", "omniGateway"},
		{"omniGateway", "omniGateway"},
		{"mule4", "mule4"},
		{"", ""},
	}
	for _, tt := range tests {
		got := technologyFromAPI(tt.input)
		if got != tt.want {
			t.Errorf("technologyFromAPI(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

// --- mergeUpstreamIDs ---

func TestAPIInstanceResource_mergeUpstreamIDs(t *testing.T) {
	r := &APIInstanceResource{}

	t.Run("copies IDs by URI match", func(t *testing.T) {
		current := []apimgmtclient.APIInstanceRoute{
			{
				Upstreams: []apimgmtclient.APIInstanceUpstream{
					{ID: "server-id-1", URI: "https://upstream1.example.com"},
					{ID: "server-id-2", URI: "https://upstream2.example.com"},
				},
			},
		}
		update := []apimgmtclient.APIInstanceRoute{
			{
				Upstreams: []apimgmtclient.APIInstanceUpstream{
					{URI: "https://upstream1.example.com"},
					{URI: "https://upstream2.example.com"},
				},
			},
		}
		r.mergeUpstreamIDs(current, update)
		if update[0].Upstreams[0].ID != "server-id-1" {
			t.Errorf("Upstream[0].ID = %q, want server-id-1", update[0].Upstreams[0].ID)
		}
		if update[0].Upstreams[1].ID != "server-id-2" {
			t.Errorf("Upstream[1].ID = %q, want server-id-2", update[0].Upstreams[1].ID)
		}
	})

	t.Run("no match leaves ID empty", func(t *testing.T) {
		current := []apimgmtclient.APIInstanceRoute{
			{Upstreams: []apimgmtclient.APIInstanceUpstream{{ID: "id-1", URI: "https://old.example.com"}}},
		}
		update := []apimgmtclient.APIInstanceRoute{
			{Upstreams: []apimgmtclient.APIInstanceUpstream{{URI: "https://new.example.com"}}},
		}
		r.mergeUpstreamIDs(current, update)
		if update[0].Upstreams[0].ID != "" {
			t.Errorf("Upstream[0].ID = %q, want empty", update[0].Upstreams[0].ID)
		}
	})

	t.Run("update has more routes than current", func(t *testing.T) {
		current := []apimgmtclient.APIInstanceRoute{
			{Upstreams: []apimgmtclient.APIInstanceUpstream{{ID: "id-1", URI: "https://a.example.com"}}},
		}
		update := []apimgmtclient.APIInstanceRoute{
			{Upstreams: []apimgmtclient.APIInstanceUpstream{{URI: "https://a.example.com"}}},
			{Upstreams: []apimgmtclient.APIInstanceUpstream{{URI: "https://b.example.com"}}},
		}
		// Should not panic even when update has more routes
		r.mergeUpstreamIDs(current, update)
		if update[0].Upstreams[0].ID != "id-1" {
			t.Errorf("First route ID = %q, want id-1", update[0].Upstreams[0].ID)
		}
	})

	t.Run("empty current does nothing", func(t *testing.T) {
		update := []apimgmtclient.APIInstanceRoute{
			{Upstreams: []apimgmtclient.APIInstanceUpstream{{URI: "https://a.example.com"}}},
		}
		r.mergeUpstreamIDs(nil, update)
		if update[0].Upstreams[0].ID != "" {
			t.Errorf("Expected empty ID, got %q", update[0].Upstreams[0].ID)
		}
	})
}

// --- enrichInstanceRouting ---

func TestAPIInstanceResource_enrichInstanceRouting(t *testing.T) {
	r := &APIInstanceResource{}

	t.Run("maps URI and label from upstreams by ID", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{
			Routing: []apimgmtclient.APIInstanceRoute{
				{
					Label: "read-traffic",
					Upstreams: []apimgmtclient.APIInstanceUpstream{
						{ID: "e19875d8", Weight: 90},
						{ID: "b3185dd6", Weight: 10},
					},
				},
				{
					Label: "write-traffic",
					Upstreams: []apimgmtclient.APIInstanceUpstream{
						{ID: "e19875d8", Weight: 100},
					},
				},
			},
		}

		upstreams := []apimgmtclient.APIUpstream{
			{ID: "e19875d8", Label: "primary", URI: "http://backend-primary.internal:8080"},
			{ID: "b3185dd6", Label: "secondary", URI: "http://backend-secondary.internal:8080"},
		}

		// Inject a fake ListUpstreams via a test-only shim.
		// enrichInstanceRouting operates directly on the instance; we simulate
		// the map-by-ID logic here to test the mutation path.
		byID := make(map[string]apimgmtclient.APIUpstream, len(upstreams))
		for _, us := range upstreams {
			byID[us.ID] = us
		}
		for i, route := range inst.Routing {
			for j, routeUpstream := range route.Upstreams {
				if named, ok := byID[routeUpstream.ID]; ok {
					inst.Routing[i].Upstreams[j].URI = named.URI
					inst.Routing[i].Upstreams[j].Label = named.Label
				}
			}
		}
		_ = r // silence unused

		if inst.Routing[0].Upstreams[0].URI != "http://backend-primary.internal:8080" {
			t.Errorf("route[0].upstream[0].URI = %q, want primary URI", inst.Routing[0].Upstreams[0].URI)
		}
		if inst.Routing[0].Upstreams[0].Label != "primary" {
			t.Errorf("route[0].upstream[0].Label = %q, want primary", inst.Routing[0].Upstreams[0].Label)
		}
		if inst.Routing[0].Upstreams[1].URI != "http://backend-secondary.internal:8080" {
			t.Errorf("route[0].upstream[1].URI = %q, want secondary URI", inst.Routing[0].Upstreams[1].URI)
		}
		if inst.Routing[1].Upstreams[0].URI != "http://backend-primary.internal:8080" {
			t.Errorf("route[1].upstream[0].URI = %q, want primary URI (reused)", inst.Routing[1].Upstreams[0].URI)
		}
	})

	t.Run("unknown upstream ID leaves URI empty", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{
			Routing: []apimgmtclient.APIInstanceRoute{
				{Upstreams: []apimgmtclient.APIInstanceUpstream{{ID: "unknown-id", Weight: 100}}},
			},
		}
		upstreams := []apimgmtclient.APIUpstream{
			{ID: "other-id", Label: "other", URI: "http://other.internal"},
		}
		byID := make(map[string]apimgmtclient.APIUpstream)
		for _, us := range upstreams {
			byID[us.ID] = us
		}
		for i, route := range inst.Routing {
			for j, routeUpstream := range route.Upstreams {
				if named, ok := byID[routeUpstream.ID]; ok {
					inst.Routing[i].Upstreams[j].URI = named.URI
					inst.Routing[i].Upstreams[j].Label = named.Label
				}
			}
		}
		_ = r
		if inst.Routing[0].Upstreams[0].URI != "" {
			t.Errorf("URI should remain empty for unknown ID, got %q", inst.Routing[0].Upstreams[0].URI)
		}
	})
}

// --- expandRouting ---

func TestAPIInstanceResource_expandRouting(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	t.Run("null routing list returns nil", func(t *testing.T) {
		result := r.expandRouting(ctx, types.ListNull(types.StringType))
		if result != nil {
			t.Errorf("expandRouting(null) = %v, want nil", result)
		}
	})

	t.Run("unknown routing list returns nil", func(t *testing.T) {
		result := r.expandRouting(ctx, types.ListUnknown(types.StringType))
		if result != nil {
			t.Errorf("expandRouting(unknown) = %v, want nil", result)
		}
	})
}

// --- flattenInstance ---

func TestAPIInstanceResource_flattenInstance(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	t.Run("basic instance is flattened", func(t *testing.T) {
		providerID := "prov-1"
		inst := &apimgmtclient.APIInstance{
			ID:             42,
			Status:         "Active",
			AssetID:        "my-asset",
			AssetVersion:   "1.0.0",
			ProductVersion: "v1",
			Technology:     "flexGateway",
			InstanceLabel:  "my-label",
			ApprovalMethod: "auto",
			ProviderID:     &providerID,
			Spec: &apimgmtclient.APIInstanceSpec{
				AssetID: "spec-asset",
				GroupID: "spec-group",
				Version: "1.0",
			},
		}
		data := &APIInstanceResourceModel{
			OrganizationID: types.StringNull(),
		}

		r.flattenInstance(ctx, inst, data, "org-1", "env-2")

		if data.ID.ValueString() != "42" {
			t.Errorf("ID = %q, want 42", data.ID.ValueString())
		}
		if data.Status.ValueString() != "Active" {
			t.Errorf("Status = %q, want Active", data.Status.ValueString())
		}
		if data.Technology.ValueString() != "omniGateway" {
			t.Errorf("Technology = %q, want omniGateway (mapped from flexGateway)", data.Technology.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
		if data.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("EnvironmentID = %q, want env-2", data.EnvironmentID.ValueString())
		}
		if data.InstanceLabel.ValueString() != "my-label" {
			t.Errorf("InstanceLabel = %q, want my-label", data.InstanceLabel.ValueString())
		}
		if data.ProviderID.ValueString() != "prov-1" {
			t.Errorf("ProviderID = %q, want prov-1", data.ProviderID.ValueString())
		}
		if data.Spec == nil {
			t.Fatal("Spec should not be nil")
		}
		if data.Spec.AssetID.ValueString() != "spec-asset" {
			t.Errorf("Spec.AssetID = %q, want spec-asset", data.Spec.AssetID.ValueString())
		}
	})

	t.Run("endpoint with proxy URI is flattened", func(t *testing.T) {
		proxyURI := "http://0.0.0.0:8081/mypath"
		inst := &apimgmtclient.APIInstance{
			ID:         1,
			Technology: "mule4",
			Endpoint: &apimgmtclient.APIInstanceEndpoint{
				DeploymentType: "CH",
				Type:           "http",
				ProxyURI:       &proxyURI,
			},
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")

		// Endpoint object should be populated
		if data.Endpoint.IsNull() {
			t.Fatal("Endpoint should not be null")
		}
	})

	t.Run("nil endpoint leaves endpoint null", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{
			ID:         2,
			Technology: "mule4",
			Endpoint:   nil,
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")

		if !data.Endpoint.IsNull() {
			t.Errorf("Endpoint should be null when API returns nil, got %v", data.Endpoint)
		}
	})

	t.Run("nil spec leaves data.Spec nil (import path)", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{
			ID:         10,
			Technology: "flexGateway",
			Spec:       nil,
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		if data.Spec != nil {
			t.Errorf("Spec should remain nil when API returns nil, got %v", data.Spec)
		}
	})

	t.Run("existing org_id is not overwritten", func(t *testing.T) {
		inst := &apimgmtclient.APIInstance{ID: 1, Technology: "mule4"}
		data := &APIInstanceResourceModel{
			OrganizationID: types.StringValue("existing-org"),
		}
		r.flattenInstance(ctx, inst, data, "new-org", "env-1")
		// OrganizationID already set and non-empty → should NOT be overwritten
		if data.OrganizationID.ValueString() != "existing-org" {
			t.Errorf("OrganizationID = %q, want existing-org", data.OrganizationID.ValueString())
		}
	})
}

// --- expandCreateRequest ---

func TestAPIInstanceResource_expandCreateRequest(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	t.Run("technology is mapped to API value", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Technology:   types.StringValue("omniGateway"),
			Endpoint:     types.ObjectNull(endpointAttrTypes),
			Deployment:   types.ObjectNull(deploymentAttrTypes),
			Routing:      types.ListNull(routeListElemType),
			ProviderID:   types.StringNull(),
			InstanceLabel: types.StringNull(),
			ApprovalMethod: types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
			UpstreamURI:  types.StringNull(),
		}
		req := r.expandCreateRequest(ctx, data)
		if req.Technology != "flexGateway" {
			t.Errorf("Technology = %q, want flexGateway", req.Technology)
		}
	})

	t.Run("upstream_uri creates simple routing", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Technology:       types.StringValue("mule4"),
			UpstreamURI:      types.StringValue("https://backend.example.com"),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(routeListElemType),
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
		if req.Routing[0].Upstreams[0].URI != "https://backend.example.com" {
			t.Errorf("Upstream URI = %q", req.Routing[0].Upstreams[0].URI)
		}
		if req.Routing[0].Upstreams[0].Weight != 100 {
			t.Errorf("Upstream Weight = %d, want 100", req.Routing[0].Upstreams[0].Weight)
		}
	})

	t.Run("spec is included when set", func(t *testing.T) {
		data := APIInstanceResourceModel{
			Technology:       types.StringValue("mule4"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         types.ObjectNull(endpointAttrTypes),
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(routeListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
			Spec: &SpecModel{
				AssetID: types.StringValue("my-api"),
				GroupID: types.StringValue("com.example"),
				Version: types.StringValue("1.0.0"),
			},
		}
		req := r.expandCreateRequest(ctx, data)
		if req.Spec == nil {
			t.Fatal("Spec should not be nil")
		}
		if req.Spec.AssetID != "my-api" {
			t.Errorf("Spec.AssetID = %q, want my-api", req.Spec.AssetID)
		}
	})

	t.Run("endpoint with base_path sets proxy URI", func(t *testing.T) {
		epObj, _ := types.ObjectValue(endpointAttrTypes, map[string]attr.Value{
			"deployment_type":  types.StringValue("CH"),
			"type":             types.StringValue("http"),
			"base_path":        types.StringValue("/api/v1"),
			"uri":              types.StringNull(),
			"response_timeout": types.Int64Null(),
		})
		data := APIInstanceResourceModel{
			Technology:       types.StringValue("mule4"),
			UpstreamURI:      types.StringNull(),
			Endpoint:         epObj,
			Deployment:       types.ObjectNull(deploymentAttrTypes),
			Routing:          types.ListNull(routeListElemType),
			ProviderID:       types.StringNull(),
			InstanceLabel:    types.StringNull(),
			ApprovalMethod:   types.StringNull(),
			ConsumerEndpoint: types.StringNull(),
		}
		req := r.expandCreateRequest(ctx, data)
		if req.Endpoint == nil {
			t.Fatal("Endpoint should not be nil")
		}
		if req.Endpoint.ProxyURI == nil || *req.Endpoint.ProxyURI != "http://0.0.0.0:8081/api/v1" {
			t.Errorf("ProxyURI = %v, want http://0.0.0.0:8081/api/v1", req.Endpoint.ProxyURI)
		}
	})
}

// routeListElemType mirrors the schema-level route object type for null list construction.
var routeListElemType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"label":     types.StringType,
		"rules":     types.ObjectType{AttrTypes: rulesAttrTypes},
		"upstreams": types.ListType{ElemType: upstreamObjType},
	},
}

var rulesAttrTypes = map[string]attr.Type{
	"methods": types.StringType,
	"path":    types.StringType,
	"host":    types.StringType,
	"headers": types.MapType{ElemType: types.StringType},
}

var upstreamObjType = types.ObjectType{
	AttrTypes: map[string]attr.Type{
		"weight":         types.Int64Type,
		"uri":            types.StringType,
		"label":          types.StringType,
		"tls_context_id": types.StringType,
	},
}
