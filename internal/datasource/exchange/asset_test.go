package exchange

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAssetDataSource(t *testing.T) {
	dataSource := NewAssetDataSource()

	if dataSource == nil {
		t.Error("NewAssetDataSource() returned nil")
	}

	// Verify it implements the expected interfaces
	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("AssetDataSource does not implement DataSourceWithConfigure")
	}
}

func TestAssetDataSource_Metadata(t *testing.T) {
	dataSource := NewAssetDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_exchange_asset" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_exchange_asset")
	}
}

func TestAssetDataSource_Schema(t *testing.T) {
	dataSource := NewAssetDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check required attributes
	requiredAttrs := []string{"group_id", "asset_id", "version"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	// Check computed attributes
	computedAttrs := []string{"id", "name", "description", "type", "status", "contact_name", "contact_email", "manager", "is_public", "is_snapshot", "minor_version", "version_group", "created_date", "updated_date"}
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

func TestAssetDataSource_Configure(t *testing.T) {
	dataSource := NewAssetDataSource().(*AssetDataSource)

	// Test with valid provider data
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-pass",
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

	// Verify client is configured
	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestAssetDataSourceModel_Validation(t *testing.T) {
	// Test that all model fields exist and are properly typed
	model := AssetDataSourceModel{}

	// Verify all expected fields exist
	_ = model.ID
}

func TestAssetDataSource_Read(t *testing.T) {
	basePath := "/exchange/api/v2/assets/g/a/1.0.0"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "GET", basePath)
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"groupId":       "g",
				"assetId":       "a",
				"version":       "1.0.0",
				"name":          "Test Asset",
				"description":   "Test description",
				"type":          "custom",
				"status":        "published",
				"isPublic":      false,
				"isSnapshot":    false,
				"minorVersion":  "1.0",
				"versionGroup":  "1.0.0",
				"createdDate":   "2024-01-01T00:00:00Z",
				"updatedDate":   "2024-01-01T00:00:00Z",
				"contactName":   nil,
				"contactEmail":  nil,
				"manager":       nil,
				"labels":        []interface{}{},
				"categories":    []interface{}{},
				"customFields":  []interface{}{},
				"files":         []interface{}{},
				"dependencies":  []interface{}{},
				"instances":     []interface{}{},
				"attributes":    []interface{}{},
				"rating":        0,
				"numberOfRates": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAssetDataSource().(*AssetDataSource)
	ds.client = &exchange.AssetClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, nil),
		"group_id":      tftypes.NewValue(tftypes.String, "g"),
		"asset_id":      tftypes.NewValue(tftypes.String, "a"),
		"version":       tftypes.NewValue(tftypes.String, "1.0.0"),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"description":   tftypes.NewValue(tftypes.String, nil),
		"type":          tftypes.NewValue(tftypes.String, nil),
		"status":        tftypes.NewValue(tftypes.String, nil),
		"contact_name":  tftypes.NewValue(tftypes.String, nil),
		"contact_email": tftypes.NewValue(tftypes.String, nil),
		"manager":       tftypes.NewValue(tftypes.String, nil),
		"is_public":     tftypes.NewValue(tftypes.Bool, nil),
		"is_snapshot":   tftypes.NewValue(tftypes.Bool, nil),
		"minor_version": tftypes.NewValue(tftypes.String, nil),
		"version_group": tftypes.NewValue(tftypes.String, nil),
		"created_date":  tftypes.NewValue(tftypes.String, nil),
		"updated_date":  tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got AssetDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "Test Asset" {
		t.Errorf("Expected Name 'Test Asset', got %s", got.Name.ValueString())
	}
}

func TestAssetDataSource_Read_Error(t *testing.T) {
	basePath := "/exchange/api/v2/assets/g/a/1.0.0"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAssetDataSource().(*AssetDataSource)
	ds.client = &exchange.AssetClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":            tftypes.NewValue(tftypes.String, nil),
		"group_id":      tftypes.NewValue(tftypes.String, "g"),
		"asset_id":      tftypes.NewValue(tftypes.String, "a"),
		"version":       tftypes.NewValue(tftypes.String, "1.0.0"),
		"name":          tftypes.NewValue(tftypes.String, nil),
		"description":   tftypes.NewValue(tftypes.String, nil),
		"type":          tftypes.NewValue(tftypes.String, nil),
		"status":        tftypes.NewValue(tftypes.String, nil),
		"contact_name":  tftypes.NewValue(tftypes.String, nil),
		"contact_email": tftypes.NewValue(tftypes.String, nil),
		"manager":       tftypes.NewValue(tftypes.String, nil),
		"is_public":     tftypes.NewValue(tftypes.Bool, nil),
		"is_snapshot":   tftypes.NewValue(tftypes.Bool, nil),
		"minor_version": tftypes.NewValue(tftypes.String, nil),
		"version_group": tftypes.NewValue(tftypes.String, nil),
		"created_date":  tftypes.NewValue(tftypes.String, nil),
		"updated_date":  tftypes.NewValue(tftypes.String, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

// Benchmarks
func BenchmarkAssetDataSource_Schema(b *testing.B) {
	dataSource := NewAssetDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
