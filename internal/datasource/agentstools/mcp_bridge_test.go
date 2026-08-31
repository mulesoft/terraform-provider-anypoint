package agentstools

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	agentsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewMCPBridgeDataSource(t *testing.T) {
	ds := NewMCPBridgeDataSource()
	if ds == nil {
		t.Fatal("NewMCPBridgeDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("MCPBridgeDataSource should implement DataSourceWithConfigure")
	}
}

func TestMCPBridgeDataSource_Metadata(t *testing.T) {
	ds := NewMCPBridgeDataSource()
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), datasource.MetadataRequest{ProviderTypeName: "test"}, resp)
	if resp.TypeName != "test_mcp_bridge" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_mcp_bridge")
	}
}

func TestMCPBridgeDataSource_Schema(t *testing.T) {
	ds := NewMCPBridgeDataSource()
	resp := &datasource.SchemaResponse{}
	ds.Schema(context.Background(), datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	for _, name := range []string{"id", "environment_id"} {
		if a, ok := resp.Schema.Attributes[name]; !ok || !a.IsRequired() {
			t.Errorf("%s should be a required attribute", name)
		}
	}
	if a, ok := resp.Schema.Attributes["source_apis"]; !ok || !a.IsComputed() {
		t.Error("source_apis should be a computed attribute")
	}
}

func TestMCPBridgeDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewMCPBridgeDataSource().(*MCPBridgeDataSource)
	resp := &datasource.ConfigureResponse{}
	ds.Configure(context.Background(), datasource.ConfigureRequest{ProviderData: "invalid"}, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should error")
	}
}

func TestMCPBridgeDataSource_Read(t *testing.T) {
	proxy := "http://0.0.0.0:8083/petstore"
	bridge := agentsclient.MCPBridge{
		ID: 10, AssetID: "b-asset", AssetVersion: "1.0.1", GroupID: "test-org-id",
		Technology: "flexGateway", Status: "active", EndpointURI: "https://mcp.example.com/petstore",
		Endpoint:   &agentsclient.MCPBridgeEndpoint{Type: "mcp", ProxyURI: &proxy},
		Deployment: &agentsclient.MCPBridgeDeployment{EnvironmentID: "test-env-id", Type: "HY", ExpectedStatus: "deployed", TargetID: "gw-1", TargetName: "gw-one", GatewayVersion: "1.2.0"},
		Routing:    []agentsclient.MCPBridgeRoute{{Label: "petstore", Upstreams: []agentsclient.MCPBridgeRouteUpstream{{ID: "u1"}}}},
	}
	upstreams := agentsclient.MCPBridgeUpstreamsResponse{
		Upstreams: []agentsclient.MCPBridgeUpstreamDetail{
			{ID: "u1", URI: "https://backend.example.com", Connection: &agentsclient.MCPBridgeConnection{AssetID: "petstore-api", Version: "1.0.0", Label: "petstore"}},
		},
	}
	policies := []apimanagement.APIPolicy{
		{ID: 1, AssetID: "mcp-transcoding-router", ConfigurationData: map[string]interface{}{
			"routes": []interface{}{
				map[string]interface{}{"upstreamName": "petstore", "tools": []interface{}{"get_pets"}},
			},
		}},
		{ID: 2, AssetID: "mcp-transcoding", ConfigurationData: map[string]interface{}{
			"tools": []interface{}{
				map[string]interface{}{"name": "get_pets", "method": "GET", "path": "/pets",
					"queryParams": []interface{}{map[string]interface{}{"key": "limit", "value": "x"}}},
			},
		}},
	}

	prefix := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/10"
	server := testutil.MockHTTPServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		prefix:                func(w http.ResponseWriter, r *http.Request) { testutil.JSONResponse(w, http.StatusOK, bridge) },
		prefix + "/upstreams": func(w http.ResponseWriter, r *http.Request) { testutil.JSONResponse(w, http.StatusOK, upstreams) },
		prefix + "/policies":  func(w http.ResponseWriter, r *http.Request) { testutil.JSONResponse(w, http.StatusOK, policies) },
	})

	ac := &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id"}
	ds := NewMCPBridgeDataSource().(*MCPBridgeDataSource)
	ds.client = &agentsclient.MCPBridgeClient{AnypointClient: ac, Policies: &apimanagement.APIPolicyClient{AnypointClient: ac}}

	got := readBridgeDS(t, ds, "10")

	if got.GatewayID.ValueString() != "gw-1" {
		t.Errorf("gateway_id = %q, want gw-1", got.GatewayID.ValueString())
	}
	if got.Port.ValueInt64() != 8083 || got.BasePath.ValueString() != "petstore" {
		t.Errorf("port/base_path = %d/%q, want 8083/petstore", got.Port.ValueInt64(), got.BasePath.ValueString())
	}
	if got.AssetVersion.ValueString() != "1.0.1" {
		t.Errorf("asset_version = %q, want 1.0.1", got.AssetVersion.ValueString())
	}

	var sources []dsBridgeSourceRead
	if diags := got.SourceAPIs.ElementsAs(context.Background(), &sources, false); diags.HasError() {
		t.Fatalf("source_apis ElementsAs: %v", diags.Errors())
	}
	if len(sources) != 1 {
		t.Fatalf("expected 1 source, got %d", len(sources))
	}
	s := sources[0]
	if s.Label != "petstore" || s.AssetID != "petstore-api" || s.Version != "1.0.0" || s.GroupID != "test-org-id" {
		t.Errorf("source metadata = %+v", s)
	}
	if len(s.Tools) != 1 || s.Tools[0].Name != "get_pets" || s.Tools[0].Method != "GET" || s.Tools[0].Path != "/pets" {
		t.Fatalf("tools = %+v", s.Tools)
	}
	if len(s.Tools[0].QueryParams) != 1 || s.Tools[0].QueryParams[0] != "limit" {
		t.Errorf("query_params = %v, want [limit]", s.Tools[0].QueryParams)
	}
}

func TestMCPBridgeDataSource_Read_NotFound(t *testing.T) {
	prefix := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/99"
	server := testutil.MockHTTPServer(t, map[string]func(w http.ResponseWriter, r *http.Request){
		prefix: func(w http.ResponseWriter, r *http.Request) { w.WriteHeader(http.StatusNotFound) },
	})
	ac := &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id"}
	ds := NewMCPBridgeDataSource().(*MCPBridgeDataSource)
	ds.client = &agentsclient.MCPBridgeClient{AnypointClient: ac, Policies: &apimanagement.APIPolicyClient{AnypointClient: ac}}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	configRaw := bridgeSingleConfigRaw(ctx, ds, "99")
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should error when the bridge is not found")
	}
	_ = stateType
}

// dsBridgeSourceRead / dsBridgeToolRead mirror the single DS source/tool object shape for
// decoding source_apis out of state in tests.
type dsBridgeToolRead struct {
	Name         string   `tfsdk:"name"`
	Description  *string  `tfsdk:"description"`
	Method       string   `tfsdk:"method"`
	Path         string   `tfsdk:"path"`
	QueryParams  []string `tfsdk:"query_params"`
	HeaderParams []string `tfsdk:"header_params"`
	HasBody      bool     `tfsdk:"has_body"`
}

type dsBridgeSourceRead struct {
	Label       string             `tfsdk:"label"`
	UpstreamURI string             `tfsdk:"upstream_uri"`
	AssetID     string             `tfsdk:"asset_id"`
	GroupID     string             `tfsdk:"group_id"`
	Version     string             `tfsdk:"version"`
	Tools       []dsBridgeToolRead `tfsdk:"tools"`
}

func bridgeSingleConfigRaw(ctx context.Context, ds *MCPBridgeDataSource, id string) tftypes.Value {
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	obj := stateType.(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for name, ty := range obj.AttributeTypes {
		switch name {
		case "id":
			vals[name] = tftypes.NewValue(tftypes.String, id)
		case "organization_id":
			vals[name] = tftypes.NewValue(tftypes.String, "test-org-id")
		case "environment_id":
			vals[name] = tftypes.NewValue(tftypes.String, "test-env-id")
		default:
			vals[name] = tftypes.NewValue(ty, nil)
		}
	}
	return tftypes.NewValue(stateType, vals)
}

func readBridgeDS(t *testing.T, ds *MCPBridgeDataSource, id string) MCPBridgeDataSourceModel {
	t.Helper()
	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	configRaw := bridgeSingleConfigRaw(ctx, ds, id)
	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got MCPBridgeDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	return got
}
