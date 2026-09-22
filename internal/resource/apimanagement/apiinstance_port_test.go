package apimanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
)

// endpointObjWithPort builds a types.Object matching endpointAttrTypes for the
// resource-side expand tests. Pass a null base_path/port with types.StringNull()/
// types.Int64Null().
func endpointObjWithPort(deploymentType, epType string, basePath types.String, port types.Int64, responseTimeout types.Int64) types.Object {
	obj, _ := types.ObjectValue(endpointAttrTypes, map[string]attr.Value{
		"deployment_type":  types.StringValue(deploymentType),
		"type":             types.StringValue(epType),
		"base_path":        basePath,
		"port":             port,
		"response_timeout": responseTimeout,
	})
	return obj
}

func portTestModel(ep types.Object) APIInstanceResourceModel {
	return APIInstanceResourceModel{
		Technology:       types.StringValue("mule4"),
		UpstreamURI:      types.StringNull(),
		Endpoint:         ep,
		Deployment:       types.ObjectNull(deploymentAttrTypes),
		Routing:          types.ListNull(routeListElemType),
		ProviderID:       types.StringNull(),
		InstanceLabel:    types.StringNull(),
		ApprovalMethod:   types.StringNull(),
		ConsumerEndpoint: types.StringNull(),
	}
}

// --- buildProxyURI (pure helper) -------------------------------------------

func TestBuildProxyURI(t *testing.T) {
	cases := []struct {
		name    string
		ep      *EndpointModel
		wantURI string
	}{
		{
			name:    "custom port with base_path",
			ep:      &EndpointModel{BasePath: types.StringValue("graphql-api"), Port: types.Int64Value(8082)},
			wantURI: "http://0.0.0.0:8082/graphql-api",
		},
		{
			name:    "null port defaults to 8081",
			ep:      &EndpointModel{BasePath: types.StringValue("api/v1"), Port: types.Int64Null()},
			wantURI: "http://0.0.0.0:8081/api/v1",
		},
		{
			name:    "unknown port defaults to 8081",
			ep:      &EndpointModel{BasePath: types.StringValue("api/v1"), Port: types.Int64Unknown()},
			wantURI: "http://0.0.0.0:8081/api/v1",
		},
		{
			name:    "leading slash on base_path is trimmed",
			ep:      &EndpointModel{BasePath: types.StringValue("/leading"), Port: types.Int64Value(9000)},
			wantURI: "http://0.0.0.0:9000/leading",
		},
		{
			name:    "null base_path yields root path on custom port",
			ep:      &EndpointModel{BasePath: types.StringNull(), Port: types.Int64Value(8083)},
			wantURI: "http://0.0.0.0:8083/",
		},
		{
			name:    "both null → legacy default 8081 root",
			ep:      &EndpointModel{BasePath: types.StringNull(), Port: types.Int64Null()},
			wantURI: "http://0.0.0.0:8081/",
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := buildProxyURI(tc.ep); got != tc.wantURI {
				t.Errorf("buildProxyURI() = %q, want %q", got, tc.wantURI)
			}
		})
	}
}

// --- parseProxyURI (pure helper) -------------------------------------------

func TestParseProxyURI(t *testing.T) {
	cases := []struct {
		name         string
		proxyURI     string
		wantBasePath string
		wantPort     int64
	}{
		{
			name:         "non-8081 port round-trips base_path and port",
			proxyURI:     "http://0.0.0.0:8082/graphql-api",
			wantBasePath: "graphql-api",
			wantPort:     8082,
		},
		{
			name:         "legacy 8081 still yields port 8081",
			proxyURI:     "http://0.0.0.0:8081/api/v1",
			wantBasePath: "api/v1",
			wantPort:     8081,
		},
		{
			name:         "root path yields empty base_path",
			proxyURI:     "http://0.0.0.0:8082/",
			wantBasePath: "",
			wantPort:     8082,
		},
		{
			name:         "no port falls back to 8081 default",
			proxyURI:     "http://example.com/some-path",
			wantBasePath: "some-path",
			wantPort:     8081,
		},
		{
			name:         "multi-segment base path preserved",
			proxyURI:     "http://0.0.0.0:8085/a/b/c",
			wantBasePath: "a/b/c",
			wantPort:     8085,
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bp, port := parseProxyURI(tc.proxyURI)
			if bp != tc.wantBasePath {
				t.Errorf("parseProxyURI(%q) base_path = %q, want %q", tc.proxyURI, bp, tc.wantBasePath)
			}
			if port != tc.wantPort {
				t.Errorf("parseProxyURI(%q) port = %d, want %d", tc.proxyURI, port, tc.wantPort)
			}
		})
	}
}

// --- expandCreateRequest wires the port into the proxy URI ------------------

func TestAPIInstanceResource_expandCreateRequest_port(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	t.Run("custom port lands in proxy URI", func(t *testing.T) {
		ep := endpointObjWithPort("CH", "graphql", types.StringValue("graphql-api"), types.Int64Value(8082), types.Int64Null())
		req := r.expandCreateRequest(ctx, portTestModel(ep))
		if req.Endpoint == nil || req.Endpoint.ProxyURI == nil {
			t.Fatal("Endpoint/ProxyURI should not be nil")
		}
		if *req.Endpoint.ProxyURI != "http://0.0.0.0:8082/graphql-api" {
			t.Errorf("ProxyURI = %q, want http://0.0.0.0:8082/graphql-api", *req.Endpoint.ProxyURI)
		}
	})

	t.Run("null port defaults to 8081 (back-compat)", func(t *testing.T) {
		ep := endpointObjWithPort("CH", "http", types.StringValue("api/v1"), types.Int64Null(), types.Int64Null())
		req := r.expandCreateRequest(ctx, portTestModel(ep))
		if req.Endpoint == nil || req.Endpoint.ProxyURI == nil {
			t.Fatal("Endpoint/ProxyURI should not be nil")
		}
		if *req.Endpoint.ProxyURI != "http://0.0.0.0:8081/api/v1" {
			t.Errorf("ProxyURI = %q, want http://0.0.0.0:8081/api/v1", *req.Endpoint.ProxyURI)
		}
	})
}

// --- expandUpdateRequest wires the port into the proxy URI ------------------

func TestAPIInstanceResource_expandUpdateRequest_port(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	t.Run("custom port lands in proxy URI on update", func(t *testing.T) {
		ep := endpointObjWithPort("HY", "http", types.StringValue("moved"), types.Int64Value(9090), types.Int64Null())
		req := r.expandUpdateRequest(ctx, portTestModel(ep))
		if req.Endpoint == nil || req.Endpoint.ProxyURI == nil {
			t.Fatal("Endpoint/ProxyURI should not be nil")
		}
		if *req.Endpoint.ProxyURI != "http://0.0.0.0:9090/moved" {
			t.Errorf("ProxyURI = %q, want http://0.0.0.0:9090/moved", *req.Endpoint.ProxyURI)
		}
	})

	t.Run("null port defaults to 8081 on update", func(t *testing.T) {
		ep := endpointObjWithPort("HY", "http", types.StringValue("stay"), types.Int64Null(), types.Int64Null())
		req := r.expandUpdateRequest(ctx, portTestModel(ep))
		if req.Endpoint == nil || req.Endpoint.ProxyURI == nil {
			t.Fatal("Endpoint/ProxyURI should not be nil")
		}
		if *req.Endpoint.ProxyURI != "http://0.0.0.0:8081/stay" {
			t.Errorf("ProxyURI = %q, want http://0.0.0.0:8081/stay", *req.Endpoint.ProxyURI)
		}
	})
}

// --- flattenInstance recovers both base_path AND port -----------------------

func TestAPIInstanceResource_flattenInstance_port(t *testing.T) {
	r := &APIInstanceResource{}
	ctx := context.Background()

	read := func(t *testing.T, proxyURI string) *EndpointModel {
		t.Helper()
		inst := &apimgmtclient.APIInstance{
			ID:         1,
			Technology: "flexGateway",
			Endpoint: &apimgmtclient.APIInstanceEndpoint{
				DeploymentType: "HY",
				Type:           "graphql",
				ProxyURI:       &proxyURI,
			},
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		ep := endpointFromObject(data.Endpoint)
		if ep == nil {
			t.Fatal("endpoint should not be nil after flatten")
		}
		return ep
	}

	t.Run("non-8081 proxyUri recovers base_path and port", func(t *testing.T) {
		ep := read(t, "http://0.0.0.0:8082/graphql-api")
		if ep.BasePath.ValueString() != "graphql-api" {
			t.Errorf("BasePath = %q, want graphql-api", ep.BasePath.ValueString())
		}
		if ep.Port.ValueInt64() != 8082 {
			t.Errorf("Port = %d, want 8082", ep.Port.ValueInt64())
		}
	})

	t.Run("legacy 8081 proxyUri yields port 8081", func(t *testing.T) {
		ep := read(t, "http://0.0.0.0:8081/mypath")
		if ep.BasePath.ValueString() != "mypath" {
			t.Errorf("BasePath = %q, want mypath", ep.BasePath.ValueString())
		}
		if ep.Port.ValueInt64() != 8081 {
			t.Errorf("Port = %d, want 8081", ep.Port.ValueInt64())
		}
	})

	t.Run("empty proxyUri yields null base_path and default port", func(t *testing.T) {
		emptyURI := ""
		inst := &apimgmtclient.APIInstance{
			ID:         2,
			Technology: "flexGateway",
			Endpoint: &apimgmtclient.APIInstanceEndpoint{
				DeploymentType: "HY",
				Type:           "http",
				ProxyURI:       &emptyURI,
			},
		}
		data := &APIInstanceResourceModel{OrganizationID: types.StringValue("org-1")}
		r.flattenInstance(ctx, inst, data, "org-1", "env-1")
		ep := endpointFromObject(data.Endpoint)
		if ep == nil {
			t.Fatal("endpoint should not be nil after flatten")
		}
		if !ep.BasePath.IsNull() {
			t.Errorf("BasePath = %q, want null", ep.BasePath.ValueString())
		}
		if ep.Port.ValueInt64() != 8081 {
			t.Errorf("Port = %d, want 8081 default", ep.Port.ValueInt64())
		}
	})

	t.Run("create->read round-trips a custom port", func(t *testing.T) {
		// expandCreateRequest builds the proxy URI; flatten must recover it.
		epObj := endpointObjWithPort("HY", "graphql", types.StringValue("rt"), types.Int64Value(8087), types.Int64Null())
		created := r.expandCreateRequest(ctx, portTestModel(epObj))
		if created.Endpoint == nil || created.Endpoint.ProxyURI == nil {
			t.Fatal("created endpoint/proxyURI nil")
		}
		ep := read(t, *created.Endpoint.ProxyURI)
		if ep.BasePath.ValueString() != "rt" {
			t.Errorf("round-trip BasePath = %q, want rt", ep.BasePath.ValueString())
		}
		if ep.Port.ValueInt64() != 8087 {
			t.Errorf("round-trip Port = %d, want 8087", ep.Port.ValueInt64())
		}
	})
}
