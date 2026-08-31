package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestAPIInstanceSingleDataSource_Metadata(t *testing.T) {
	ds := NewAPIInstanceSingleDataSource().(*APIInstanceSingleDataSource)
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(context.Background(), req, resp)

	if resp.TypeName != "test_api_instance" {
		t.Errorf("TypeName = %q, want test_api_instance", resp.TypeName)
	}
}

func TestAPIInstanceSingleDataSource_Read(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/12345"

	mockInstance := &apimgmtclient.APIInstance{
		ID:             12345,
		AssetID:        "my-api",
		AssetVersion:   "1.0.0",
		ProductVersion: "v1",
		Technology:     "flexGateway",
		InstanceLabel:  "Production API",
		Status:         "active",
		EndpointURI:    "https://api.example.com/v1",
		Spec: &apimgmtclient.APIInstanceSpec{
			AssetID: "my-api",
			GroupID: "test-org-id",
			Version: "1.0.0",
		},
		Endpoint: &apimgmtclient.APIInstanceEndpoint{
			DeploymentType: "HY",
			Type:           "http",
			ProxyURI:       strPtr("http://0.0.0.0:8081/my-api"),
		},
		Deployment: &apimgmtclient.APIInstanceDeployment{
			EnvironmentID:  "test-env-id",
			Type:           "HY",
			ExpectedStatus: "deployed",
			Overwrite:      false,
			TargetID:       "gw-123",
			TargetName:     "Production Gateway",
			GatewayVersion: "1.6.0",
		},
		Routing: []apimgmtclient.APIInstanceRoute{
			{
				Label: "default",
				Upstreams: []apimgmtclient.APIInstanceUpstream{
					{Weight: 100, URI: "http://backend.example.com"},
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

	ds := NewAPIInstanceSingleDataSource().(*APIInstanceSingleDataSource)
	ds.client = &apimgmtclient.APIInstanceClient{
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
	var got APIInstanceSingleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.InstanceLabel.ValueString() != "Production API" {
		t.Errorf("InstanceLabel = %q, want Production API", got.InstanceLabel.ValueString())
	}
	if got.Status.ValueString() != "active" {
		t.Errorf("Status = %q, want active", got.Status.ValueString())
	}
	if got.Technology.ValueString() != "omniGateway" {
		t.Errorf("Technology = %q, want omniGateway (converted from flexGateway)", got.Technology.ValueString())
	}
	if got.ConsumerEndpoint.ValueString() != "https://api.example.com/v1" {
		t.Errorf("ConsumerEndpoint = %q, want https://api.example.com/v1", got.ConsumerEndpoint.ValueString())
	}
}

func TestAPIInstanceSingleDataSource_Read_NotFound(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/99999"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "API instance not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIInstanceSingleDataSource().(*APIInstanceSingleDataSource)
	ds.client = &apimgmtclient.APIInstanceClient{
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
		t.Error("Read() should report errors on 404")
	}
}

func strPtr(s string) *string {
	return &s
}

// TestParseProxyURI_DataSource guards the singular data source's proxyUri parse.
// It replaced the old extractBasePath, which hardcoded the "http://0.0.0.0:8081/"
// prefix: for a non-8081 listener it dropped the port entirely and, when the port
// had a different digit count, sliced the base path at the wrong offset. The new
// parser recovers base_path AND port generically.
func TestParseProxyURI_DataSource(t *testing.T) {
	cases := []struct {
		name         string
		proxyURI     string
		wantBasePath string
		wantPort     int64
	}{
		{"non-8081 recovers base_path and port", "http://0.0.0.0:8082/graphql-api", "graphql-api", 8082},
		{"legacy 8081", "http://0.0.0.0:8081/my-api", "my-api", 8081},
		{"root path", "http://0.0.0.0:8082/", "", 8082},
		{"short port digit count does not corrupt base_path", "http://0.0.0.0:80/svc", "svc", 80},
		{"no port defaults to 8081", "http://host/only-path", "only-path", 8081},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			bp, port := parseProxyURI(tc.proxyURI)
			if bp != tc.wantBasePath {
				t.Errorf("base_path = %q, want %q", bp, tc.wantBasePath)
			}
			if port != tc.wantPort {
				t.Errorf("port = %d, want %d", port, tc.wantPort)
			}
		})
	}
}

// TestAPIInstanceSingleDataSource_Read_customPort exercises the full data-source
// Read path for an instance whose proxyUri is on a non-default port (the user's
// live 8082 case). base_path must be the path only (NOT the whole URI) and the
// new port attribute must surface 8082.
func TestAPIInstanceSingleDataSource_Read_customPort(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/test-org-id/environments/test-env-id/apis/21054006"

	mockInstance := &apimgmtclient.APIInstance{
		ID:            21054006,
		AssetID:       "lifecycle-test-graphql-api",
		AssetVersion:  "2.0.0",
		Technology:    "flexGateway",
		InstanceLabel: "tf-smg-onefile-graphql",
		Status:        "active",
		Endpoint: &apimgmtclient.APIInstanceEndpoint{
			DeploymentType: "HY",
			Type:           "graphql",
			ProxyURI:       strPtr("http://0.0.0.0:8082/graphql-api"),
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockInstance)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIInstanceSingleDataSource().(*APIInstanceSingleDataSource)
	ds.client = &apimgmtclient.APIInstanceClient{
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
		"id":                tftypes.NewValue(tftypes.String, "21054006"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"technology":        tftypes.NewValue(tftypes.String, nil),
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
	var got APIInstanceSingleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	epAttrs := got.Endpoint.Attributes()
	if bp, ok := epAttrs["base_path"].(interface{ ValueString() string }); !ok || bp.ValueString() != "graphql-api" {
		t.Errorf("endpoint.base_path = %v, want graphql-api", epAttrs["base_path"])
	}
	if p, ok := epAttrs["port"].(interface{ ValueInt64() int64 }); !ok || p.ValueInt64() != 8082 {
		t.Errorf("endpoint.port = %v, want 8082", epAttrs["port"])
	}
}
