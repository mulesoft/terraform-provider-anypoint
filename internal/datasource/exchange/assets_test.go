package exchange

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAssetsDataSource(t *testing.T) {
	dataSource := NewAssetsDataSource()

	if dataSource == nil {
		t.Error("NewAssetsDataSource() returned nil")
	}

	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("AssetsDataSource does not implement DataSourceWithConfigure")
	}
}

func TestAssetsDataSource_Metadata(t *testing.T) {
	dataSource := NewAssetsDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "anypoint",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "anypoint_exchange_assets" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "anypoint_exchange_assets")
	}
}

func TestAssetsDataSource_Schema(t *testing.T) {
	dataSource := NewAssetsDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check required attributes
	if attr, exists := resp.Schema.Attributes["organization_id"]; exists {
		if !attr.IsRequired() {
			t.Error("Schema() attribute organization_id should be required")
		}
	} else {
		t.Error("Schema() missing required attribute: organization_id")
	}

	// Check optional attributes
	optionalAttrs := []string{"type", "search", "limit"}
	for _, attrName := range optionalAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsOptional() {
				t.Errorf("Schema() attribute %s should be optional", attrName)
			}
		} else {
			t.Errorf("Schema() missing optional attribute: %s", attrName)
		}
	}

	// Check computed attribute (assets list)
	if attr, exists := resp.Schema.Attributes["assets"]; exists {
		if !attr.IsComputed() {
			t.Error("Schema() attribute assets should be computed")
		}
	} else {
		t.Error("Schema() missing computed attribute: assets")
	}
}

func TestAssetsDataSource_Configure(t *testing.T) {
	dataSource := NewAssetsDataSource().(*AssetsDataSource)

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

	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestAssetsDataSource_Configure_InvalidProviderData(t *testing.T) {
	dataSource := NewAssetsDataSource().(*AssetsDataSource)

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: "invalid-data",
	}
	resp := &datasource.ConfigureResponse{}

	dataSource.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}
}

func TestAssetsDataSource_Read(t *testing.T) {
	listPath := "/exchange/api/v2/assets"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			// Verify query params
			if r.URL.Query().Get("organizationId") != "test-org" {
				t.Errorf("Expected organizationId=test-org, got %s", r.URL.Query().Get("organizationId"))
			}
			if r.URL.Query().Get("type") != "rest-api" {
				t.Errorf("Expected type=rest-api, got %s", r.URL.Query().Get("type"))
			}

			assets := []map[string]interface{}{
				{
					"groupId":     "test-org",
					"assetId":     "my-api",
					"version":     "1.0.0",
					"name":        "My API",
					"description": "A test API",
					"type":        "rest-api",
					"status":      "published",
					"isPublic":    false,
					"createdDate": "2024-01-01T00:00:00Z",
					"updatedDate": "2024-01-02T00:00:00Z",
				},
				{
					"groupId":     "test-org",
					"assetId":     "another-api",
					"version":     "2.0.0",
					"name":        "Another API",
					"description": "Another test API",
					"type":        "rest-api",
					"status":      "published",
					"isPublic":    true,
					"createdDate": "2024-02-01T00:00:00Z",
					"updatedDate": "2024-02-02T00:00:00Z",
				},
			}

			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(assets)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAssetsDataSource().(*AssetsDataSource)
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
		"organization_id": tftypes.NewValue(tftypes.String, "test-org"),
		"type":            tftypes.NewValue(tftypes.String, "rest-api"),
		"search":          tftypes.NewValue(tftypes.String, nil),
		"limit":           tftypes.NewValue(tftypes.Number, nil),
		"assets": tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"group_id":     tftypes.String,
			"asset_id":     tftypes.String,
			"version":      tftypes.String,
			"name":         tftypes.String,
			"description":  tftypes.String,
			"type":         tftypes.String,
			"status":       tftypes.String,
			"is_public":    tftypes.Bool,
			"created_date": tftypes.String,
			"updated_date": tftypes.String,
		}}}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got AssetsDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.Assets.IsNull() || got.Assets.IsUnknown() {
		t.Fatal("Expected assets list to be set")
	}

	elements := got.Assets.Elements()
	if len(elements) != 2 {
		t.Errorf("Expected 2 assets, got %d", len(elements))
	}
}

func TestAssetsDataSource_Read_Empty(t *testing.T) {
	listPath := "/exchange/api/v2/assets"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode([]map[string]interface{}{})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAssetsDataSource().(*AssetsDataSource)
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
		"organization_id": tftypes.NewValue(tftypes.String, "test-org"),
		"type":            tftypes.NewValue(tftypes.String, nil),
		"search":          tftypes.NewValue(tftypes.String, nil),
		"limit":           tftypes.NewValue(tftypes.Number, nil),
		"assets": tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"group_id":     tftypes.String,
			"asset_id":     tftypes.String,
			"version":      tftypes.String,
			"name":         tftypes.String,
			"description":  tftypes.String,
			"type":         tftypes.String,
			"status":       tftypes.String,
			"is_public":    tftypes.Bool,
			"created_date": tftypes.String,
			"updated_date": tftypes.String,
		}}}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got AssetsDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	elements := got.Assets.Elements()
	if len(elements) != 0 {
		t.Errorf("Expected 0 assets, got %d", len(elements))
	}
}

func TestAssetsDataSource_Read_Error(t *testing.T) {
	listPath := "/exchange/api/v2/assets"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAssetsDataSource().(*AssetsDataSource)
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
		"organization_id": tftypes.NewValue(tftypes.String, "test-org"),
		"type":            tftypes.NewValue(tftypes.String, nil),
		"search":          tftypes.NewValue(tftypes.String, nil),
		"limit":           tftypes.NewValue(tftypes.Number, nil),
		"assets": tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{
			"group_id":     tftypes.String,
			"asset_id":     tftypes.String,
			"version":      tftypes.String,
			"name":         tftypes.String,
			"description":  tftypes.String,
			"type":         tftypes.String,
			"status":       tftypes.String,
			"is_public":    tftypes.Bool,
			"created_date": tftypes.String,
			"updated_date": tftypes.String,
		}}}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}
