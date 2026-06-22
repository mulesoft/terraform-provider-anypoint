package agentstools

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	agentstoolsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestAgentInstanceSingleDataSource_Metadata(t *testing.T) {
	ds := NewAgentInstanceSingleDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)

	if resp.TypeName != "test_agent_instance" {
		t.Errorf("TypeName = %q, want test_agent_instance", resp.TypeName)
	}
}

func TestAgentInstanceSingleDataSource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/12345"

	mockInstance := &agentstoolsclient.AgentInstance{
		ID:             12345,
		AssetID:        "my-agent-asset",
		AssetVersion:   "1.0.0",
		ProductVersion: "v1",
		GroupID:        "test-group",
		Technology:     "flexGateway",
		InstanceLabel:  "My Agent Instance",
		Status:         "active",
		EndpointURI:    "https://agent.example.com",
		Spec: &agentstoolsclient.AgentInstanceSpec{
			AssetID: "my-agent-asset",
			GroupID: "test-group",
			Version: "1.0.0",
		},
		Endpoint: &agentstoolsclient.AgentInstanceEndpoint{
			DeploymentType: "HY",
			Type:           "a2a",
			ProxyURI:       strPtr("http://0.0.0.0:8081/my-agent"),
		},
		Deployment: &agentstoolsclient.AgentInstanceDeployment{
			EnvironmentID:  "test-env-id",
			Type:           "HY",
			ExpectedStatus: "deployed",
			Overwrite:      false,
			TargetID:       "gw-123",
			TargetName:     "Test Gateway",
			GatewayVersion: "1.6.0",
		},
		Routing: []agentstoolsclient.AgentInstanceRoute{
			{
				Label: "default",
				Upstreams: []agentstoolsclient.AgentInstanceUpstream{
					{
						Weight: 100,
						URI:    "http://upstream.example.com",
						Label:  "main-upstream",
					},
				},
			},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockInstance)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAgentInstanceSingleDataSource().(*AgentInstanceSingleDataSource)
	ds.client = &agentstoolsclient.AgentInstanceClient{
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
		"id":                tftypes.NewValue(tftypes.String, "12345"),
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
	var got AgentInstanceSingleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "12345" {
		t.Errorf("ID = %q, want 12345", got.ID.ValueString())
	}
	if got.Technology.ValueString() != "omniGateway" {
		t.Errorf("Technology = %q, want omniGateway", got.Technology.ValueString())
	}
	if got.InstanceLabel.ValueString() != "My Agent Instance" {
		t.Errorf("InstanceLabel = %q, want My Agent Instance", got.InstanceLabel.ValueString())
	}
	if got.Status.ValueString() != "active" {
		t.Errorf("Status = %q, want active", got.Status.ValueString())
	}
	if got.ConsumerEndpoint.ValueString() != "https://agent.example.com" {
		t.Errorf("ConsumerEndpoint = %q, want https://agent.example.com", got.ConsumerEndpoint.ValueString())
	}
}

func TestAgentInstanceSingleDataSource_Read_NotFound(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/99999"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAgentInstanceSingleDataSource().(*AgentInstanceSingleDataSource)
	ds.client = &agentstoolsclient.AgentInstanceClient{
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
		"id":                tftypes.NewValue(tftypes.String, "99999"),
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
		"spec":              tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["spec"], nil),
		"endpoint":          tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["endpoint"], nil),
		"deployment":        tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["deployment"], nil),
		"routing":           tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["routing"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report errors when agent instance not found")
	}
}

func strPtr(s string) *string {
	return &s
}
