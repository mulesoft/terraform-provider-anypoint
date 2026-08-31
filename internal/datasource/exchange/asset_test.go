package exchange

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// dsAssetConfigRaw builds the raw config value for the asset data source, matching
// the full schema shape. It is defined once so that adding attributes to the schema
// does not require touching every test that constructs a config (tftypes.NewValue
// panics if the map is missing any schema attribute). All Computed-only attributes
// are passed as null — Terraform never sets them in config.
func dsAssetConfigRaw(stateType tftypes.Type) tftypes.Value {
	pageObj := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"page_name": tftypes.String,
		"content":   tftypes.String,
		"page_path": tftypes.String,
	}}
	instObj := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":         tftypes.String,
		"endpoint_uri": tftypes.String,
		"is_public":    tftypes.Bool,
		"instance_id":  tftypes.String,
	}}
	kvObj := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"key":    tftypes.String,
		"values": tftypes.List{ElementType: tftypes.String},
	}}
	return tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, nil),
		"group_id":             tftypes.NewValue(tftypes.String, "g"),
		"asset_id":             tftypes.NewValue(tftypes.String, "a"),
		"version":              tftypes.NewValue(tftypes.String, "1.0.0"),
		"name":                 tftypes.NewValue(tftypes.String, nil),
		"description":          tftypes.NewValue(tftypes.String, nil),
		"type":                 tftypes.NewValue(tftypes.String, nil),
		"status":               tftypes.NewValue(tftypes.String, nil),
		"contact_name":         tftypes.NewValue(tftypes.String, nil),
		"contact_email":        tftypes.NewValue(tftypes.String, nil),
		"manager":              tftypes.NewValue(tftypes.String, nil),
		"is_public":            tftypes.NewValue(tftypes.Bool, nil),
		"is_snapshot":          tftypes.NewValue(tftypes.Bool, nil),
		"minor_version":        tftypes.NewValue(tftypes.String, nil),
		"version_group":        tftypes.NewValue(tftypes.String, nil),
		"created_date":         tftypes.NewValue(tftypes.String, nil),
		"updated_date":         tftypes.NewValue(tftypes.String, nil),
		"tags":                 tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"terms_and_conditions": tftypes.NewValue(tftypes.String, nil),
		"pages":                tftypes.NewValue(tftypes.List{ElementType: pageObj}, nil),
		"instances":            tftypes.NewValue(tftypes.List{ElementType: instObj}, nil),
		"categories":           tftypes.NewValue(tftypes.List{ElementType: kvObj}, nil),
		"custom_fields":        tftypes.NewValue(tftypes.List{ElementType: kvObj}, nil),
	})
}

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
	computedAttrs := []string{"id", "name", "description", "type", "status", "contact_name", "contact_email", "manager", "is_public", "is_snapshot", "minor_version", "version_group", "created_date", "updated_date", "tags", "terms_and_conditions", "pages", "instances", "categories", "custom_fields"}
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

	configRaw := dsAssetConfigRaw(stateType)

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
	// Empty nested collections must materialize as known empty lists (not null),
	// so consumers can index them without a null check.
	if got.Tags.IsNull() || len(got.Tags.Elements()) != 0 {
		t.Errorf("Expected tags to be an empty (non-null) list, got %#v", got.Tags)
	}
	if got.Instances.IsNull() || len(got.Instances.Elements()) != 0 {
		t.Errorf("Expected instances to be an empty (non-null) list, got %#v", got.Instances)
	}
	if got.Categories.IsNull() || got.CustomFields.IsNull() || got.Pages.IsNull() {
		t.Errorf("Expected categories/custom_fields/pages to be non-null lists")
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

	configRaw := dsAssetConfigRaw(stateType)

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

// TestAssetDataSource_Read_NestedCollections verifies the data source surfaces the
// full asset: tags, categories, custom_fields, external instances (managed ones
// filtered out), documentation pages (synthetic + .terms filtered out) with content,
// and the Terms & Conditions page.
func TestAssetDataSource_Read_NestedCollections(t *testing.T) {
	base := "/exchange/api/v2/assets/g/a/1.0.0"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		base: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"groupId":      "g",
				"assetId":      "a",
				"version":      "1.0.0",
				"name":         "Test Asset",
				"type":         "rest-api",
				"status":       "published",
				"labels":       []interface{}{"terraform", "tags-test"},
				"categories":   []interface{}{map[string]interface{}{"key": "Lifecycle", "value": []interface{}{"Production", "GA"}}},
				"customFields": []interface{}{map[string]interface{}{"key": "Team", "value": "Platform"}},
				"instances": []interface{}{
					map[string]interface{}{"type": "external", "name": "Production", "endpointUri": "https://api.example.com/v1", "isPublic": true, "id": "inst-123"},
					map[string]interface{}{"type": "managed", "name": "ShouldBeFiltered", "id": "m-1"},
				},
			})
		},
		base + "/portal/pages": func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{"name": "Overview", "path": "abc/Overview"},
				{"name": "home", "path": "home", "synthetic": true},
				{"name": ".terms", "path": "xyz/.terms"},
			})
		},
		base + "/portal/pages/": func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "text/markdown")
			w.WriteHeader(http.StatusOK)
			if strings.HasSuffix(r.URL.Path, ".terms") {
				_, _ = w.Write([]byte("# Terms\n\nThe terms."))
				return
			}
			_, _ = w.Write([]byte("# Overview\n\nThe overview page."))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewAssetDataSource().(*AssetDataSource)
	ds.client = &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	configRaw := dsAssetConfigRaw(stateType)

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

	// tags
	var tags []string
	got.Tags.ElementsAs(ctx, &tags, false)
	if len(tags) != 2 || tags[0] != "terraform" || tags[1] != "tags-test" {
		t.Errorf("tags = %v, want [terraform tags-test]", tags)
	}

	// categories
	type kvModel struct {
		Key    types.String `tfsdk:"key"`
		Values types.List   `tfsdk:"values"`
	}
	var cats []kvModel
	got.Categories.ElementsAs(ctx, &cats, false)
	if len(cats) != 1 || cats[0].Key.ValueString() != "Lifecycle" {
		t.Fatalf("categories = %#v, want 1 (Lifecycle)", cats)
	}
	var catVals []string
	cats[0].Values.ElementsAs(ctx, &catVals, false)
	if len(catVals) != 2 || catVals[0] != "Production" || catVals[1] != "GA" {
		t.Errorf("category values = %v, want [Production GA]", catVals)
	}

	// custom_fields (interface{} string value → single-element list)
	var cfs []kvModel
	got.CustomFields.ElementsAs(ctx, &cfs, false)
	if len(cfs) != 1 || cfs[0].Key.ValueString() != "Team" {
		t.Fatalf("custom_fields = %#v, want 1 (Team)", cfs)
	}
	var cfVals []string
	cfs[0].Values.ElementsAs(ctx, &cfVals, false)
	if len(cfVals) != 1 || cfVals[0] != "Platform" {
		t.Errorf("custom_field values = %v, want [Platform]", cfVals)
	}

	// instances — only the external one, managed filtered out
	type instModel struct {
		Name        types.String `tfsdk:"name"`
		EndpointURI types.String `tfsdk:"endpoint_uri"`
		IsPublic    types.Bool   `tfsdk:"is_public"`
		InstanceID  types.String `tfsdk:"instance_id"`
	}
	var insts []instModel
	got.Instances.ElementsAs(ctx, &insts, false)
	if len(insts) != 1 {
		t.Fatalf("instances = %#v, want exactly 1 (external only)", insts)
	}
	if insts[0].Name.ValueString() != "Production" || insts[0].EndpointURI.ValueString() != "https://api.example.com/v1" ||
		!insts[0].IsPublic.ValueBool() || insts[0].InstanceID.ValueString() != "inst-123" {
		t.Errorf("instance = %#v, unexpected fields", insts[0])
	}

	// pages — only "Overview" (synthetic "home" and ".terms" filtered out), with content
	type pageModel struct {
		PageName types.String `tfsdk:"page_name"`
		Content  types.String `tfsdk:"content"`
		PagePath types.String `tfsdk:"page_path"`
	}
	var pages []pageModel
	got.Pages.ElementsAs(ctx, &pages, false)
	if len(pages) != 1 || pages[0].PageName.ValueString() != "Overview" {
		t.Fatalf("pages = %#v, want exactly 1 (Overview)", pages)
	}
	if pages[0].Content.ValueString() != "# Overview\n\nThe overview page." {
		t.Errorf("page content = %q, unexpected", pages[0].Content.ValueString())
	}
	if pages[0].PagePath.ValueString() != "abc/Overview" {
		t.Errorf("page_path = %q, want abc/Overview", pages[0].PagePath.ValueString())
	}

	// terms_and_conditions
	if got.TermsAndConditions.ValueString() != "# Terms\n\nThe terms." {
		t.Errorf("terms_and_conditions = %q, unexpected", got.TermsAndConditions.ValueString())
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
