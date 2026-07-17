package exchange

import (
	"context"
	"io"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
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
		"additional_file":      tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"path": tftypes.String, "classifier": tftypes.String}}}, nil),
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
		"additional_file":      tftypes.NewValue(tftypes.List{ElementType: tftypes.Object{AttributeTypes: map[string]tftypes.Type{"path": tftypes.String, "classifier": tftypes.String}}}, nil),
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

// TestAssetResource_Read_MultiFile_PreservesAdditionalFileAndClassifier is the
// core Read-path regression for task #103. When additional_file is set (a policy
// asset published with schema.json + metadata.yaml), Read must:
//
//	(1) preserve additional_file verbatim — it is a local, upload-only field the
//	    API never echoes back, so reconciling it would drift/wipe it; and
//	(2) NOT run the extractFileMetadata first-file heuristic, which is ambiguous
//	    with multiple user files. Here the API's files array lists "metadata"
//	    FIRST, so a broken guard would overwrite the configured classifier
//	    ("schema") with "metadata" and force perpetual drift. The multi-file
//	    guard must keep classifier/main_file frozen to their configured values.
func TestAssetResource_Read_MultiFile_PreservesAdditionalFileAndClassifier(t *testing.T) {
	basePath := "/exchange/api/v2/assets/g/a/1.0.0"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"groupId":     "g",
				"assetId":     "a",
				"version":     "1.0.0",
				"name":        "E2E policy",
				"description": "policy",
				"type":        "policy",
				"status":      "published",
				"isPublic":    false,
				"isSnapshot":  false,
				// files array lists metadata FIRST on purpose: extractFileMetadata
				// would pick it and clobber the configured classifier="schema" if
				// the multi-file guard were not skipping reconciliation.
				"files": []interface{}{
					map[string]interface{}{"classifier": "metadata", "packaging": "yaml", "mainFile": "metadata.yaml", "isGenerated": false},
					map[string]interface{}{"classifier": "schema", "packaging": "json", "mainFile": "schema.json", "isGenerated": false},
				},
				"labels":       []interface{}{},
				"categories":   []interface{}{},
				"customFields": []interface{}{},
				"dependencies": []interface{}{},
				"instances":    []interface{}{},
				"attributes":   []interface{}{},
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

	additionalFileElemType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{"path": tftypes.String, "classifier": tftypes.String}}
	additionalFileVal := tftypes.NewValue(
		tftypes.List{ElementType: additionalFileElemType},
		[]tftypes.Value{
			tftypes.NewValue(additionalFileElemType, map[string]tftypes.Value{
				"path":       tftypes.NewValue(tftypes.String, "specs/metadata.yaml"),
				"classifier": tftypes.NewValue(tftypes.String, "metadata"),
			}),
		},
	)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "g/a/1.0.0"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"group_id":             tftypes.NewValue(tftypes.String, "g"),
		"asset_id":             tftypes.NewValue(tftypes.String, "a"),
		"version":              tftypes.NewValue(tftypes.String, "1.0.0"),
		"name":                 tftypes.NewValue(tftypes.String, "E2E policy"),
		"description":          tftypes.NewValue(tftypes.String, "policy"),
		"type":                 tftypes.NewValue(tftypes.String, "policy"),
		"status":               tftypes.NewValue(tftypes.String, "published"),
		"file_path":            tftypes.NewValue(tftypes.String, "specs/schema.json"),
		"additional_file":      additionalFileVal,
		"classifier":           tftypes.NewValue(tftypes.String, "schema"),
		"keywords":             tftypes.NewValue(tftypes.String, nil),
		"api_version":          tftypes.NewValue(tftypes.String, nil),
		"main_file":            tftypes.NewValue(tftypes.String, "schema.json"),
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

	// (2) classifier/main_file must be frozen to config, NOT overwritten by the
	// first API file ("metadata").
	if got.Classifier.ValueString() != "schema" {
		t.Errorf("classifier = %q, want \"schema\" (multi-file guard must skip extractFileMetadata; got the first API file instead)", got.Classifier.ValueString())
	}
	if got.MainFile.ValueString() != "schema.json" {
		t.Errorf("main_file = %q, want \"schema.json\"", got.MainFile.ValueString())
	}

	// (1) additional_file must survive Read verbatim.
	if got.AdditionalFiles.IsNull() || got.AdditionalFiles.IsUnknown() {
		t.Fatalf("additional_file was not preserved across Read: %v", got.AdditionalFiles)
	}
	var addl []AdditionalFileModel
	if diags := got.AdditionalFiles.ElementsAs(ctx, &addl, false); diags.HasError() {
		t.Fatalf("failed to decode additional_file: %v", diags)
	}
	if len(addl) != 1 {
		t.Fatalf("expected 1 additional_file entry, got %d: %+v", len(addl), addl)
	}
	if addl[0].Path.ValueString() != "specs/metadata.yaml" || addl[0].Classifier.ValueString() != "metadata" {
		t.Errorf("additional_file[0] = {path:%q classifier:%q}, want {specs/metadata.yaml metadata}", addl[0].Path.ValueString(), addl[0].Classifier.ValueString())
	}
}

// TestNormalizeClassifier_FatPrefix guards against the perpetual-drift bug found
// during E2E of the API-spec-fragment type: the Exchange API bundles uploaded
// spec files and stores their classifier with a "fat-" prefix (raml -> fat-raml,
// oas -> fat-oas, raml-fragment -> fat-raml-fragment, oas-components ->
// fat-oas-components, ...). If the provider does not map "fat-<x>" back to the
// user's "<x>", every plan shows classifier drift and forces replacement.
//
// This previously slipped because normalizeClassifier hardcoded only fat-raml
// and fat-oas and had NO unit test — so fat-raml-fragment regressed silently.
func TestNormalizeClassifier_FatPrefix(t *testing.T) {
	tests := []struct {
		name          string
		apiClassifier string
		stateValue    types.String // user's configured/imported classifier ("" => null, e.g. import)
		want          string
	}{
		// The exact drift case that E2E caught.
		{"fragment reconcile", "fat-raml-fragment", types.StringValue("raml-fragment"), "raml-fragment"},
		{"fragment import", "fat-raml-fragment", types.StringNull(), "raml-fragment"},
		// Pre-existing cases must still hold.
		{"raml reconcile", "fat-raml", types.StringValue("raml"), "raml"},
		{"raml import", "fat-raml", types.StringNull(), "raml"},
		{"oas reconcile", "fat-oas", types.StringValue("oas"), "oas"},
		{"oas import", "fat-oas", types.StringNull(), "oas"},
		// Future OAS fragment family — should work generically with no code change.
		{"oas-components reconcile", "fat-oas-components", types.StringValue("oas-components"), "oas-components"},
		{"oas-components import", "fat-oas-components", types.StringNull(), "oas-components"},
		// Non-bundled classifiers must pass through unchanged (no spurious trimming).
		{"wsdl passthrough", "wsdl", types.StringValue("wsdl"), "wsdl"},
		{"graphql passthrough", "graphql", types.StringValue("graphql"), "graphql"},
		{"proto passthrough", "proto", types.StringNull(), "proto"},
		{"json-schema passthrough", "json-schema", types.StringNull(), "json-schema"},
		{"custom passthrough", "custom", types.StringNull(), "custom"},
		// API returns exactly what user set (no bundling) — preserve it.
		{"exact match preserved", "raml-fragment", types.StringValue("raml-fragment"), "raml-fragment"},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := normalizeClassifier(tc.apiClassifier, tc.stateValue)
			if got != tc.want {
				t.Errorf("normalizeClassifier(%q, %v) = %q; want %q",
					tc.apiClassifier, tc.stateValue, got, tc.want)
			}
		})
	}
}

// TestAPIClassifierToUserClassifier verifies the raw inverse used at import time.
func TestAPIClassifierToUserClassifier(t *testing.T) {
	cases := map[string]string{
		"fat-raml":           "raml",
		"fat-oas":            "oas",
		"fat-raml-fragment":  "raml-fragment",
		"fat-oas-components": "oas-components",
		"wsdl":               "wsdl",
		"graphql":            "graphql",
		"proto":              "proto",
		"json-schema":        "json-schema",
	}
	for in, want := range cases {
		if got := apiClassifierToUserClassifier(in); got != want {
			t.Errorf("apiClassifierToUserClassifier(%q) = %q; want %q", in, got, want)
		}
	}
}

// TestSyncInstances_NoRePostExisting guards against the external-instance recreate
// bug found during E2E: external instances live in api-metadata-service keyed by
// versionGroup and are NOT cascade-deleted with the asset version. On a recreate at
// the same versionGroup, the pre-existing ("orphan") instances are still attached, so
// blindly POSTing them again returns 409 API_METADATA_EXTERNAL_API_CONFLICT
// ("instance ... already exists"). syncInstances must treat instances passed as
// "current" as already-present and NOT re-create them.
func TestSyncInstances_NoRePostExisting(t *testing.T) {
	var posts, patches, deletes int
	basePrefix := "/exchange/api/v2/assets/g/a/versionGroups/v1/instances/external"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePrefix: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				posts++
				testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
					"id": "new-id", "name": "New", "type": "external",
				})
			default:
				w.WriteHeader(http.StatusOK)
			}
		},
		basePrefix + "/": func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPatch:
				patches++
				w.WriteHeader(http.StatusOK)
			case http.MethodDelete:
				deletes++
				w.WriteHeader(http.StatusNoContent)
			default:
				w.WriteHeader(http.StatusOK)
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewAssetResource().(*AssetResource)
	res.client = &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

	// current: "Production" already exists on the versionGroup (orphan from prior delete),
	//          "Gone" exists but is no longer desired.
	current := []InstanceModel{
		{Name: types.StringValue("Production"), EndpointURI: types.StringValue("https://p"), IsPublic: types.BoolValue(false), InstanceID: types.StringValue("prod-id")},
		{Name: types.StringValue("Gone"), EndpointURI: types.StringValue("https://g"), IsPublic: types.BoolValue(false), InstanceID: types.StringValue("gone-id")},
	}
	// desired: keep "Production" unchanged, add "New".
	desired := []InstanceModel{
		{Name: types.StringValue("Production"), EndpointURI: types.StringValue("https://p"), IsPublic: types.BoolValue(false)},
		{Name: types.StringValue("New"), EndpointURI: types.StringValue("https://n"), IsPublic: types.BoolValue(false)},
	}

	if err := res.syncInstances(context.Background(), "g", "a", "v1", current, desired); err != nil {
		t.Fatalf("syncInstances returned error: %v", err)
	}

	// "Production" is unchanged & already exists -> must NOT be re-POSTed (the 409 bug).
	// Only "New" should be created.
	if posts != 1 {
		t.Errorf("expected exactly 1 POST (only 'New'), got %d — existing instance was re-created (409 bug)", posts)
	}
	// "Gone" is no longer desired -> must be deleted.
	if deletes != 1 {
		t.Errorf("expected exactly 1 DELETE (only 'Gone'), got %d", deletes)
	}
	// "Production" unchanged -> no PATCH.
	if patches != 0 {
		t.Errorf("expected 0 PATCH (nothing changed), got %d", patches)
	}
}

// TestAssetResource_ChurnFields_UseStateForUnknown guards against the nested-computed
// churn bug reproduced LIVE on prod: editing ONE field (e.g. a single tag) forced six
// unrelated Computed / Optional+Computed attributes to "(known after apply)" on every
// in-place update, because they lacked the UseStateForUnknown() plan modifier —
// pages, instances, categories, custom_fields, terms_and_conditions, updated_date.
//
// Why it slipped: all prior E2E ran create -> import -> delete. A fresh create legitimately
// shows everything as "known after apply", so the noise was invisible. The bug only
// surfaces on UPDATE-then-plan (change one field, assert the plan shows ONLY that field).
// This test institutionalizes that missing "mutate-one-field -> one-line plan" check by
// driving each attribute's plan modifiers with the exact churn scenario:
//
//	prior state = known value (what read*IntoState writes: a known-EMPTY list, a known
//	              timestamp, or a null string) ; plan = unknown ; config = null
//
// and asserting the modifier copies prior state into the plan instead of leaving it
// unknown. If a modifier is removed, PlanValue stays unknown and this test fails.
func TestAssetResource_ChurnFields_UseStateForUnknown(t *testing.T) {
	res := NewAssetResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	attrs := schemaResp.Schema.Attributes

	// A non-null resource state so UseStateForUnknown does NOT bail — it only bails when
	// the WHOLE resource state is null (create). The struct/type here is irrelevant; the
	// modifier checks only req.State.Raw.IsNull().
	nonNullState := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")}

	// List-typed churn fields. read*IntoState writes a KNOWN empty list when the asset has
	// none (types.ListValueMust(..., []attr.Value{})), so the modifier reuses it.
	listFields := []string{"pages", "instances", "categories", "custom_fields"}
	for _, name := range listFields {
		t.Run(name, func(t *testing.T) {
			a := attrs[name]
			lna, ok := a.(schema.ListNestedAttribute)
			if !ok {
				t.Fatalf("%s: expected ListNestedAttribute, got %T", name, a)
			}
			if len(lna.PlanModifiers) == 0 {
				t.Fatalf("%s: NO plan modifiers — churn regression (missing UseStateForUnknown)", name)
			}
			lt, ok := a.GetType().(types.ListType)
			if !ok {
				t.Fatalf("%s: expected list type, got %T", name, a.GetType())
			}
			elemType := lt.ElementType()

			priorState := types.ListValueMust(elemType, []attr.Value{}) // known empty list
			req := planmodifier.ListRequest{
				State:       nonNullState,
				StateValue:  priorState,
				PlanValue:   types.ListUnknown(elemType), // framework marks computed unknown on update
				ConfigValue: types.ListNull(elemType),    // unconfigured
			}
			resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}
			for _, m := range lna.PlanModifiers {
				m.PlanModifyList(ctx, req, resp)
			}
			if resp.PlanValue.IsUnknown() {
				t.Errorf("%s: still (known after apply) after plan modifiers — churn NOT fixed", name)
			}
			if !resp.PlanValue.Equal(priorState) {
				t.Errorf("%s: expected plan to reuse prior state %v, got %v", name, priorState, resp.PlanValue)
			}
		})
	}

	// String-typed churn field: terms_and_conditions. Its prior state is "" once the
	// asset has been created/read (readTermsIntoState writes "" when there is no T&C
	// page — never null), and an unrelated in-place edit does not change it, so
	// UseStateForUnknown is a safe display-only fix: the planned prior value equals the
	// value the post-apply Read re-derives.
	t.Run("terms_and_conditions", func(t *testing.T) {
		const name = "terms_and_conditions"
		priorState := types.StringValue("")
		a := attrs[name]
		sa, ok := a.(schema.StringAttribute)
		if !ok {
			t.Fatalf("%s: expected StringAttribute, got %T", name, a)
		}
		if len(sa.PlanModifiers) == 0 {
			t.Fatalf("%s: NO plan modifiers — churn regression (missing UseStateForUnknown)", name)
		}
		req := planmodifier.StringRequest{
			State:       nonNullState,
			StateValue:  priorState,
			PlanValue:   types.StringUnknown(),
			ConfigValue: types.StringNull(),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, m := range sa.PlanModifiers {
			m.PlanModifyString(ctx, req, resp)
		}
		if resp.PlanValue.IsUnknown() {
			t.Errorf("%s: still (known after apply) after plan modifiers — churn NOT fixed", name)
		}
		if !resp.PlanValue.Equal(priorState) {
			t.Errorf("%s: expected plan to reuse prior state %v, got %v", name, priorState, resp.PlanValue)
		}
	})

	// tags: Optional+Computed ListAttribute (not nested). mapAssetToState always writes a
	// KNOWN list (never null/unknown), so UseStateForUnknown reuses it on unrelated in-place
	// updates instead of "(known after apply)". Crash-safe because mapAssetToState reorders
	// the API labels to the prior tag order before writing state (labels are order-unstable
	// from the API), so the frozen plan order equals the applied order.
	t.Run("tags", func(t *testing.T) {
		const name = "tags"
		a := attrs[name]
		la, ok := a.(schema.ListAttribute)
		if !ok {
			t.Fatalf("%s: expected ListAttribute, got %T", name, a)
		}
		if len(la.PlanModifiers) == 0 {
			t.Fatalf("%s: NO plan modifiers — churn regression (missing UseStateForUnknown)", name)
		}
		priorState := types.ListValueMust(types.StringType, []attr.Value{
			types.StringValue("prod"), types.StringValue("beta"),
		})
		req := planmodifier.ListRequest{
			State:       nonNullState,
			StateValue:  priorState,
			PlanValue:   types.ListUnknown(types.StringType),
			ConfigValue: types.ListNull(types.StringType),
		}
		resp := &planmodifier.ListResponse{PlanValue: req.PlanValue}
		for _, m := range la.PlanModifiers {
			m.PlanModifyList(ctx, req, resp)
		}
		if resp.PlanValue.IsUnknown() {
			t.Errorf("%s: still (known after apply) after plan modifiers — churn NOT fixed", name)
		}
		if !resp.PlanValue.Equal(priorState) {
			t.Errorf("%s: expected plan to reuse prior state %v, got %v", name, priorState, resp.PlanValue)
		}
	})

	// classifier/api_version/main_file: Optional+Computed strings that COMBINE
	// UseStateForUnknown (the churn fix) with RequiresReplaceExceptOnImport (immutability).
	// Two invariants must both hold, proving the two modifiers compose without conflict:
	//   (1) churn scenario — config omitted so the plan value is unknown: USFU reuses prior
	//       state AND RequiresReplaceExceptOnImport no-ops (it early-returns on unknown
	//       plan), so the plan shows the prior value with NO spurious replacement.
	//   (2) real-change scenario — config sets a new KNOWN value differing from state: USFU
	//       no-ops (plan already known) and RequiresReplaceExceptOnImport STILL forces
	//       replacement. This proves adding USFU did not swallow the replace signal.
	// Safe against inconsistent-result because on an in-place update the API returns the
	// same immutable value (normalizeClassifier / extractFileMetadata / extractAttributeValue
	// preserve it) and Read seeds it with the identical helpers, so the USFU-frozen plan
	// equals what Update re-derives.
	replaceFields := []string{"classifier", "api_version", "main_file"}
	for _, name := range replaceFields {
		t.Run(name, func(t *testing.T) {
			a := attrs[name]
			sa, ok := a.(schema.StringAttribute)
			if !ok {
				t.Fatalf("%s: expected StringAttribute, got %T", name, a)
			}
			if len(sa.PlanModifiers) < 2 {
				t.Fatalf("%s: expected UseStateForUnknown + RequiresReplaceExceptOnImport "+
					"(>=2 plan modifiers), got %d", name, len(sa.PlanModifiers))
			}

			// (1) churn: plan unknown (config omitted), non-null prior state.
			priorState := types.StringValue("oas")
			churnReq := planmodifier.StringRequest{
				State:       nonNullState,
				Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
				StateValue:  priorState,
				PlanValue:   types.StringUnknown(),
				ConfigValue: types.StringNull(),
			}
			churnResp := &planmodifier.StringResponse{PlanValue: churnReq.PlanValue}
			for _, m := range sa.PlanModifiers {
				// Thread the previous modifier's output plan value in as this
				// modifier's input, exactly as terraform-plugin-framework chains
				// them. Without this, RequiresReplaceExceptOnImport would see the
				// original unknown plan and hit its unknown early-return instead of
				// the real Equal() comparison against prior state — i.e. the test
				// would validate the two modifiers independently, not composed.
				churnReq.PlanValue = churnResp.PlanValue
				m.PlanModifyString(ctx, churnReq, churnResp)
			}
			if churnResp.PlanValue.IsUnknown() {
				t.Errorf("%s: churn scenario still (known after apply) — UseStateForUnknown not applied", name)
			}
			if !churnResp.PlanValue.Equal(priorState) {
				t.Errorf("%s: churn scenario expected reuse of prior state %v, got %v", name, priorState, churnResp.PlanValue)
			}
			if churnResp.RequiresReplace {
				t.Errorf("%s: churn scenario must NOT force replacement (config merely omitted)", name)
			}

			// (2) real change: plan is a KNOWN value that differs from state.
			changeReq := planmodifier.StringRequest{
				State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")},
				Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
				StateValue:  types.StringValue("oas"),
				PlanValue:   types.StringValue("raml"),
				ConfigValue: types.StringValue("raml"),
			}
			changeResp := &planmodifier.StringResponse{PlanValue: changeReq.PlanValue}
			for _, m := range sa.PlanModifiers {
				changeReq.PlanValue = changeResp.PlanValue
				m.PlanModifyString(ctx, changeReq, changeResp)
			}
			if !changeResp.RequiresReplace {
				t.Errorf("%s: real-change scenario must force replacement (oas->raml); "+
					"UseStateForUnknown wrongly suppressed RequiresReplaceExceptOnImport", name)
			}
		})
	}

	// updated_date MUST NOT carry UseStateForUnknown. It is a server-bumped timestamp
	// that changes on EVERY write (including unrelated in-place edits like a tags-only
	// change), and Update re-reads the new timestamp from the API with no preservation.
	// UseStateForUnknown would freeze the plan to the prior timestamp, so apply would
	// fail with "Provider produced inconsistent result after apply" (planned old value
	// != applied new value). "(known after apply)" is the CORRECT display here. This
	// sub-test guards against anyone re-introducing that modifier. (created_date, by
	// contrast, is immutable, so it legitimately keeps UseStateForUnknown.)
	t.Run("updated_date_must_not_freeze", func(t *testing.T) {
		a := attrs["updated_date"]
		sa, ok := a.(schema.StringAttribute)
		if !ok {
			t.Fatalf("updated_date: expected StringAttribute, got %T", a)
		}
		if len(sa.PlanModifiers) != 0 {
			t.Errorf("updated_date: expected NO plan modifiers (server-bumped timestamp; "+
				"UseStateForUnknown would cause inconsistent-result at apply), got %d", len(sa.PlanModifiers))
		}
	})
}

// TestAssetResource_TypeField_NoReplaceOnImport is a targeted regression guard for a
// HIGH-SEVERITY bug: importing an existing asset (or any plan where `type` is omitted from
// config) used to force a DESTROY+RECREATE of the live asset.
//
// Root cause: `type` is Optional+Computed with a bare stringplanmodifier.RequiresReplace().
// When config omits `type`, the framework marks its plan value UNKNOWN. The built-in
// RequiresReplace compares plan-vs-state with Equal(); unknown != "rest-api", so it set
// RequiresReplace=true — catastrophic on import, where the asset already exists.
//
// Fix: prepend stringplanmodifier.UseStateForUnknown() so the omitted value reuses the
// prior state ("rest-api") BEFORE RequiresReplace runs; RequiresReplace then sees
// "rest-api" == "rest-api" and no-ops. ORDER MATTERS — USFU must precede RequiresReplace
// (this test threads the plan value between modifiers exactly as the framework does, so a
// reorder regression that reintroduces the replace is caught by the churn scenario).
//
// Unlike classifier/api_version/main_file (which use the custom RequiresReplaceExceptOnImport),
// `type` uses the BUILT-IN RequiresReplace, so it is NOT covered by the replaceFields loop in
// TestAssetResource_ChurnFields_UseStateForUnknown — it needs its own coverage.
//
// Safe against "inconsistent result after apply": `type` is immutable, so on an in-place
// update Read maps the API type back to the same state value (normalizeType), and on a
// genuine type change the plan value is KNOWN (not unknown) so USFU no-ops and RequiresReplace
// still forces replacement — proven by the real-change scenario below.
func TestAssetResource_TypeField_NoReplaceOnImport(t *testing.T) {
	res := NewAssetResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	a := schemaResp.Schema.Attributes["type"]
	sa, ok := a.(schema.StringAttribute)
	if !ok {
		t.Fatalf("type: expected StringAttribute, got %T", a)
	}
	if !sa.Optional || !sa.Computed {
		t.Fatalf("type: expected Optional+Computed (got Optional=%v Computed=%v)", sa.Optional, sa.Computed)
	}
	if len(sa.PlanModifiers) < 2 {
		t.Fatalf("type: expected UseStateForUnknown + RequiresReplace (>=2 plan modifiers), got %d — "+
			"the import-forces-replace fix is missing", len(sa.PlanModifiers))
	}

	nonNullState := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")}

	// (1) CHURN / IMPORT scenario — config omits `type` so the framework plans it UNKNOWN.
	// Prior state (seeded by import's Read) is "rest-api". Expect: plan reuses "rest-api"
	// AND no replacement. This is the exact scenario that used to destroy+recreate.
	t.Run("import_omitted_type_no_replace", func(t *testing.T) {
		priorState := types.StringValue("rest-api")
		req := planmodifier.StringRequest{
			State:       nonNullState,
			Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
			StateValue:  priorState,
			PlanValue:   types.StringUnknown(),
			ConfigValue: types.StringNull(),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, m := range sa.PlanModifiers {
			// Thread the previous modifier's output plan value in as this modifier's
			// input, exactly as terraform-plugin-framework chains them. This is what
			// makes the ordering (USFU before RequiresReplace) meaningful: if reordered,
			// RequiresReplace would see the still-unknown plan and set RequiresReplace.
			req.PlanValue = resp.PlanValue
			m.PlanModifyString(ctx, req, resp)
		}
		if resp.PlanValue.IsUnknown() {
			t.Errorf("type: import scenario still (known after apply) — UseStateForUnknown not applied")
		}
		if !resp.PlanValue.Equal(priorState) {
			t.Errorf("type: import scenario expected reuse of prior state %v, got %v", priorState, resp.PlanValue)
		}
		if resp.RequiresReplace {
			t.Errorf("type: import scenario MUST NOT force replacement (config merely omitted) — " +
				"this is the destroy-on-import regression")
		}
	})

	// (2) REAL-CHANGE scenario — config sets a KNOWN new type differing from state. Expect:
	// USFU no-ops (plan already known) and RequiresReplace STILL fires. Proves adding USFU
	// did not swallow the replace signal for a genuine immutable-field change.
	t.Run("real_type_change_still_replaces", func(t *testing.T) {
		req := planmodifier.StringRequest{
			State:       nonNullState,
			Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
			StateValue:  types.StringValue("rest-api"),
			PlanValue:   types.StringValue("http-api"),
			ConfigValue: types.StringValue("http-api"),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, m := range sa.PlanModifiers {
			req.PlanValue = resp.PlanValue
			m.PlanModifyString(ctx, req, resp)
		}
		if !resp.RequiresReplace {
			t.Errorf("type: real-change scenario must force replacement (rest-api->http-api); " +
				"UseStateForUnknown wrongly suppressed RequiresReplace")
		}
	})

	// (3) CREATE scenario — whole resource state is null. USFU must bail (leaving the plan
	// unknown so the value is computed fresh) and RequiresReplace must not fire on create.
	t.Run("create_leaves_computed", func(t *testing.T) {
		req := planmodifier.StringRequest{
			State:       tfsdk.State{Raw: tftypes.NewValue(tftypes.String, nil)}, // null == create
			Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
			StateValue:  types.StringNull(),
			PlanValue:   types.StringUnknown(),
			ConfigValue: types.StringNull(),
		}
		resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
		for _, m := range sa.PlanModifiers {
			req.PlanValue = resp.PlanValue
			m.PlanModifyString(ctx, req, resp)
		}
		if !resp.PlanValue.IsUnknown() {
			t.Errorf("type: create scenario should leave plan unknown (computed fresh), got %v", resp.PlanValue)
		}
		if resp.RequiresReplace {
			t.Errorf("type: create scenario must NOT force replacement")
		}
	})
}

// TestReorderByKey guards the invariant that makes UseStateForUnknown crash-safe on the
// order-unstable computed collections (pages, categories, custom_fields, tags): the
// readback must emit elements in the model's prior order so the applied value equals the
// UseStateForUnknown-frozen plan value. If the readback returned a different order,
// Terraform would fail with "Provider produced inconsistent result after apply".
//
// The over-arching contract is that the result is always a PERMUTATION of the API input
// (same multiset of elements — nothing added, dropped, or de-duplicated), reordered so
// matched keys lead in desired order and unmatched keys trail in API order.
func TestReorderByKey(t *testing.T) {
	id := func(s string) string { return s }

	// sorted copy for multiset comparison (order-independent)
	sortedCopy := func(in []string) []string {
		out := append([]string(nil), in...)
		for i := 1; i < len(out); i++ {
			for j := i; j > 0 && out[j-1] > out[j]; j-- {
				out[j-1], out[j] = out[j], out[j-1]
			}
		}
		return out
	}
	sameMultiset := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		sa, sb := sortedCopy(a), sortedCopy(b)
		for i := range sa {
			if sa[i] != sb[i] {
				return false
			}
		}
		return true
	}
	eq := func(a, b []string) bool {
		if len(a) != len(b) {
			return false
		}
		for i := range a {
			if a[i] != b[i] {
				return false
			}
		}
		return true
	}

	tests := []struct {
		name    string
		api     []string
		desired []string
		want    []string
	}{
		{
			name:    "API arbitrary order is reordered to desired (the core crash fix)",
			api:     []string{"c", "a", "b"}, // API returns arbitrary order
			desired: []string{"a", "b", "c"}, // frozen plan / prior state order
			want:    []string{"a", "b", "c"},
		},
		{
			name:    "unmatched API keys are appended in API order, not dropped",
			api:     []string{"z", "a", "y", "b"},
			desired: []string{"a", "b"},
			want:    []string{"a", "b", "z", "y"},
		},
		{
			name:    "empty desired returns API unchanged (Create/import: no prior order)",
			api:     []string{"b", "a"},
			desired: nil,
			want:    []string{"b", "a"},
		},
		{
			name:    "empty API returns empty (asset has none)",
			api:     nil,
			desired: []string{"a", "b"},
			want:    nil,
		},
		{
			name:    "desired key absent from API is skipped (deleted out-of-band)",
			api:     []string{"a"},
			desired: []string{"a", "gone"},
			want:    []string{"a"},
		},
		{
			name:    "duplicate keys preserve multiplicity (no de-dup -> no inconsistent result)",
			api:     []string{"x", "a", "x"},
			desired: []string{"a", "x"},
			want:    []string{"a", "x", "x"}, // one x consumed by desired, extra x appended
		},
		{
			name:    "duplicate in desired drains multiple API copies in API order",
			api:     []string{"x1a", "x1b"}, // two copies of key 'x' (distinguished for assert)
			desired: []string{"x", "x"},
			want:    []string{"x1a", "x1b"},
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			// For the "duplicate in desired" case, key is a prefix so both map to "x".
			keyOf := id
			if tc.name == "duplicate in desired drains multiple API copies in API order" {
				keyOf = func(s string) string { return "x" }
			}
			got := reorderByKey(tc.api, tc.desired, keyOf)
			if !eq(got, tc.want) {
				t.Errorf("reorderByKey(%v, %v) = %v, want %v", tc.api, tc.desired, got, tc.want)
			}
			// Invariant: output is always a permutation (multiset) of the API input.
			if !sameMultiset(got, tc.api) {
				t.Errorf("reorderByKey(%v, %v) = %v is NOT a permutation of the API input "+
					"(would cause inconsistent-result at apply)", tc.api, tc.desired, got)
			}
		})
	}
}

// TestMapAssetToState_ContactFields_Idempotent guards against a
// "Provider produced inconsistent result after apply" crash on contact_name,
// contact_email and manager. These Optional+Computed fields carry
// UseStateForUnknown, so on an omit-path update the plan is frozen to the prior
// value and the Update readback (mapAssetToState called a second time, over the
// frozen plan) MUST reproduce that exact value. The readback must therefore be a
// fixed point: f(f(x)) == f(x). The pre-fix tri-branch forced StringValue("")
// when the API returned no contact, which is NOT a fixed point — f("Alice")=""
// but f("")=null — so a second pass silently flipped ""->null and apply aborted.
// This maps twice and asserts stability, and separately asserts a prior "" is
// preserved as "" (not null). It FAILS against the pre-fix code.
func TestMapAssetToState_ContactFields_Idempotent(t *testing.T) {
	r := &AssetResource{}

	// API returns NO contact info (all pointers nil) — the omitted / cleared case.
	emptyAsset := func() *exchange.Asset {
		return &exchange.Asset{GroupID: "g", AssetID: "a", Version: "1.0.0"}
	}

	t.Run("prior_value_preserved_and_stable_across_two_passes", func(t *testing.T) {
		state := &AssetResourceModel{
			ContactName:  types.StringValue("Alice"),
			ContactEmail: types.StringValue("alice@example.com"),
			Manager:      types.StringValue("Bob"),
		}
		r.mapAssetToState(state, emptyAsset())
		first := *state
		// Feed the (now USFU-frozen) result back through the readback and require
		// no change — this is exactly what Update does after refresh.
		r.mapAssetToState(state, emptyAsset())

		if !state.ContactName.Equal(first.ContactName) {
			t.Errorf("contact_name not idempotent: pass1=%v pass2=%v (=> inconsistent-result crash)", first.ContactName, state.ContactName)
		}
		if !state.ContactEmail.Equal(first.ContactEmail) {
			t.Errorf("contact_email not idempotent: pass1=%v pass2=%v", first.ContactEmail, state.ContactEmail)
		}
		if !state.Manager.Equal(first.Manager) {
			t.Errorf("manager not idempotent: pass1=%v pass2=%v", first.Manager, state.Manager)
		}
		// And the preserved value must be the user's, not a forced "".
		if state.ContactName.ValueString() != "Alice" {
			t.Errorf("contact_name: expected prior \"Alice\" preserved, got %v", state.ContactName)
		}
	})

	t.Run("empty_string_prior_is_a_fixed_point", func(t *testing.T) {
		// The exact frozen value that triggered the crash: a prior "".
		state := &AssetResourceModel{
			ContactName:  types.StringValue(""),
			ContactEmail: types.StringValue(""),
			Manager:      types.StringValue(""),
		}
		r.mapAssetToState(state, emptyAsset())
		if state.ContactName.IsNull() {
			t.Error("contact_name: prior \"\" flipped to null — NOT a fixed point (frozen \"\" != applied null => apply crash)")
		}
		if state.ContactEmail.IsNull() {
			t.Error("contact_email: prior \"\" flipped to null")
		}
		if state.Manager.IsNull() {
			t.Error("manager: prior \"\" flipped to null")
		}
	})

	t.Run("null_prior_stays_null", func(t *testing.T) {
		state := &AssetResourceModel{} // all fields null
		r.mapAssetToState(state, emptyAsset())
		if !state.ContactName.IsNull() {
			t.Errorf("contact_name: null prior + empty API should stay null, got %v", state.ContactName)
		}
	})

	t.Run("api_value_wins_over_prior", func(t *testing.T) {
		cn, ce, mgr := "Carol", "carol@example.com", "Dave"
		asset := emptyAsset()
		asset.ContactName, asset.ContactEmail, asset.Manager = &cn, &ce, &mgr
		state := &AssetResourceModel{ContactName: types.StringValue("Alice")}
		r.mapAssetToState(state, asset)
		if state.ContactName.ValueString() != "Carol" {
			t.Errorf("contact_name: API value should win, got %v", state.ContactName)
		}
	})
}

// assetRawValue builds a fully-typed tftypes object value for the asset schema with
// every attribute defaulted to null, then applies the given per-attribute overrides.
// Deriving the attribute types from the schema (rather than restating ~25 tftypes
// entries per case, as TestAssetResource_Read does) keeps the ModifyPlan cases focused
// on the handful of attributes the guard actually reads, and immune to schema drift.
func assetRawValue(ctx context.Context, s schema.Schema, overrides map[string]tftypes.Value) tftypes.Value {
	objType := s.Type().TerraformType(ctx).(tftypes.Object)
	attrs := make(map[string]tftypes.Value, len(objType.AttributeTypes))
	for name, typ := range objType.AttributeTypes {
		if v, ok := overrides[name]; ok {
			attrs[name] = v
		} else {
			attrs[name] = tftypes.NewValue(typ, nil) // null of the correct type
		}
	}
	return tftypes.NewValue(objType, attrs)
}

// TestAssetTypeRequiresFile pins the file-required allowlist (#104). Exchange infers an
// asset's type from its uploaded spec, so these types reject a fileless publish; each was
// confirmed live on 2026-07-16 (rest-api/soap-api/graphql-api long-proven; evented-api and
// ruleset → COULD_NOT_DETERMINE_ASSET_TYPE, grpc-api → MISSING_FILES_ERROR). The excluded
// types publish fine with no file — blocking them would be a false positive — with llm the
// live-verified fileless case. The lookup must normalize case and surrounding whitespace.
func TestAssetTypeRequiresFile(t *testing.T) {
	requires := []string{"rest-api", "soap-api", "graphql-api", "evented-api", "grpc-api", "ruleset"}
	// Fileless/unknown types the allowlist must NEVER block (a false positive would break a
	// legitimate fileless workflow). "" and an unknown string stand in for future types.
	fileless := []string{"custom", "app", "template", "example", "connector", "policy", "agent", "llm", "mcp", "http-api", "", "some-future-type"}

	for _, ty := range requires {
		if !assetTypeRequiresFile(ty) {
			t.Errorf("assetTypeRequiresFile(%q) = false, want true (file-backed type)", ty)
		}
	}
	for _, ty := range fileless {
		if assetTypeRequiresFile(ty) {
			t.Errorf("assetTypeRequiresFile(%q) = true, want false (fileless/unknown type must never be blocked)", ty)
		}
	}

	// Normalization: uppercase and surrounding whitespace must not defeat the lookup for a
	// file-backed type, nor accidentally promote a fileless type into the allowlist.
	if !assetTypeRequiresFile("  Evented-API  ") {
		t.Error("assetTypeRequiresFile should normalize case+whitespace for a file-backed type")
	}
	if assetTypeRequiresFile("  CUSTOM  ") {
		t.Error("assetTypeRequiresFile should normalize case+whitespace for a fileless type (must stay false)")
	}
}

// TestAssetResource_ModifyPlan_FilePathGuard covers the #78 plan-time guard that turns a
// silent data-loss footgun into a loud error: publishing (CREATE) or REPLACING a
// file-backed asset type (rest-api/soap-api/graphql-api) with no file_path set. The
// guard must be carefully scoped so it NEVER fires on the cases that legitimately have a
// null file_path — a fileless type, an unresolved (unknown) type, and above all the
// post-import in-place update (where file_path is a local-only field that stays null).
// The replace case is the dangerous one the guard exists for: version forces replacement,
// so a failed upload can strand the user after the previous version was already destroyed.
func TestAssetResource_ModifyPlan_FilePathGuard(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	raw := func(overrides map[string]tftypes.Value) tftypes.Value {
		return assetRawValue(ctx, schemaResp.Schema, overrides)
	}
	nullObj := tftypes.NewValue(objType, nil)

	// identity returns the replace-forcing attributes (all KNOWN) for a given version.
	// A replace is detected when a known identity attr differs between state and plan.
	identity := func(version string) map[string]tftypes.Value {
		return map[string]tftypes.Value{
			"organization_id": tftypes.NewValue(tftypes.String, "org"),
			"group_id":        tftypes.NewValue(tftypes.String, "grp"),
			"asset_id":        tftypes.NewValue(tftypes.String, "asset"),
			"version":         tftypes.NewValue(tftypes.String, version),
			"type":            tftypes.NewValue(tftypes.String, "rest-api"),
		}
	}
	withFilePath := func(m map[string]tftypes.Value, fp string) map[string]tftypes.Value {
		m["file_path"] = tftypes.NewValue(tftypes.String, fp)
		return m
	}

	// identityType builds the replace-forcing identity for a given version but with an
	// arbitrary file-backed type, so the replace-guard cases can cover the #104 additions
	// (evented-api/grpc-api/ruleset), not just rest-api.
	identityType := func(version, assetType string) map[string]tftypes.Value {
		m := identity(version)
		m["type"] = tftypes.NewValue(tftypes.String, assetType)
		return m
	}

	tests := []struct {
		name       string
		planRaw    tftypes.Value
		stateRaw   tftypes.Value
		wantErr    bool
		wantAction string // substring expected in the diagnostic detail when wantErr
		wantType   string // type name expected in the diagnostic detail; defaults to "rest-api"
	}{
		{
			name:       "create file-backed type without file_path is blocked",
			planRaw:    raw(map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, "rest-api")}),
			stateRaw:   nullObj, // null state == create
			wantErr:    true,
			wantAction: "created",
		},
		{
			name:     "create fileless type without file_path is allowed",
			planRaw:  raw(map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, "custom")}),
			stateRaw: nullObj,
			wantErr:  false,
		},
		{
			name:     "create file-backed type WITH file_path is allowed",
			planRaw:  raw(withFilePath(map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, "rest-api")}, "spec.json")),
			stateRaw: nullObj,
			wantErr:  false,
		},
		{
			name:     "create with unknown type is skipped (cannot classify)",
			planRaw:  raw(map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, tftypes.UnknownValue)}),
			stateRaw: nullObj,
			wantErr:  false,
		},
		{
			name:       "replace (version change) of file-backed type without file_path is blocked",
			planRaw:    raw(identity("2.0.0")),
			stateRaw:   raw(identity("1.0.0")),
			wantErr:    true,
			wantAction: "replaced (destroyed and recreated)",
		},
		{
			name:     "in-place update of file-backed type without file_path is allowed (post-import steady state)",
			planRaw:  raw(identity("1.0.0")),
			stateRaw: raw(identity("1.0.0")),
			wantErr:  false,
		},
		{
			name:       "create evented-api without file_path is blocked (#104)",
			planRaw:    raw(map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, "evented-api")}),
			stateRaw:   nullObj,
			wantErr:    true,
			wantAction: "created",
			wantType:   "evented-api",
		},
		{
			name:       "create grpc-api without file_path is blocked (#104)",
			planRaw:    raw(map[string]tftypes.Value{"type": tftypes.NewValue(tftypes.String, "grpc-api")}),
			stateRaw:   nullObj,
			wantErr:    true,
			wantAction: "created",
			wantType:   "grpc-api",
		},
		{
			name:       "replace (version change) of ruleset without file_path is blocked (#104)",
			planRaw:    raw(identityType("2.0.0", "ruleset")),
			stateRaw:   raw(identityType("1.0.0", "ruleset")),
			wantErr:    true,
			wantAction: "replaced (destroyed and recreated)",
			wantType:   "ruleset",
		},
		{
			name:     "destroy plan (null plan) is skipped without panic",
			planRaw:  nullObj,
			stateRaw: raw(identity("1.0.0")),
			wantErr:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resource.ModifyPlanRequest{
				Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: tt.planRaw},
				State:  tfsdk.State{Schema: schemaResp.Schema, Raw: tt.stateRaw},
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tt.planRaw},
			}
			resp := &resource.ModifyPlanResponse{}
			res.ModifyPlan(ctx, req, resp)

			if !tt.wantErr {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error, got: %v", resp.Diagnostics.Errors())
				}
				return
			}

			if !resp.Diagnostics.HasError() {
				t.Fatalf("expected a plan-time error, got none")
			}
			var detail string
			found := false
			for _, d := range resp.Diagnostics {
				if strings.Contains(d.Summary(), "Missing file_path for a file-backed asset type") {
					detail, found = d.Detail(), true
				}
			}
			if !found {
				t.Fatalf("expected the file_path guard diagnostic, got: %v", resp.Diagnostics.Errors())
			}
			if !strings.Contains(detail, tt.wantAction) {
				t.Errorf("diagnostic detail = %q, want containing action %q", detail, tt.wantAction)
			}
			wantType := tt.wantType
			if wantType == "" {
				wantType = "rest-api"
			}
			if !strings.Contains(detail, wantType) {
				t.Errorf("diagnostic detail should name the offending type %q; got %q", wantType, detail)
			}
		})
	}
}

// TestAssetResource_SyncPages_AdoptsAutoProvisionedHomePage covers the #77 fix. Exchange
// auto-provisions a "home" page on every asset version, so a fresh Create (which passes
// NO current pages) hits a 409 on CreateDraftPage for "home". Instead of failing the whole
// apply, syncPages must adopt the existing page: resolve its real draft path via
// ListDraftPages, upsert the desired content, and publish. This mirrors the external-
// instances idempotent-create reconcile (client.IsConflict).
func TestAssetResource_SyncPages_AdoptsAutoProvisionedHomePage(t *testing.T) {
	base := "/exchange/api/v2/assets/g/a/1.0.0"
	pagesPath := base + "/portal/draft/pages" // POST create + GET list share this exact path
	homeContentPath := pagesPath + "/home"    // PUT content
	publishPath := base + "/portal/draft"     // PUT publish

	tests := []struct {
		name           string
		desiredContent string
		listBody       []map[string]interface{}
		wantErr        bool
		errContains    string
		wantContentPut bool
		wantPublish    bool
	}{
		{
			name:           "home 409 is adopted and content upserted",
			desiredContent: "# Welcome home",
			listBody:       []map[string]interface{}{{"name": "home", "path": "home"}},
			wantContentPut: true,
			wantPublish:    true,
		},
		{
			name:           "home 409 with empty content skips the PUT but still publishes",
			desiredContent: "",
			listBody:       []map[string]interface{}{{"name": "home", "path": "home"}},
			wantContentPut: false,
			wantPublish:    true,
		},
		{
			name:           "409 but page missing from the draft listing is a hard error",
			desiredContent: "# Welcome",
			listBody:       []map[string]interface{}{}, // lookup resolves to "" -> cannot adopt
			wantErr:        true,
			errContains:    "was not found in the draft page listing",
			wantContentPut: false,
			wantPublish:    false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var creates, lists, contentPuts, publishes int
			var gotContent, gotContentType string

			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				pagesPath: func(w http.ResponseWriter, r *http.Request) {
					switch r.Method {
					case http.MethodPost:
						creates++
						// Exchange auto-provisioned "home" already exists -> 409.
						testutil.ErrorResponse(w, http.StatusConflict, "page already exists")
					case http.MethodGet:
						lists++
						testutil.JSONResponse(w, http.StatusOK, tt.listBody)
					default:
						w.WriteHeader(http.StatusOK)
					}
				},
				homeContentPath: func(w http.ResponseWriter, r *http.Request) {
					contentPuts++
					gotContentType = r.Header.Get("Content-Type")
					b, _ := io.ReadAll(r.Body)
					gotContent = string(b)
					w.WriteHeader(http.StatusNoContent)
				},
				publishPath: func(w http.ResponseWriter, r *http.Request) {
					publishes++
					w.WriteHeader(http.StatusNoContent)
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			res := NewAssetResource().(*AssetResource)
			res.client = &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

			// Fresh Create passes NO current pages, mirroring the real path where Exchange
			// has already auto-provisioned "home" server-side.
			desired := []PageModel{
				{PageName: types.StringValue("home"), Content: types.StringValue(tt.desiredContent), PagePath: types.StringNull()},
			}

			err := res.syncPages(context.Background(), "g", "a", "1.0.0", nil, desired)

			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got nil")
				}
				if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("error = %v, want containing %q", err, tt.errContains)
				}
			} else if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}

			// The Create is always attempted exactly once and always 409s; the adoption
			// path then does exactly one draft listing to resolve the real path.
			if creates != 1 {
				t.Errorf("expected exactly 1 CreateDraftPage attempt, got %d", creates)
			}
			if lists != 1 {
				t.Errorf("expected exactly 1 ListDraftPages lookup, got %d", lists)
			}
			if tt.wantContentPut {
				if contentPuts != 1 {
					t.Errorf("expected exactly 1 content PUT, got %d", contentPuts)
				}
				if gotContent != tt.desiredContent {
					t.Errorf("content PUT body = %q, want %q", gotContent, tt.desiredContent)
				}
				if gotContentType != "text/markdown" {
					t.Errorf("content PUT Content-Type = %q, want text/markdown", gotContentType)
				}
			} else if contentPuts != 0 {
				t.Errorf("expected 0 content PUTs, got %d", contentPuts)
			}
			if tt.wantPublish {
				if publishes != 1 {
					t.Errorf("expected exactly 1 PublishDraft, got %d", publishes)
				}
			} else if publishes != 0 {
				t.Errorf("expected 0 PublishDraft, got %d", publishes)
			}
		})
	}
}

// TestAssetResource_LookupDraftPagePath exercises the draft-path resolution used by the
// #77 adoption flow. CreateDraftPage POSTs into the DRAFT namespace and Exchange assigns
// the real path (which may carry a random prefix), so the page must be resolved from the
// draft listing by name, exact path, OR trailing path segment — and absence returns ""
// (no error) so the caller can raise a precise diagnostic.
func TestAssetResource_LookupDraftPagePath(t *testing.T) {
	pagesPath := "/exchange/api/v2/assets/g/a/1.0.0/portal/draft/pages"

	tests := []struct {
		name     string
		pageName string
		listBody []map[string]interface{}
		want     string
	}{
		{
			name:     "matches by exact name",
			pageName: "home",
			listBody: []map[string]interface{}{{"name": "home", "path": "home"}},
			want:     "home",
		},
		{
			name:     "matches by exact path when name differs",
			pageName: "getting-started",
			listBody: []map[string]interface{}{{"name": "Getting Started", "path": "getting-started"}},
			want:     "getting-started",
		},
		{
			name:     "falls back to trailing path segment when path is prefixed",
			pageName: "home",
			listBody: []map[string]interface{}{{"name": "Home Page", "path": "abc123/home"}},
			want:     "abc123/home",
		},
		{
			name:     "returns empty (no error) when absent",
			pageName: "home",
			listBody: []map[string]interface{}{{"name": "other", "path": "other"}},
			want:     "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				pagesPath: func(w http.ResponseWriter, r *http.Request) {
					testutil.AssertHTTPRequest(t, r, http.MethodGet, pagesPath)
					testutil.JSONResponse(w, http.StatusOK, tt.listBody)
				},
			}
			server := testutil.MockHTTPServer(t, handlers)

			res := NewAssetResource().(*AssetResource)
			res.client = &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

			got, err := res.lookupDraftPagePath(context.Background(), "g", "a", "1.0.0", tt.pageName)
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			if got != tt.want {
				t.Errorf("lookupDraftPagePath = %q, want %q", got, tt.want)
			}
		})
	}
}

// additionalFileObjectType is the element type of the additional_file list, used
// to build test list values. It mirrors the schema's NestedObject attributes.
func additionalFileObjectType() types.ObjectType {
	return types.ObjectType{AttrTypes: map[string]attr.Type{
		"path":       types.StringType,
		"classifier": types.StringType,
	}}
}

// makeAdditionalFileList builds a known additional_file list value from (path,
// classifier) pairs, for exercising additionalFilesToUploads and the plan modifier.
func makeAdditionalFileList(t *testing.T, pairs ...[2]string) types.List {
	t.Helper()
	elems := make([]attr.Value, 0, len(pairs))
	for _, p := range pairs {
		obj, diags := types.ObjectValue(additionalFileObjectType().AttrTypes, map[string]attr.Value{
			"path":       types.StringValue(p[0]),
			"classifier": types.StringValue(p[1]),
		})
		if diags.HasError() {
			t.Fatalf("failed to build additional_file object: %v", diags)
		}
		elems = append(elems, obj)
	}
	return types.ListValueMust(additionalFileObjectType(), elems)
}

// TestAdditionalFilesToUploads covers the plan-list → client-upload mapping for
// task #103, including the null/unknown/empty short-circuits and the canonical
// two-file policy shape.
func TestAdditionalFilesToUploads(t *testing.T) {
	ctx := context.Background()

	t.Run("null list returns nil", func(t *testing.T) {
		got, diags := additionalFilesToUploads(ctx, types.ListNull(additionalFileObjectType()))
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got != nil {
			t.Errorf("expected nil uploads for null list, got %+v", got)
		}
	})

	t.Run("unknown list returns nil", func(t *testing.T) {
		got, diags := additionalFilesToUploads(ctx, types.ListUnknown(additionalFileObjectType()))
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got != nil {
			t.Errorf("expected nil uploads for unknown list, got %+v", got)
		}
	})

	t.Run("empty list returns nil", func(t *testing.T) {
		empty := types.ListValueMust(additionalFileObjectType(), []attr.Value{})
		got, diags := additionalFilesToUploads(ctx, empty)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		if got != nil {
			t.Errorf("expected nil uploads for empty list, got %+v", got)
		}
	})

	t.Run("policy two-file shape maps in order", func(t *testing.T) {
		list := makeAdditionalFileList(t,
			[2]string{"specs/metadata.yaml", "metadata"},
			[2]string{"specs/extra.json", "extra"},
		)
		got, diags := additionalFilesToUploads(ctx, list)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags)
		}
		want := []exchange.AssetFileUpload{
			{FilePath: "specs/metadata.yaml", Classifier: "metadata"},
			{FilePath: "specs/extra.json", Classifier: "extra"},
		}
		if len(got) != len(want) {
			t.Fatalf("got %d uploads, want %d: %+v", len(got), len(want), got)
		}
		for i := range want {
			if got[i] != want[i] {
				t.Errorf("upload[%d] = %+v, want %+v", i, got[i], want[i])
			}
		}
	})
}

// TestAssetResource_AdditionalFile_Schema asserts the additional_file block is
// wired the way the multi-file design requires: Optional, ListNested, with the
// import-aware RequiresReplace modifier and two Required string children.
func TestAssetResource_AdditionalFile_Schema(t *testing.T) {
	res := NewAssetResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	a, ok := schemaResp.Schema.Attributes["additional_file"]
	if !ok {
		t.Fatal("additional_file attribute missing from schema")
	}
	lna, ok := a.(schema.ListNestedAttribute)
	if !ok {
		t.Fatalf("additional_file: expected ListNestedAttribute, got %T", a)
	}
	if !lna.Optional {
		t.Error("additional_file should be Optional")
	}
	if lna.Required {
		t.Error("additional_file must not be Required (single-file/fileless types must still work)")
	}
	if len(lna.PlanModifiers) == 0 {
		t.Fatal("additional_file: expected a RequiresReplace plan modifier, found none")
	}
	for _, child := range []string{"path", "classifier"} {
		ca, ok := lna.NestedObject.Attributes[child]
		if !ok {
			t.Fatalf("additional_file.%s missing", child)
		}
		sa, ok := ca.(schema.StringAttribute)
		if !ok {
			t.Fatalf("additional_file.%s: expected StringAttribute, got %T", child, ca)
		}
		if !sa.Required {
			t.Errorf("additional_file.%s should be Required", child)
		}
	}
}

// TestAssetResource_AdditionalFile_RequiresReplace exercises the List-typed plan
// modifier's four branches: create (null state), post-import settle (null→value),
// no-op (value→same), and mutation (value→different ⇒ replace).
func TestAssetResource_AdditionalFile_RequiresReplace(t *testing.T) {
	ctx := context.Background()
	mod := RequiresReplaceListExceptOnImport()
	elemType := additionalFileObjectType()

	nullRaw := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, nil)}
	setRaw := tfsdk.State{Raw: tftypes.NewValue(tftypes.String, "exists")}
	setPlan := tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")}

	oneFile := makeAdditionalFileList(t, [2]string{"specs/metadata.yaml", "metadata"})
	otherFile := makeAdditionalFileList(t, [2]string{"specs/other.yaml", "metadata"})

	tests := []struct {
		name        string
		state       tfsdk.State
		plan        tfsdk.Plan
		stateValue  types.List
		planValue   types.List
		wantReplace bool
	}{
		{
			name:        "create (null prior state) does not replace",
			state:       nullRaw,
			plan:        setPlan,
			stateValue:  types.ListNull(elemType),
			planValue:   oneFile,
			wantReplace: false,
		},
		{
			name:        "post-import settle null->value does not replace",
			state:       setRaw,
			plan:        setPlan,
			stateValue:  types.ListNull(elemType),
			planValue:   oneFile,
			wantReplace: false,
		},
		{
			name:        "unchanged value->same does not replace",
			state:       setRaw,
			plan:        setPlan,
			stateValue:  oneFile,
			planValue:   oneFile,
			wantReplace: false,
		},
		{
			name:        "mutated value->different forces replace",
			state:       setRaw,
			plan:        setPlan,
			stateValue:  oneFile,
			planValue:   otherFile,
			wantReplace: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := planmodifier.ListRequest{
				State:      tt.state,
				Plan:       tt.plan,
				StateValue: tt.stateValue,
				PlanValue:  tt.planValue,
			}
			resp := &planmodifier.ListResponse{PlanValue: tt.planValue}
			mod.PlanModifyList(ctx, req, resp)
			if resp.RequiresReplace != tt.wantReplace {
				t.Errorf("RequiresReplace = %v, want %v", resp.RequiresReplace, tt.wantReplace)
			}
		})
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
