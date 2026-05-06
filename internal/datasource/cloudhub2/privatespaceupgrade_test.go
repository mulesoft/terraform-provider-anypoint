package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewPrivateSpaceUpgradeDataSource(t *testing.T) {
	dataSource := NewPrivateSpaceUpgradeDataSource()

	if dataSource == nil {
		t.Error("NewPrivateSpaceUpgradeDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("does not implement DataSourceWithConfigure")
	}
}

func TestPrivateSpaceUpgradeDataSource_Metadata(t *testing.T) {
	dataSource := NewPrivateSpaceUpgradeDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_private_space_upgrade" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_private_space_upgrade")
	}
}

func TestPrivateSpaceUpgradeDataSource_Schema(t *testing.T) {
	dataSource := NewPrivateSpaceUpgradeDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	requiredAttrs := []string{"private_space_id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	computedAttrs := []string{"id", "scheduled_update_time", "status"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}
}

func TestPrivateSpaceUpgradeDataSource_Configure(t *testing.T) {
	dataSource := NewPrivateSpaceUpgradeDataSource().(*PrivateSpaceUpgradeDataSource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: providerData,
	}
	resp := &datasource.ConfigureResponse{}

	dataSource.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}

	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestPrivateSpaceUpgradeDataSourceModel_Validation(t *testing.T) {
	model := PrivateSpaceUpgradeDataSourceModel{}
	_ = model.ID
}

func TestPrivateSpaceUpgradeDataSource_Read(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/upgradestatus"

	mockResp := ch2client.PrivateSpaceUpgradeResponse{
		ScheduledUpdateTime: "2024-01-15T10:00:00Z",
		Status:              "pending",
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockResp)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewPrivateSpaceUpgradeDataSource().(*PrivateSpaceUpgradeDataSource)
	ds.client = &ch2client.PrivateSpaceUpgradeClient{
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
		"id":                    tftypes.NewValue(tftypes.String, nil),
		"private_space_id":      tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":       tftypes.NewValue(tftypes.String, "test-org-id"),
		"scheduled_update_time": tftypes.NewValue(tftypes.String, nil),
		"status":                tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got PrivateSpaceUpgradeDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Status.ValueString() != "pending" {
		t.Errorf("Expected Status pending, got %s", got.Status.ValueString())
	}
}

func TestPrivateSpaceUpgradeDataSource_Read_Error(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/upgradestatus"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewPrivateSpaceUpgradeDataSource().(*PrivateSpaceUpgradeDataSource)
	ds.client = &ch2client.PrivateSpaceUpgradeClient{
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
		"id":                    tftypes.NewValue(tftypes.String, nil),
		"private_space_id":      tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":       tftypes.NewValue(tftypes.String, "test-org-id"),
		"scheduled_update_time": tftypes.NewValue(tftypes.String, nil),
		"status":                tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

func BenchmarkPrivateSpaceUpgradeDataSource_Schema(b *testing.B) {
	dataSource := NewPrivateSpaceUpgradeDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
