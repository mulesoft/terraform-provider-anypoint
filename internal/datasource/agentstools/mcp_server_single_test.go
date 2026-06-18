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
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestMCPServerSingleDataSource_Schema(t *testing.T) {
	ds := NewMCPServerSingleDataSource()
	ctx := context.Background()
	metaResp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, datasource.MetadataRequest{ProviderTypeName: "test"}, metaResp)

	if metaResp.TypeName != "test_mcp_server" {
		t.Errorf("TypeName = %q, want test_mcp_server", metaResp.TypeName)
	}
}

func TestMCPServerSingleDataSource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/123"

	mockMCPServer := &agentsclient.MCPServer{
		ID:             123,
		Status:         "APPLIED",
		AssetID:        "test-asset",
		AssetVersion:   "1.0.0",
		ProductVersion: "v1",
		Technology:     "flexGateway",
		ProviderID:     ptrString("provider-123"),
		InstanceLabel:  "Test MCP Server",
		ApprovalMethod: "manual",
		EndpointURI:    "https://mcp-server.example.com/proxy",
		Spec: &agentsclient.MCPServerSpec{
			AssetID: "test-asset",
			GroupID: "org-group",
			Version: "1.0.0",
		},
		Endpoint: &agentsclient.MCPServerEndpoint{
			DeploymentType:  "HY",
			Type:            "mcp",
			ProxyURI:        ptrString("http://0.0.0.0:8081/test-mcp"),
			ResponseTimeout: ptrInt(30000),
		},
		Deployment: &agentsclient.MCPServerDeployment{
			EnvironmentID:  "test-env-id",
			Type:           "HY",
			ExpectedStatus: "deployed",
			Overwrite:      false,
			TargetID:       "target-gw-1",
			TargetName:     "Test Gateway",
			GatewayVersion: "1.6.0",
		},
		Routing: []agentsclient.MCPServerRoute{
			{
				Label: "default-route",
				Rules: &agentsclient.MCPServerRules{
					Methods: "GET|POST",
					Path:    "/api/*",
					Host:    "example.com",
					Headers: map[string]string{"X-Custom": "value"},
				},
				Upstreams: []agentsclient.MCPServerUpstream{
					{
						ID:     "upstream-1",
						Weight: 100,
						URI:    "http://backend.example.com",
						Label:  "backend",
						TLSContext: &agentsclient.MCPServerUpstreamTLS{
							SecretGroupID: "sg-123",
							TLSContextID:  "tls-456",
						},
					},
				},
			},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockMCPServer)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewMCPServerSingleDataSource().(*MCPServerSingleDataSource)
	ds.client = &agentsclient.MCPServerClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "123"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_id":       tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["spec"], nil),
		"endpoint":          tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["endpoint"], nil),
		"deployment":        tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["deployment"], nil),
		"routing":           tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["routing"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got MCPServerSingleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "123" {
		t.Errorf("ID = %q, want 123", got.ID.ValueString())
	}
	if got.Status.ValueString() != "APPLIED" {
		t.Errorf("Status = %q, want APPLIED", got.Status.ValueString())
	}
	if got.InstanceLabel.ValueString() != "Test MCP Server" {
		t.Errorf("InstanceLabel = %q, want Test MCP Server", got.InstanceLabel.ValueString())
	}
	if got.Technology.ValueString() != "omniGateway" {
		t.Errorf("Technology = %q, want omniGateway", got.Technology.ValueString())
	}
	if got.AssetID.ValueString() != "test-asset" {
		t.Errorf("AssetID = %q, want test-asset", got.AssetID.ValueString())
	}
	if got.UpstreamID.ValueString() != "upstream-1" {
		t.Errorf("UpstreamID = %q, want upstream-1", got.UpstreamID.ValueString())
	}
}

func TestMCPServerSingleDataSource_Read_NotFound(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/999"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewMCPServerSingleDataSource().(*MCPServerSingleDataSource)
	ds.client = &agentsclient.MCPServerClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "999"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_id":       tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["spec"], nil),
		"endpoint":          tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["endpoint"], nil),
		"deployment":        tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["deployment"], nil),
		"routing":           tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["routing"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report errors on 404")
	}
}

func TestMCPServerSingleDataSource_Read_InvalidID(t *testing.T) {
	ds := NewMCPServerSingleDataSource().(*MCPServerSingleDataSource)
	ds.client = &agentsclient.MCPServerClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    "http://example.com",
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, "not-a-number"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"provider_id":       tftypes.NewValue(tftypes.String, nil),
		"instance_label":    tftypes.NewValue(tftypes.String, nil),
		"approval_method":   tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"upstream_id":       tftypes.NewValue(tftypes.String, nil),
		"spec":              tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["spec"], nil),
		"endpoint":          tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["endpoint"], nil),
		"deployment":        tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["deployment"], nil),
		"routing":           tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["routing"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report errors on invalid ID")
	}
}

func ptrString(s string) *string {
	return &s
}

func ptrInt(i int) *int {
	return &i
}
