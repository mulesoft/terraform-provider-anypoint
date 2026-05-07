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

func TestNewManagedFlexGatewayDataSource(t *testing.T) {
	ds := NewManagedFlexGatewayDataSource()
	if ds == nil {
		t.Error("NewManagedFlexGatewayDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("ManagedFlexGatewayDataSource should implement DataSourceWithConfigure")
	}
}

func TestManagedFlexGatewayDataSource_Metadata(t *testing.T) {
	ds := NewManagedFlexGatewayDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "test_managed_omni_gateway" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_managed_omni_gateway")
	}
}

func TestManagedFlexGatewayDataSource_Schema(t *testing.T) {
	ds := NewManagedFlexGatewayDataSource()
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
	computed := []string{"id", "gateways"}
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

func TestManagedFlexGatewayDataSource_Configure(t *testing.T) {
	ds := NewManagedFlexGatewayDataSource().(*ManagedFlexGatewayDataSource)
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

func TestManagedFlexGatewayDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewManagedFlexGatewayDataSource().(*ManagedFlexGatewayDataSource)
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

func TestManagedFlexGatewayDataSourceModel_Validation(t *testing.T) {
	model := ManagedFlexGatewayDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.Gateways
}

func TestManagedFlexGatewayDataSource_Read(t *testing.T) {
	basePath := "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways"

	mockResp := apimgmtclient.ManagedFlexGatewayListResponse{
		Content: []apimgmtclient.ManagedFlexGatewayListItem{
			{ID: "gw-id-1", Name: "gateway-one", Status: "Active"},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockResp)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewManagedFlexGatewayDataSource().(*ManagedFlexGatewayDataSource)
	ds.client = &apimgmtclient.ManagedFlexGatewayClient{
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
	elemType := objType.AttributeTypes["gateways"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"gateways":        tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got ManagedFlexGatewayDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Gateways) != 1 {
		t.Fatalf("Expected 1 gateway, got %d", len(got.Gateways))
	}
}

func TestManagedFlexGatewayDataSource_Read_Error(t *testing.T) {
	basePath := "/gatewaymanager/api/v1/organizations/test-org-id/environments/test-env-id/gateways"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewManagedFlexGatewayDataSource().(*ManagedFlexGatewayDataSource)
	ds.client = &apimgmtclient.ManagedFlexGatewayClient{
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
	elemType := objType.AttributeTypes["gateways"].(tftypes.List).ElementType

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"gateways":        tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func BenchmarkManagedFlexGatewayDataSource_Schema(b *testing.B) {
	ds := NewManagedFlexGatewayDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		ds.Schema(ctx, req, resp)
	}
}

// --- Single datasource ---

func TestNewManagedFlexGatewaySingleDataSource(t *testing.T) {
	ds := NewManagedFlexGatewaySingleDataSource()
	if ds == nil {
		t.Error("NewManagedFlexGatewaySingleDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("ManagedFlexGatewaySingleDataSource should implement DataSourceWithConfigure")
	}
}

func TestManagedFlexGatewaySingleDataSource_Metadata(t *testing.T) {
	ds := NewManagedFlexGatewaySingleDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "test_managed_omni_gateway" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_managed_omni_gateway")
	}
}

func TestManagedFlexGatewaySingleDataSource_Schema(t *testing.T) {
	ds := NewManagedFlexGatewaySingleDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}
	ds.Schema(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	required := []string{"environment_id", "id"}
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
	computed := []string{"organization_id", "name", "target_id"}
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

func TestManagedFlexGatewaySingleDataSource_Configure(t *testing.T) {
	ds := NewManagedFlexGatewaySingleDataSource().(*ManagedFlexGatewaySingleDataSource)
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

func TestManagedFlexGatewaySingleDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewManagedFlexGatewaySingleDataSource().(*ManagedFlexGatewaySingleDataSource)
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

func TestManagedFlexGatewaySingleDataSourceModel_Validation(t *testing.T) {
	model := ManagedFlexGatewaySingleDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.Name
	_ = model.TargetID
	_ = model.RuntimeVersion
	_ = model.ReleaseChannel
	_ = model.Size
}
