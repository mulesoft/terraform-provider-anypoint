package exchange

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewAssetResource(t *testing.T) {
	r := NewAssetResource()

	if r == nil {
		t.Error("NewAssetResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("AssetResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("AssetResource should implement ResourceWithImportState")
	}
}

func TestAssetResource_Metadata(t *testing.T) {
	r := NewAssetResource()
	testutil.TestResourceMetadata(t, r, "_exchange_asset")
}

func TestAssetResource_Schema(t *testing.T) {
	res := NewAssetResource()

	requiredAttrs := []string{"organization_id", "group_id", "asset_id", "version", "name"}
	optionalAttrs := []string{"description", "type", "status", "file_path", "classifier", "keywords", "api_version", "main_file", "contact_name", "contact_email", "manager"}
	computedAttrs := []string{"id", "is_public", "is_snapshot", "minor_version", "version_group", "created_date", "updated_date"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestAssetResource_Configure(t *testing.T) {
	res := NewAssetResource().(*AssetResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-pass",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestAssetResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewAssetResource().(*AssetResource)

	ctx := context.Background()
	req := resource.ConfigureRequest{
		ProviderData: "invalid-data",
	}
	resp := &resource.ConfigureResponse{}

	res.Configure(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}

	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestAssetResourceModel_Validation(t *testing.T) {
	model := AssetResourceModel{}
	_ = model.ID
}

func TestAssetResource_Read(t *testing.T) {
	basePath := "/exchange/api/v2/assets/g/a/1.0.0"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
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

	res := NewAssetResource().(*AssetResource)
	res.client = &exchange.AssetClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "g/a/1.0.0"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"group_id":             tftypes.NewValue(tftypes.String, "g"),
		"asset_id":             tftypes.NewValue(tftypes.String, "a"),
		"version":              tftypes.NewValue(tftypes.String, "1.0.0"),
		"name":                 tftypes.NewValue(tftypes.String, "Test Asset"),
		"description":          tftypes.NewValue(tftypes.String, nil),
		"type":                 tftypes.NewValue(tftypes.String, "custom"),
		"status":               tftypes.NewValue(tftypes.String, "published"),
		"file_path":            tftypes.NewValue(tftypes.String, nil),
		"classifier":           tftypes.NewValue(tftypes.String, nil),
		"keywords":             tftypes.NewValue(tftypes.String, nil),
		"api_version":          tftypes.NewValue(tftypes.String, nil),
		"main_file":            tftypes.NewValue(tftypes.String, nil),
		"contact_name":         tftypes.NewValue(tftypes.String, nil),
		"contact_email":        tftypes.NewValue(tftypes.String, nil),
		"manager":              tftypes.NewValue(tftypes.String, nil),
		"tags":                 tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"pages":                tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"page_name": tftypes.String, "content": tftypes.String, "page_path": tftypes.String}}}, nil),
		"terms_and_conditions": tftypes.NewValue(tftypes.String, nil),
		"instances":            tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String, "endpoint_uri": tftypes.String, "is_public": tftypes.Bool, "instance_id": tftypes.String}}}, nil),
		"categories":           tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"key": tftypes.String, "values": tftypes.List{ElementType: tftypes.String}}}}, nil),
		"custom_fields":        tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"key": tftypes.String, "values": tftypes.List{ElementType: tftypes.String}}}}, nil),
		"file_sha256":          tftypes.NewValue(tftypes.String, nil),
		"is_public":            tftypes.NewValue(tftypes.Bool, false),
		"is_snapshot":          tftypes.NewValue(tftypes.Bool, false),
		"minor_version":        tftypes.NewValue(tftypes.String, "1.0"),
		"version_group":        tftypes.NewValue(tftypes.String, "1.0.0"),
		"created_date":         tftypes.NewValue(tftypes.String, ""),
		"updated_date":         tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got AssetResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "Test Asset" {
		t.Errorf("Expected Name 'Test Asset', got %s", got.Name.ValueString())
	}
}

func TestAssetResource_Read_NotFound(t *testing.T) {
	basePath := "/exchange/api/v2/assets/g/a/1.0.0"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewAssetResource().(*AssetResource)
	res.client = &exchange.AssetClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "g/a/1.0.0"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"group_id":             tftypes.NewValue(tftypes.String, "g"),
		"asset_id":             tftypes.NewValue(tftypes.String, "a"),
		"version":              tftypes.NewValue(tftypes.String, "1.0.0"),
		"name":                 tftypes.NewValue(tftypes.String, "Test Asset"),
		"description":          tftypes.NewValue(tftypes.String, nil),
		"type":                 tftypes.NewValue(tftypes.String, "custom"),
		"status":               tftypes.NewValue(tftypes.String, "published"),
		"file_path":            tftypes.NewValue(tftypes.String, nil),
		"classifier":           tftypes.NewValue(tftypes.String, nil),
		"keywords":             tftypes.NewValue(tftypes.String, nil),
		"api_version":          tftypes.NewValue(tftypes.String, nil),
		"main_file":            tftypes.NewValue(tftypes.String, nil),
		"contact_name":         tftypes.NewValue(tftypes.String, nil),
		"contact_email":        tftypes.NewValue(tftypes.String, nil),
		"manager":              tftypes.NewValue(tftypes.String, nil),
		"tags":                 tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
		"pages":                tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"page_name": tftypes.String, "content": tftypes.String, "page_path": tftypes.String}}}, nil),
		"terms_and_conditions": tftypes.NewValue(tftypes.String, nil),
		"instances":            tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"name": tftypes.String, "endpoint_uri": tftypes.String, "is_public": tftypes.Bool, "instance_id": tftypes.String}}}, nil),
		"categories":           tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"key": tftypes.String, "values": tftypes.List{ElementType: tftypes.String}}}}, nil),
		"custom_fields":        tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"key": tftypes.String, "values": tftypes.List{ElementType: tftypes.String}}}}, nil),
		"file_sha256":          tftypes.NewValue(tftypes.String, nil),
		"is_public":            tftypes.NewValue(tftypes.Bool, false),
		"is_snapshot":          tftypes.NewValue(tftypes.Bool, false),
		"minor_version":        tftypes.NewValue(tftypes.String, "1.0"),
		"version_group":        tftypes.NewValue(tftypes.String, "1.0.0"),
		"created_date":         tftypes.NewValue(tftypes.String, ""),
		"updated_date":         tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

func BenchmarkAssetResource_Schema(b *testing.B) {
	res := NewAssetResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
