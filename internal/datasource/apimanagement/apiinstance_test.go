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

func TestNewAPIInstanceDataSource(t *testing.T) {
	ds := NewAPIInstanceDataSource()
	if ds == nil {
		t.Error("NewAPIInstanceDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("APIInstanceDataSource should implement DataSourceWithConfigure")
	}
}

func TestAPIInstanceDataSource_Metadata(t *testing.T) {
	ds := NewAPIInstanceDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "test_api_instances" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_api_instances")
	}
}

func TestAPIInstanceDataSource_Schema(t *testing.T) {
	ds := NewAPIInstanceDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}
	ds.Schema(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	required := []string{"environment_id"}
	for _, attr := range required {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing required attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("Schema() attribute %q should be required", attr)
		}
	}
	computed := []string{"id", "instances"}
	for _, attr := range computed {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Schema() attribute %q should be computed", attr)
		}
	}
}

func TestAPIInstanceDataSource_Configure(t *testing.T) {
	ds := NewAPIInstanceDataSource().(*APIInstanceDataSource)
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: providerData}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}
	if ds.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestAPIInstanceDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewAPIInstanceDataSource().(*APIInstanceDataSource)
	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: "invalid"}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if ds.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestAPIInstanceDataSourceModel_Validation(t *testing.T) {
	model := APIInstanceDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.Instances
}

func TestAPIInstanceDataSource_Read(t *testing.T) {
	basePath := "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis"

	mockItems := apimgmtclient.APIInstanceListResponse{
		Instances: []apimgmtclient.APIInstance{
			{ID: 1, AssetID: "api-asset-1", Technology: "omniGateway", Status: "Active"},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockItems)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIInstanceDataSource().(*APIInstanceDataSource)
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["instances"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"gateway_id":      tftypes.NewValue(tftypes.String, nil),
		"instances":       tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIInstanceDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Instances) != 1 {
		t.Fatalf("Expected 1 instance, got %d", len(got.Instances))
	}
}

func TestAPIInstanceDataSource_Read_Error(t *testing.T) {
	basePath := "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIInstanceDataSource().(*APIInstanceDataSource)
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["instances"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"gateway_id":      tftypes.NewValue(tftypes.String, nil),
		"instances":       tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

// TestAPIInstanceDataSource_Read_GatewayFilter verifies that setting gateway_id
// narrows the returned instances to only those whose deployment target matches,
// that the per-instance gateway_id output is populated from deployment.targetId,
// and that instances with no deployment are excluded when the filter is set.
func TestAPIInstanceDataSource_Read_GatewayFilter(t *testing.T) {
	basePath := "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis"

	const wantGW = "c3709c68-a429-4e8b-b5e7-c5c214c2a8ea"
	mockItems := apimgmtclient.APIInstanceListResponse{
		Instances: []apimgmtclient.APIInstance{
			{ID: 1, AssetID: "match-a", Status: "Active", Deployment: &apimgmtclient.APIInstanceDeployment{TargetID: wantGW}},
			{ID: 2, AssetID: "other-gw", Status: "Active", Deployment: &apimgmtclient.APIInstanceDeployment{TargetID: "some-other-gateway"}},
			{ID: 3, AssetID: "match-b", Status: "Active", Deployment: &apimgmtclient.APIInstanceDeployment{TargetID: wantGW}},
			{ID: 4, AssetID: "no-deployment", Status: "Active"},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockItems)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIInstanceDataSource().(*APIInstanceDataSource)
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["instances"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"gateway_id":      tftypes.NewValue(tftypes.String, wantGW),
		"instances":       tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIInstanceDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	// Only the two instances targeting wantGW should survive the filter;
	// the other-gateway and no-deployment instances must be excluded.
	if len(got.Instances) != 2 {
		t.Fatalf("Expected 2 instances after gateway filter, got %d: %+v", len(got.Instances), got.Instances)
	}
	for _, inst := range got.Instances {
		if inst.GatewayID.ValueString() != wantGW {
			t.Errorf("instance %s: gateway_id = %q, want %q", inst.ID.ValueString(), inst.GatewayID.ValueString(), wantGW)
		}
		if inst.AssetID.ValueString() == "other-gw" || inst.AssetID.ValueString() == "no-deployment" {
			t.Errorf("instance %s (%s) should have been filtered out", inst.ID.ValueString(), inst.AssetID.ValueString())
		}
	}
}

// TestAPIInstanceDataSource_Read_NoFilterKeepsAll verifies that when gateway_id
// is unset every instance is returned (including those without a deployment),
// and the per-instance gateway_id output is null for undeployed instances.
func TestAPIInstanceDataSource_Read_NoFilterKeepsAll(t *testing.T) {
	basePath := "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/apis"

	const gwID = "c3709c68-a429-4e8b-b5e7-c5c214c2a8ea"
	mockItems := apimgmtclient.APIInstanceListResponse{
		Instances: []apimgmtclient.APIInstance{
			{ID: 1, AssetID: "with-gw", Status: "Active", Deployment: &apimgmtclient.APIInstanceDeployment{TargetID: gwID}},
			{ID: 2, AssetID: "no-deployment", Status: "Active"},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockItems)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAPIInstanceDataSource().(*APIInstanceDataSource)
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
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["instances"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"gateway_id":      tftypes.NewValue(tftypes.String, nil),
		"instances":       tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got APIInstanceDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Instances) != 2 {
		t.Fatalf("Expected all 2 instances with no filter, got %d", len(got.Instances))
	}
	for _, inst := range got.Instances {
		switch inst.AssetID.ValueString() {
		case "with-gw":
			if inst.GatewayID.ValueString() != gwID {
				t.Errorf("with-gw: gateway_id = %q, want %q", inst.GatewayID.ValueString(), gwID)
			}
		case "no-deployment":
			if !inst.GatewayID.IsNull() {
				t.Errorf("no-deployment: gateway_id should be null, got %q", inst.GatewayID.ValueString())
			}
		}
	}
}

func BenchmarkAPIInstanceDataSource_Schema(b *testing.B) {
	ds := NewAPIInstanceDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		ds.Schema(ctx, req, resp)
	}
}
