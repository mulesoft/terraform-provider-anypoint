package exchange

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"os"
	"strings"
	"sync"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/diag"
	"github.com/hashicorp/terraform-plugin-framework/path"
	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema/planmodifier"
	"github.com/hashicorp/terraform-plugin-framework/schema/validator"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/exchange"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// errString is a minimal string-backed error for table tests.
type errString string

func (e errString) Error() string { return string(e) }

func TestIsAssetVersionConflict(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"pre-conditions code", errString("failed to create asset with status 409: {\"code\":\"ASSET_PRE_CONDITIONS_FAILED\"}"), true},
		{"409 already exists", errString("failed to create asset with status 409: an asset already exists with this version"), true},
		// Exact body captured from a live devx republish (2026-08-10) — regression fixture.
		{"real devx 409 body", errString(`failed to create asset with status 409: {"status":409,"code":"ASSET_PRE_CONDITIONS_FAILED","message":"Cannot create a new asset with the provided groupId, assetId, version, state","details":{"errors":["An asset already exists with this version and published lifecycle state."]}}`), true},
		{"other 409", errString("failed to create asset with status 409: something else"), false},
		{"400 missing prop", errString("failed to create asset with status 400: MISSING_REQUIRED_PROPERTIES"), false},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			if got := isAssetVersionConflict(tc.err); got != tc.want {
				t.Errorf("isAssetVersionConflict(%v) = %v, want %v", tc.err, got, tc.want)
			}
		})
	}
}

func TestNextPatchVersionHint(t *testing.T) {
	cases := map[string]string{
		"1.0.0":          "1.0.1",
		"2.3.9":          "2.3.10",
		"1":              "2",
		"1.0.0-SNAPSHOT": "1.0.1",
		"1.0.x":          "1.0.x-2",
	}
	for in, want := range cases {
		if got := nextPatchVersionHint(in); got != want {
			t.Errorf("nextPatchVersionHint(%q) = %q, want %q", in, got, want)
		}
	}
}

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
	optionalAttrs := []string{"description", "type", "status", "file_path", "classifier", "keywords", "api_version", "main_file", "contact_name", "contact_email"}
	// manager is Computed-only (#65): the Exchange metadata PATCH endpoint rejects any
	// attempt to SET it (HTTP 403/400 — LIVE-VERIFIED), so it is surfaced read-only rather
	// than being an Optional writable field.
	computedAttrs := []string{"id", "is_public", "is_snapshot", "minor_version", "version_group", "created_date", "updated_date", "manager"}

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

// TestDeclaredClassifierPresent locks the W-23914161 hardening: when a multi-file
// API-spec asset comes back with several classifiers (a RAML rest-api that Exchange
// also transcodes to oas/fat-oas/...), the readback must recognise that the user's
// declared classifier really IS one of the returned files and preserve it verbatim,
// so unstable file ordering across control planes cannot drift "raml" -> "oas".
func TestDeclaredClassifierPresent(t *testing.T) {
	// Mirrors the live prod file set for a RAML rest-api publish.
	ramlRestAPI := []exchange.AssetFile{
		{Classifier: "fat-oas"},
		{Classifier: "fat-raml"},
		{Classifier: "oas"},
		{Classifier: "original-raml"},
		{Classifier: "raml"},
		{Classifier: "rest-api-metadata"},
	}
	tests := []struct {
		name     string
		files    []exchange.AssetFile
		declared string
		want     bool
	}{
		{"raml present as own classifier", ramlRestAPI, "raml", true},
		{"oas present (transcoded sibling)", ramlRestAPI, "oas", true},
		{"fat-only match via bundled form", []exchange.AssetFile{{Classifier: "fat-raml"}}, "raml", true},
		{"declared not among files", ramlRestAPI, "wsdl", false},
		{"empty declared never matches", ramlRestAPI, "", false},
		{"no files", nil, "raml", false},
	}
	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			if got := declaredClassifierPresent(tc.files, tc.declared); got != tc.want {
				t.Errorf("declaredClassifierPresent(_, %q) = %v; want %v", tc.declared, got, tc.want)
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

// TestNormalizeType_ExtensionFamily pins the fix for STGX EXC-01i (2026-08-13):
// Exchange stores the mule-plugin family (policy, connector) under the single
// super-type "extension". When the user declares the semantic type, Create's
// readback receives "extension" from the API; normalizeType MUST return the user's
// declared value so the applied state equals the plan (otherwise Terraform fails
// with `inconsistent result after apply: .type: was "policy", but now "extension"`).
// It must also: preserve the identity case if a member is NOT normalized on some
// tenant; preserve the existing graphql-api<->graphql mapping; and, on a bare
// import (no prior state), surface the stored "extension" because the semantic
// sub-type is unrecoverable from the API.
func TestNormalizeType_ExtensionFamily(t *testing.T) {
	tests := []struct {
		name      string
		apiType   string
		stateType types.String
		want      string
	}{
		{"policy declared, api normalized to extension -> preserve policy", "extension", types.StringValue("policy"), "policy"},
		{"connector declared, api normalized to extension -> preserve connector", "extension", types.StringValue("connector"), "connector"},
		{"extension declared, api extension -> extension", "extension", types.StringValue("extension"), "extension"},
		{"policy declared, api NOT normalized (returns policy) -> policy (identity)", "policy", types.StringValue("policy"), "policy"},
		{"import (null state), api extension -> extension (sub-type unrecoverable)", "extension", types.StringNull(), "extension"},
		{"import (null state), api graphql -> graphql-api (reverse map)", "graphql", types.StringNull(), "graphql-api"},
		{"graphql-api declared, api returns graphql -> preserve graphql-api", "graphql", types.StringValue("graphql-api"), "graphql-api"},
		{"plain rest-api identity", "rest-api", types.StringValue("rest-api"), "rest-api"},
		{"unknown state falls through to api type", "extension", types.StringUnknown(), "extension"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := normalizeType(tt.apiType, tt.stateType); got != tt.want {
				t.Errorf("normalizeType(%q, %v) = %q, want %q", tt.apiType, tt.stateType, got, tt.want)
			}
		})
	}
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
// `type` uses the custom RequiresReplaceOnTypeChange, so it is NOT covered by the replaceFields
// loop in TestAssetResource_ChurnFields_UseStateForUnknown — it needs its own coverage.
//
// Safe against "inconsistent result after apply": `type` is immutable, so on an in-place
// update Read maps the API type back to the same state value (normalizeType), and on a
// genuine type change the plan value is KNOWN (not unknown) so USFU no-ops and the replace
// modifier still forces replacement — proven by the real-change scenario below.
//
// RequiresReplaceOnTypeChange additionally compares the API-NORMALIZED type, so the
// mule-plugin aliases (policy/connector/extension, all stored as "extension") are treated as
// the SAME type and do NOT force replacement — proven by the extension-family scenario below.
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

	// (4) EXTENSION-FAMILY scenario — the import limitation fix (STGX EXC-01i, 2026-08-13).
	// A bare import can only surface the stored super-type "extension" for a mule-plugin
	// asset. When the user then declares the semantic sub-type (policy/connector), the plan
	// value is KNOWN and differs textually from state "extension" — the BUILT-IN
	// RequiresReplace would destroy+recreate a live asset. RequiresReplaceOnTypeChange
	// compares the API-normalized form (all three -> "extension"), so it must NOT replace.
	// A genuine cross-type change (extension -> rest-api) must STILL replace.
	extensionFamily := []struct {
		name        string
		stateType   string
		planType    string
		wantReplace bool
	}{
		{"import extension -> declare policy: reconcile in place", "extension", "policy", false},
		{"import extension -> declare connector: reconcile in place", "extension", "connector", false},
		{"import extension -> declare extension: identity, no replace", "extension", "extension", false},
		{"policy -> extension: same super-type, no replace", "policy", "extension", false},
		{"policy -> connector: same super-type, no replace", "policy", "connector", false},
		{"extension -> rest-api: real cross-type change, replace", "extension", "rest-api", true},
		{"policy -> rest-api: real cross-type change, replace", "policy", "rest-api", true},
	}
	for _, tc := range extensionFamily {
		t.Run(tc.name, func(t *testing.T) {
			req := planmodifier.StringRequest{
				State:       nonNullState,
				Plan:        tfsdk.Plan{Raw: tftypes.NewValue(tftypes.String, "exists")},
				StateValue:  types.StringValue(tc.stateType),
				PlanValue:   types.StringValue(tc.planType),
				ConfigValue: types.StringValue(tc.planType),
			}
			resp := &planmodifier.StringResponse{PlanValue: req.PlanValue}
			for _, m := range sa.PlanModifiers {
				req.PlanValue = resp.PlanValue
				m.PlanModifyString(ctx, req, resp)
			}
			if resp.RequiresReplace != tc.wantReplace {
				t.Errorf("type: %s -> %s: RequiresReplace = %v, want %v",
					tc.stateType, tc.planType, resp.RequiresReplace, tc.wantReplace)
			}
		})
	}
}

// TestAssetResource_TypeNormalization_Hermetic is the end-to-end regression guard for
// STGX EXC-01i (2026-08-13): declaring a mule-plugin sub-type (policy/connector) used to
// fail Terraform's post-apply consistency check because Exchange stores the whole family
// as the super-type "extension" and the create readback rewrote state to "extension"
// (`inconsistent result after apply: .type: was "policy", but now "extension"`).
//
// It drives the resource's REAL Create and Read methods against an in-process, stateful
// fake Exchange whose GET readback ALWAYS returns type="extension" — i.e. it reproduces
// the exact normalization the live platform performs. The assertion IS the consistency
// invariant Terraform core enforces: for a KNOWN planned value, the applied state value
// must equal it. So asserting created state .type == the declared type (policy/connector/
// extension) proves the apply is consistent and would not error under real Terraform.
//
// It also pins the KNOWN LIMITATION as an explicit expectation: a bare import (state
// .type == null, as ImportState leaves it) resolves to "extension", because the semantic
// sub-type is unrecoverable from the API. The companion fix (RequiresReplaceOnTypeChange,
// covered by TestAssetResource_TypeField_NoReplaceOnImport) makes that recoverable in
// place — set type=policy in config afterward and it reconciles WITHOUT a destroy+recreate.
func TestAssetResource_TypeNormalization_Hermetic(t *testing.T) {
	const (
		orgID   = "test-org-id"
		groupID = "g"
		assetID = "a"
		version = "1.0.0"
	)

	// The fake stores the mule-plugin family the way Exchange does: whatever sub-type the
	// provider publishes, the GET readback reports the normalized super-type "extension".
	// This is precisely what makes a naive provider drift policy -> extension on apply.
	assetJSON := func(name string) map[string]interface{} {
		return map[string]interface{}{
			"groupId":       groupID,
			"assetId":       assetID,
			"version":       version,
			"name":          name,
			"description":   "",
			"type":          "extension", // <-- Exchange's normalization, always.
			"status":        "published",
			"isPublic":      false,
			"isSnapshot":    false,
			"minorVersion":  "1.0",
			"versionGroup":  version,
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
		}
	}

	postPath := "/exchange/api/v2/organizations/" + orgID + "/assets/" + groupID + "/" + assetID + "/" + version
	versionPath := "/exchange/api/v2/assets/" + groupID + "/" + assetID + "/" + version
	metaPath := "/exchange/api/v2/assets/" + groupID + "/" + assetID
	pagesPath := versionPath + "/portal/pages"

	var mu sync.Mutex
	name := ""
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		postPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want POST")
				return
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				testutil.ErrorResponse(w, http.StatusBadRequest, "bad multipart: "+err.Error())
				return
			}
			mu.Lock()
			name = r.FormValue("name")
			mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		},
		versionPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				mu.Lock()
				body := assetJSON(name)
				mu.Unlock()
				testutil.JSONResponse(w, http.StatusOK, body)
			case http.MethodDelete:
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want GET/DELETE")
			}
		},
		metaPath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNoContent)
		},
		pagesPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []interface{}{})
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

	// --- Part 1: CREATE with the semantic sub-type must NOT drift ---------------
	// Declared type is a KNOWN planned value; the fake normalizes it to "extension" on
	// readback; the applied state .type must come back EQUAL to the declared value or
	// real Terraform would fail the post-apply consistency check.
	for _, declared := range []string{"policy", "connector", "extension"} {
		t.Run("create_"+declared+"_no_drift", func(t *testing.T) {
			planRaw := assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
				"organization_id": tftypes.NewValue(tftypes.String, orgID),
				"group_id":        tftypes.NewValue(tftypes.String, groupID),
				"asset_id":        tftypes.NewValue(tftypes.String, assetID),
				"version":         tftypes.NewValue(tftypes.String, version),
				"name":            tftypes.NewValue(tftypes.String, "policy asset"),
				"type":            tftypes.NewValue(tftypes.String, declared),
				"status":          tftypes.NewValue(tftypes.String, "published"),
			})
			createReq := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
			createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			res.Create(ctx, createReq, createResp)
			if createResp.Diagnostics.HasError() {
				t.Fatalf("Create(type=%s) errors: %v", declared, createResp.Diagnostics.Errors())
			}
			var m AssetResourceModel
			if diags := createResp.State.Get(ctx, &m); diags.HasError() {
				t.Fatalf("Create(type=%s) State.Get errors: %v", declared, diags.Errors())
			}
			if got := m.Type.ValueString(); got != declared {
				t.Errorf("applied state .type = %q, want %q — the platform normalized to "+
					"\"extension\" and the provider failed to preserve the declared type "+
					"(this is the `inconsistent result after apply` regression)", got, declared)
			}

			// A follow-up refresh Read on that state must ALSO stay put (no perpetual diff).
			readReq := resource.ReadRequest{State: createResp.State}
			readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
			res.Read(ctx, readReq, readResp)
			if readResp.Diagnostics.HasError() {
				t.Fatalf("Read(type=%s) errors: %v", declared, readResp.Diagnostics.Errors())
			}
			var rm AssetResourceModel
			if diags := readResp.State.Get(ctx, &rm); diags.HasError() {
				t.Fatalf("Read(type=%s) State.Get errors: %v", declared, diags.Errors())
			}
			if got := rm.Type.ValueString(); got != declared {
				t.Errorf("refresh Read .type = %q, want %q — normalizeType did not preserve "+
					"the declared type on refresh (perpetual drift)", got, declared)
			}
		})
	}

	// --- Part 2: KNOWN LIMITATION — a bare import surfaces "extension" ----------
	// ImportState seeds ids but leaves .type null; the first Read then resolves it from
	// the API, which only knows "extension". This asserts the documented behavior so it
	// can't silently change. The RequiresReplaceOnTypeChange fix (separate test) is what
	// lets the user set type=policy afterward WITHOUT a destroy+recreate.
	t.Run("bare_import_surfaces_extension", func(t *testing.T) {
		importedRaw := assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
			"organization_id": tftypes.NewValue(tftypes.String, orgID),
			"group_id":        tftypes.NewValue(tftypes.String, groupID),
			"asset_id":        tftypes.NewValue(tftypes.String, assetID),
			"version":         tftypes.NewValue(tftypes.String, version),
			"id":              tftypes.NewValue(tftypes.String, groupID+"/"+assetID+"/"+version),
			// type deliberately omitted (null) — exactly what ImportState leaves behind.
		})
		readReq := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: importedRaw}}
		readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		res.Read(ctx, readReq, readResp)
		if readResp.Diagnostics.HasError() {
			t.Fatalf("Read(import) errors: %v", readResp.Diagnostics.Errors())
		}
		var m AssetResourceModel
		if diags := readResp.State.Get(ctx, &m); diags.HasError() {
			t.Fatalf("Read(import) State.Get errors: %v", diags.Errors())
		}
		if got := m.Type.ValueString(); got != "extension" {
			t.Errorf("bare import .type = %q, want \"extension\" — the API cannot recover the "+
				"semantic sub-type, so import must surface the stored super-type", got)
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
// contact_email and manager. contact_name/contact_email are Optional+Computed and
// manager is Computed-only (#65, read-only); ALL THREE carry UseStateForUnknown, so
// on an omit-path update the plan is frozen to the prior value and the Update readback
// (mapAssetToState called a second time, over the frozen plan) MUST reproduce that
// exact value. The readback must therefore be a
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
			name: "create file-backed type WITH file_path is allowed",
			// api_version is set too: a rest-api create requires BOTH a spec file and
			// api_version, so a case asserting a well-formed create is "allowed" must
			// satisfy both — otherwise the sibling api_version guard (which also runs on
			// create) would fire and this would no longer isolate the file_path guard.
			planRaw: raw(withFilePath(map[string]tftypes.Value{
				"type":        tftypes.NewValue(tftypes.String, "rest-api"),
				"api_version": tftypes.NewValue(tftypes.String, "v1"),
			}, "spec.json")),
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

// TestAssetResource_ModifyPlan_VersionCollisionGuard covers Guard #0 (Q-PROV-1 Option B):
// a CREATE — or a version BUMP — onto a GAV that ALREADY exists in Exchange is rejected at
// PLAN time (before any destroy) instead of hitting the raw apply-time 409
// ASSET_PRE_CONDITIONS_FAILED. The guard probes GetAsset; a mock server returns 200 for the
// "taken" version and 404 for anything else. A fileless type ("custom") is used so neither the
// file_path nor api_version sibling guard can fire — any diagnostic here is unambiguously the
// version-collision guard. It must NOT fire on: a create onto a FREE version, an in-place
// update (same version, we own it), a version bump onto a FREE version, a destroy plan, or an
// unknown version (deferred to apply). The nil-client path is covered implicitly by every
// other ModifyPlan test (they construct the resource without a client and never probe).
func TestAssetResource_ModifyPlan_VersionCollisionGuard(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	nullObj := tftypes.NewValue(objType, nil)

	// Mock Exchange: GET .../assets/grp/asset/1.0.0 -> 200 (taken); everything else 404.
	const takenVersion = "1.0.0"
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method == http.MethodGet && strings.HasSuffix(r.URL.Path, "/assets/grp/asset/"+takenVersion) {
			w.WriteHeader(http.StatusOK)
			_, _ = w.Write([]byte(`{"groupId":"grp","assetId":"asset","version":"1.0.0","type":"custom"}`))
			return
		}
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte(`{"status":404,"message":"not found"}`))
	}))
	defer server.Close()
	res.client = &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }
	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)

	// obj builds a fileless "custom" asset at the given version with fixed group/asset ids.
	obj := func(versionVal tftypes.Value) tftypes.Value {
		return assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
			"organization_id": str("org"),
			"group_id":        str("grp"),
			"asset_id":        str("asset"),
			"type":            str("custom"),
			"version":         versionVal,
		})
	}

	tests := []struct {
		name     string
		planRaw  tftypes.Value
		stateRaw tftypes.Value
		wantErr  bool
	}{
		{name: "create onto a TAKEN version is blocked", planRaw: obj(str(takenVersion)), stateRaw: nullObj, wantErr: true},
		{name: "create onto a FREE version is allowed", planRaw: obj(str("9.9.9")), stateRaw: nullObj, wantErr: false},
		{name: "version bump ONTO a taken version is blocked", planRaw: obj(str(takenVersion)), stateRaw: obj(str("0.9.0")), wantErr: true},
		{name: "version bump onto a FREE version is allowed", planRaw: obj(str("9.9.9")), stateRaw: obj(str("0.9.0")), wantErr: false},
		{name: "in-place update (same version we own) is allowed even though it exists", planRaw: obj(str(takenVersion)), stateRaw: obj(str(takenVersion)), wantErr: false},
		{name: "unknown version defers to apply (no probe, no error)", planRaw: obj(unknown), stateRaw: nullObj, wantErr: false},
		{name: "destroy plan is skipped", planRaw: nullObj, stateRaw: obj(str(takenVersion)), wantErr: false},
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

			hasCollision := false
			for _, d := range resp.Diagnostics {
				if strings.Contains(d.Summary(), "Exchange asset version already exists") {
					hasCollision = true
				}
			}
			if tt.wantErr && !hasCollision {
				t.Fatalf("expected the version-collision diagnostic, got: %v", resp.Diagnostics.Errors())
			}
			if !tt.wantErr && hasCollision {
				t.Fatalf("unexpected version-collision diagnostic: %v", resp.Diagnostics.Errors())
			}
		})
	}
}

// TestAssetTypeRequiresAPIVersion pins the api_version-required allowlist (#143). The
// multipart CREATE publish for these API-spec types 400s with
// MISSING_REQUIRED_PROPERTIES: apiVersion when properties.apiVersion is omitted —
// rest-api live-verified 2026-07-27; evented-api/grpc-api 2026-07-16 (E2E Finding B).
// The excluded types either don't require it (soap-api/graphql-api — file-backed but
// apiVersion not confirmed mandatory) or are fileless and unrelated; blocking any of them
// would be a false positive. The lookup must normalize case and surrounding whitespace.
func TestAssetTypeRequiresAPIVersion(t *testing.T) {
	requires := []string{"rest-api", "evented-api", "grpc-api"}
	// Types the allowlist must NEVER block. soap-api/graphql-api are deliberately here:
	// they are file-backed (so the file_path guard covers them) but their apiVersion
	// requirement is not confirmed, so this guard must leave them alone.
	notRequired := []string{"soap-api", "graphql-api", "custom", "app", "template", "policy", "ruleset", "llm", "mcp", "http-api", "", "some-future-type"}

	for _, ty := range requires {
		if !assetTypeRequiresAPIVersion(ty) {
			t.Errorf("assetTypeRequiresAPIVersion(%q) = false, want true (API-spec type requires apiVersion at create)", ty)
		}
	}
	for _, ty := range notRequired {
		if assetTypeRequiresAPIVersion(ty) {
			t.Errorf("assetTypeRequiresAPIVersion(%q) = true, want false (must never be blocked — unconfirmed/unrelated type)", ty)
		}
	}

	if !assetTypeRequiresAPIVersion("  Rest-API  ") {
		t.Error("assetTypeRequiresAPIVersion should normalize case+whitespace for an API-spec type")
	}
	if assetTypeRequiresAPIVersion("  SOAP-API  ") {
		t.Error("assetTypeRequiresAPIVersion should normalize case+whitespace for an excluded type (must stay false)")
	}
}

// TestAssetResource_ModifyPlan_APIVersionGuard covers the #143 plan-time guard: an API-spec
// type (rest-api/evented-api/grpc-api) CREATED or REPLACED without api_version is rejected by
// Exchange at apply with "400 MISSING_REQUIRED_PROPERTIES: apiVersion". On a version bump
// (RequiresReplace) the old version is destroyed BEFORE that failure, so — exactly like the
// file_path guard — raising it at plan time is strictly safer.
//
// The guard reads CONFIG, not plan, which this test models faithfully by giving each case a
// SEPARATE config and plan raw. api_version is Optional+Computed, so an UNCONFIGURED value and
// a value bound to another resource's not-yet-known output are BOTH Unknown in the plan — but
// they differ in the config (omitted => null; unresolved reference => unknown). The guard must
// fire on the omission and DEFER on the reference; the two "unknown in plan, differ in config"
// cases below are the crux that proves it. The guard must also NEVER fire on: a type not on the
// allowlist (soap-api/graphql-api/custom), an unresolved (unknown) type, an in-place update
// (post-import steady state), or a destroy plan. A file_path is always present so the sibling
// file_path guard can never fire and any diagnostic here is unambiguously the api_version guard.
func TestAssetResource_ModifyPlan_APIVersionGuard(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	raw := func(overrides map[string]tftypes.Value) tftypes.Value {
		return assetRawValue(ctx, schemaResp.Schema, overrides)
	}
	nullObj := tftypes.NewValue(objType, nil)

	unknown := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
	nullStr := tftypes.NewValue(tftypes.String, nil)
	str := func(s string) tftypes.Value { return tftypes.NewValue(tftypes.String, s) }

	// obj builds an asset object at a version + type, always WITH a file_path (so the sibling
	// file_path guard can never fire) and with the api_version cell set to exactly the caller's
	// value — so a case can put a different cell in the plan vs the config, which is the whole
	// point (Unknown-in-plan can be either null or unknown in config).
	obj := func(version, assetType string, apiVer tftypes.Value) tftypes.Value {
		return raw(map[string]tftypes.Value{
			"organization_id": str("org"),
			"group_id":        str("grp"),
			"asset_id":        str("asset"),
			"version":         str(version),
			"type":            str(assetType),
			"file_path":       str("spec.json"),
			"api_version":     apiVer,
		})
	}

	tests := []struct {
		name      string
		planRaw   tftypes.Value
		configRaw tftypes.Value
		stateRaw  tftypes.Value
		wantErr   bool
		wantType  string // type expected in the diagnostic detail when wantErr
	}{
		{
			// OMITTED: plan computes to unknown, config is null. This is the multi_version.tf bug.
			name:      "create rest-api with api_version OMITTED is blocked",
			planRaw:   obj("1.0.0", "rest-api", unknown),
			configRaw: obj("1.0.0", "rest-api", nullStr),
			stateRaw:  nullObj,
			wantErr:   true,
			wantType:  "rest-api",
		},
		{
			// REFERENCE: plan AND config are unknown (bound to a not-yet-known output). The guard
			// must DEFER (not fire) — blocking here would be a false positive on valid config.
			// Contrast with the OMITTED case above: identical plan, config is the discriminator.
			name:      "create rest-api with api_version from unresolved reference is deferred (not blocked)",
			planRaw:   obj("1.0.0", "rest-api", unknown),
			configRaw: obj("1.0.0", "rest-api", unknown),
			stateRaw:  nullObj,
			wantErr:   false,
		},
		{
			name:      "create rest-api with empty api_version is blocked",
			planRaw:   obj("1.0.0", "rest-api", str("")),
			configRaw: obj("1.0.0", "rest-api", str("")),
			stateRaw:  nullObj,
			wantErr:   true,
			wantType:  "rest-api",
		},
		{
			name:      "create rest-api WITH api_version is allowed",
			planRaw:   obj("1.0.0", "rest-api", str("v1")),
			configRaw: obj("1.0.0", "rest-api", str("v1")),
			stateRaw:  nullObj,
			wantErr:   false,
		},
		{
			name:      "create evented-api with api_version OMITTED is blocked",
			planRaw:   obj("1.0.0", "evented-api", unknown),
			configRaw: obj("1.0.0", "evented-api", nullStr),
			stateRaw:  nullObj,
			wantErr:   true,
			wantType:  "evented-api",
		},
		{
			name:      "create grpc-api with api_version OMITTED is blocked",
			planRaw:   obj("1.0.0", "grpc-api", unknown),
			configRaw: obj("1.0.0", "grpc-api", nullStr),
			stateRaw:  nullObj,
			wantErr:   true,
			wantType:  "grpc-api",
		},
		{
			name:      "replace (version bump) rest-api with api_version OMITTED is blocked",
			planRaw:   obj("2.0.0", "rest-api", unknown),
			configRaw: obj("2.0.0", "rest-api", nullStr),
			stateRaw:  obj("1.0.0", "rest-api", str("v1")),
			wantErr:   true,
			wantType:  "rest-api",
		},
		{
			// In-place update (same version): creating=false, replacing=false, so the guard block
			// is skipped entirely — api_version being absent from config is the normal post-import
			// steady state and must never error.
			name:      "in-place update rest-api without api_version is allowed (post-import steady state)",
			planRaw:   obj("1.0.0", "rest-api", str("v1")),
			configRaw: obj("1.0.0", "rest-api", nullStr),
			stateRaw:  obj("1.0.0", "rest-api", str("v1")),
			wantErr:   false,
		},
		{
			name:      "create soap-api without api_version is allowed (not on allowlist)",
			planRaw:   obj("1.0.0", "soap-api", unknown),
			configRaw: obj("1.0.0", "soap-api", nullStr),
			stateRaw:  nullObj,
			wantErr:   false,
		},
		{
			name:      "create graphql-api without api_version is allowed (not on allowlist)",
			planRaw:   obj("1.0.0", "graphql-api", unknown),
			configRaw: obj("1.0.0", "graphql-api", nullStr),
			stateRaw:  nullObj,
			wantErr:   false,
		},
		{
			name: "create with unknown type is skipped (cannot classify)",
			planRaw: raw(map[string]tftypes.Value{
				"type":        unknown,
				"api_version": unknown,
			}),
			configRaw: raw(map[string]tftypes.Value{
				"type":        unknown,
				"api_version": nullStr,
			}),
			stateRaw: nullObj,
			wantErr:  false,
		},
		{
			name:      "destroy plan (null plan) is skipped without panic",
			planRaw:   nullObj,
			configRaw: nullObj,
			stateRaw:  obj("1.0.0", "rest-api", str("v1")),
			wantErr:   false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			req := resource.ModifyPlanRequest{
				Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: tt.planRaw},
				State:  tfsdk.State{Schema: schemaResp.Schema, Raw: tt.stateRaw},
				Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: tt.configRaw},
			}
			resp := &resource.ModifyPlanResponse{}
			res.ModifyPlan(ctx, req, resp)

			if !tt.wantErr {
				if resp.Diagnostics.HasError() {
					t.Fatalf("expected no error, got: %v", resp.Diagnostics.Errors())
				}
				return
			}

			var detail string
			found := false
			for _, d := range resp.Diagnostics {
				if strings.Contains(d.Summary(), "Missing api_version for an API-spec asset type") {
					detail, found = d.Detail(), true
				}
			}
			if !found {
				t.Fatalf("expected the api_version guard diagnostic, got: %v", resp.Diagnostics.Errors())
			}
			if !strings.Contains(detail, tt.wantType) {
				t.Errorf("diagnostic detail should name the offending type %q; got %q", tt.wantType, detail)
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

// TestAssetResource_StatusValidator_OneOf pins the #68 plan-time status validator. The
// Exchange status API is asymmetric and CASE-SENSITIVE (both LIVE-VERIFIED 2026-07-22 from
// the platform's own 400 bodies): the multipart CREATE accepts {development, published} and
// the PUT /status UPDATE accepts {published, deprecated}. The schema validator is the union
// OneOf(development, published, deprecated) so a typo (e.g. "Published") is caught at PLAN
// time — crucially before a `version`-RequiresReplace apply destroys the prior version. This
// test drives the wired validator(s) directly rather than asserting a hard-coded count, so it
// stays correct if the validator set is refactored.
func TestAssetResource_StatusValidator_OneOf(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	attr, ok := schemaResp.Schema.Attributes["status"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("status attribute is not a schema.StringAttribute")
	}
	validators := attr.StringValidators()
	if len(validators) == 0 {
		t.Fatal("status attribute has no validators; expected OneOf(development, published, deprecated)")
	}

	// An attribute's validation is the union of ALL its validators, so a value is
	// accepted only if NO wired validator rejects it. Drive them all and aggregate.
	validate := func(v types.String) diag.Diagnostics {
		var diags diag.Diagnostics
		for _, sv := range validators {
			r := &validator.StringResponse{}
			sv.ValidateString(ctx, validator.StringRequest{
				Path:        path.Root("status"),
				ConfigValue: v,
			}, r)
			diags.Append(r.Diagnostics...)
		}
		return diags
	}

	for _, s := range []string{"development", "published", "deprecated"} {
		if d := validate(types.StringValue(s)); d.HasError() {
			t.Errorf("status %q should be accepted, got: %v", s, d.Errors())
		}
	}

	// Case typos are the headline case: the API does NOT normalize case, so these must be
	// rejected at plan time rather than silently reaching an apply and 400-ing.
	for _, s := range []string{"Published", "Development", "DEPRECATED", "publish", "deleted", "draft", "", "  published  "} {
		if d := validate(types.StringValue(s)); !d.HasError() {
			t.Errorf("status %q should be rejected by OneOf, but was accepted", s)
		}
	}

	// Optional+Computed: an absent config (null) or an apply-time-derived value (unknown)
	// must never trip the validator.
	if d := validate(types.StringNull()); d.HasError() {
		t.Errorf("null status should be skipped by the validator, got: %v", d.Errors())
	}
	if d := validate(types.StringUnknown()); d.HasError() {
		t.Errorf("unknown status should be skipped by the validator, got: %v", d.Errors())
	}
}

// TestAssetResource_ModifyPlan_DevelopmentStatusGuard covers the #68 plan-time guard for the
// asymmetry the plain OneOf validator cannot express: `development` is CREATE-only. Moving an
// already-published version's status TO `development` in place hits PUT /status, which the
// platform rejects with HTTP 400 ("allowed values: published, deprecated"). The guard must
// fire ONLY on an in-place change to `development` — never on create or on a version-bump
// replace (both publish through multipart, which DOES accept development), and never when the
// status is not actually changing. A fileless type (custom) is used throughout so the sibling
// file_path guard can never fire; any diagnostic here is therefore unambiguously the status guard.
func TestAssetResource_ModifyPlan_DevelopmentStatusGuard(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)

	raw := func(overrides map[string]tftypes.Value) tftypes.Value {
		return assetRawValue(ctx, schemaResp.Schema, overrides)
	}
	nullObj := tftypes.NewValue(objType, nil)

	// base builds a fileless (custom-type) asset at a given version + status. Keeping the
	// replace-forcing identity (org/group/asset/type) constant means a version change is the
	// only replace trigger, matching assetReplaceTriggered.
	base := func(version, status string) map[string]tftypes.Value {
		return map[string]tftypes.Value{
			"organization_id": tftypes.NewValue(tftypes.String, "org"),
			"group_id":        tftypes.NewValue(tftypes.String, "grp"),
			"asset_id":        tftypes.NewValue(tftypes.String, "asset"),
			"version":         tftypes.NewValue(tftypes.String, version),
			"type":            tftypes.NewValue(tftypes.String, "custom"),
			"status":          tftypes.NewValue(tftypes.String, status),
		}
	}

	tests := []struct {
		name     string
		planRaw  tftypes.Value
		stateRaw tftypes.Value
		wantErr  bool
	}{
		{
			name:     "in-place published->development is blocked",
			planRaw:  raw(base("1.0.0", "development")),
			stateRaw: raw(base("1.0.0", "published")),
			wantErr:  true,
		},
		{
			name:     "in-place deprecated->development is blocked",
			planRaw:  raw(base("1.0.0", "development")),
			stateRaw: raw(base("1.0.0", "deprecated")),
			wantErr:  true,
		},
		{
			name:     "create with development is allowed (multipart accepts it)",
			planRaw:  raw(base("1.0.0", "development")),
			stateRaw: nullObj,
			wantErr:  false,
		},
		{
			name:     "replace (version bump) publishing development is allowed",
			planRaw:  raw(base("2.0.0", "development")),
			stateRaw: raw(base("1.0.0", "published")),
			wantErr:  false,
		},
		{
			name:     "in-place development->development (no change) is allowed",
			planRaw:  raw(base("1.0.0", "development")),
			stateRaw: raw(base("1.0.0", "development")),
			wantErr:  false,
		},
		{
			name:     "in-place ->published is allowed",
			planRaw:  raw(base("1.0.0", "published")),
			stateRaw: raw(base("1.0.0", "development")),
			wantErr:  false,
		},
		{
			name:     "in-place ->deprecated is allowed",
			planRaw:  raw(base("1.0.0", "deprecated")),
			stateRaw: raw(base("1.0.0", "published")),
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

			if tt.wantErr {
				if !resp.Diagnostics.HasError() {
					t.Fatalf("expected a plan-time error, got none")
				}
				found := false
				for _, d := range resp.Diagnostics {
					if strings.Contains(d.Summary(), `Cannot set status to "development" on an existing asset version`) {
						found = true
					}
				}
				if !found {
					t.Fatalf("expected the development-status guard diagnostic, got: %v", resp.Diagnostics.Errors())
				}
				return
			}
			if resp.Diagnostics.HasError() {
				t.Fatalf("expected no error, got: %v", resp.Diagnostics.Errors())
			}
		})
	}
}

// TestAssetResource_ManagerReadOnly_Schema pins #65: the Exchange metadata PATCH endpoint
// rejects any attempt to SET the manager (HTTP 403 for a username, HTTP 400 for a uuid —
// LIVE-VERIFIED 2026-07-22). There is no supported way to write it, so the attribute must be
// Computed-only (surfaced from Exchange, never configurable). If it were Optional, a user
// could set it and only discover at apply time that it 403s, killing the whole apply.
func TestAssetResource_ManagerReadOnly_Schema(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	attr, ok := schemaResp.Schema.Attributes["manager"].(schema.StringAttribute)
	if !ok {
		t.Fatalf("manager attribute is not a schema.StringAttribute")
	}
	if !attr.IsComputed() {
		t.Error("manager must be Computed (read-only value surfaced from Exchange)")
	}
	if attr.IsOptional() {
		t.Error("manager must NOT be Optional — it is read-only (setting it returns HTTP 403/400)")
	}
	if attr.IsRequired() {
		t.Error("manager must NOT be Required — it is read-only")
	}
}

// TestAssetResource_ModifyPlan_FillsComputedChildrenByKey pins #67: the computed CHILDREN
// of a CONFIGURED nested list (pages.page_path; instances.instance_id / is_public) must be
// carried over from prior state on an in-place update instead of going "(known after
// apply)". The list-level UseStateForUnknown modifiers only fire when the WHOLE list is
// unknown; once the user configures the list, each element's computed child is unknown,
// which churns the plan AND trips the Update-path !plan.X.Equal(state.X) sync gate into a
// needless re-sync. ModifyPlan fills those unknown children from the prior-state element
// with the SAME KEY (page_name / instance name).
//
// The fill MUST be keyed, never positional — the reorder sub-test is the load-bearing
// proof: a positional copy would assign the wrong page_path and (in a real apply) crash
// with "inconsistent result after apply". And it MUST be skipped on create/replace, where
// a new version's children are genuinely unknown (the create/replace sub-tests pin that a
// stale value is NOT copied forward).
func TestAssetResource_ModifyPlan_FillsComputedChildrenByKey(t *testing.T) {
	res := NewAssetResource().(*AssetResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	nullObj := tftypes.NewValue(objType, nil)

	strptr := func(s string) *string { return &s }
	boolptr := func(b bool) *bool { return &b }

	pageElem := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"page_name": tftypes.String, "content": tftypes.String, "page_path": tftypes.String,
	}}
	instElem := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name": tftypes.String, "endpoint_uri": tftypes.String, "is_public": tftypes.Bool, "instance_id": tftypes.String,
	}}

	// mkPage builds a page element; a nil path means page_path is unknown (as Terraform
	// renders a computed child of a configured element before apply).
	mkPage := func(name, content string, path *string) tftypes.Value {
		pv := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		if path != nil {
			pv = tftypes.NewValue(tftypes.String, *path)
		}
		return tftypes.NewValue(pageElem, map[string]tftypes.Value{
			"page_name": tftypes.NewValue(tftypes.String, name),
			"content":   tftypes.NewValue(tftypes.String, content),
			"page_path": pv,
		})
	}
	pageList := func(elems ...tftypes.Value) tftypes.Value {
		return tftypes.NewValue(tftypes.List{ElementType: pageElem}, elems)
	}
	// mkInst builds an instance element; nil isPublic/id means that child is unknown.
	mkInst := func(name, uri string, isPublic *bool, id *string) tftypes.Value {
		ipv := tftypes.NewValue(tftypes.Bool, tftypes.UnknownValue)
		if isPublic != nil {
			ipv = tftypes.NewValue(tftypes.Bool, *isPublic)
		}
		idv := tftypes.NewValue(tftypes.String, tftypes.UnknownValue)
		if id != nil {
			idv = tftypes.NewValue(tftypes.String, *id)
		}
		return tftypes.NewValue(instElem, map[string]tftypes.Value{
			"name":         tftypes.NewValue(tftypes.String, name),
			"endpoint_uri": tftypes.NewValue(tftypes.String, uri),
			"is_public":    ipv,
			"instance_id":  idv,
		})
	}
	instList := func(elems ...tftypes.Value) tftypes.Value {
		return tftypes.NewValue(tftypes.List{ElementType: instElem}, elems)
	}

	// build a full asset object at a version (fileless custom type so no other guard fires),
	// overlaying the given attribute overrides (pages / instances).
	build := func(version string, extra map[string]tftypes.Value) tftypes.Value {
		o := map[string]tftypes.Value{
			"organization_id": tftypes.NewValue(tftypes.String, "org"),
			"group_id":        tftypes.NewValue(tftypes.String, "grp"),
			"asset_id":        tftypes.NewValue(tftypes.String, "asset"),
			"version":         tftypes.NewValue(tftypes.String, version),
			"type":            tftypes.NewValue(tftypes.String, "custom"),
			"status":          tftypes.NewValue(tftypes.String, "published"),
		}
		for k, v := range extra {
			o[k] = v
		}
		return assetRawValue(ctx, schemaResp.Schema, o)
	}

	// run ModifyPlan and return the resulting plan model.
	run := func(t *testing.T, stateRaw, planRaw tftypes.Value) AssetResourceModel {
		t.Helper()
		req := resource.ModifyPlanRequest{
			Plan:   tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
			State:  tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
			Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: planRaw},
		}
		// resp.Plan must be pre-seeded with the proposed plan, exactly as the framework
		// does, so ModifyPlan's SetAttribute has a plan to mutate.
		resp := &resource.ModifyPlanResponse{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
		res.ModifyPlan(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("unexpected diagnostics: %v", resp.Diagnostics.Errors())
		}
		var out AssetResourceModel
		if d := resp.Plan.Get(ctx, &out); d.HasError() {
			t.Fatalf("resp.Plan.Get: %v", d.Errors())
		}
		return out
	}
	pagesOf := func(t *testing.T, m AssetResourceModel) []PageModel {
		t.Helper()
		var ps []PageModel
		if d := m.Pages.ElementsAs(ctx, &ps, false); d.HasError() {
			t.Fatalf("pages ElementsAs: %v", d.Errors())
		}
		return ps
	}
	instsOf := func(t *testing.T, m AssetResourceModel) []InstanceModel {
		t.Helper()
		var is []InstanceModel
		if d := m.Instances.ElementsAs(ctx, &is, false); d.HasError() {
			t.Fatalf("instances ElementsAs: %v", d.Errors())
		}
		return is
	}

	t.Run("in-place: unknown page_path filled from state by page_name", func(t *testing.T) {
		state := build("1.0.0", map[string]tftypes.Value{"pages": pageList(mkPage("overview", "old", strptr("p/over")))})
		plan := build("1.0.0", map[string]tftypes.Value{"pages": pageList(mkPage("overview", "new", nil))}) // content edited, path unknown
		ps := pagesOf(t, run(t, state, plan))
		if len(ps) != 1 {
			t.Fatalf("want 1 page, got %d", len(ps))
		}
		if ps[0].PagePath.IsUnknown() || ps[0].PagePath.ValueString() != "p/over" {
			t.Fatalf("page_path not filled from state: got %v", ps[0].PagePath)
		}
	})

	t.Run("in-place REORDERED: fill is keyed, not positional", func(t *testing.T) {
		// State order [home, overview]; plan order [overview, home]. A positional copy
		// would give overview home's path — this asserts the keyed match instead.
		state := build("1.0.0", map[string]tftypes.Value{"pages": pageList(
			mkPage("home", "h", strptr("p/home")),
			mkPage("overview", "o", strptr("p/over")),
		)})
		plan := build("1.0.0", map[string]tftypes.Value{"pages": pageList(
			mkPage("overview", "o", nil),
			mkPage("home", "h", nil),
		)})
		ps := pagesOf(t, run(t, state, plan))
		if len(ps) != 2 {
			t.Fatalf("want 2 pages, got %d", len(ps))
		}
		if ps[0].PageName.ValueString() != "overview" || ps[0].PagePath.ValueString() != "p/over" {
			t.Fatalf("elem0 wrong (positional bug?): name=%v path=%v", ps[0].PageName, ps[0].PagePath)
		}
		if ps[1].PageName.ValueString() != "home" || ps[1].PagePath.ValueString() != "p/home" {
			t.Fatalf("elem1 wrong (positional bug?): name=%v path=%v", ps[1].PageName, ps[1].PagePath)
		}
	})

	t.Run("in-place: a NEW page with no state match keeps unknown page_path", func(t *testing.T) {
		state := build("1.0.0", map[string]tftypes.Value{"pages": pageList(mkPage("overview", "o", strptr("p/over")))})
		plan := build("1.0.0", map[string]tftypes.Value{"pages": pageList(
			mkPage("overview", "o", nil),
			mkPage("guide", "g", nil), // brand-new page
		)})
		ps := pagesOf(t, run(t, state, plan))
		if len(ps) != 2 {
			t.Fatalf("want 2 pages, got %d", len(ps))
		}
		if ps[0].PagePath.ValueString() != "p/over" {
			t.Fatalf("existing page not filled: %v", ps[0].PagePath)
		}
		if !ps[1].PagePath.IsUnknown() {
			t.Fatalf("new page page_path must stay unknown (known after apply), got %v", ps[1].PagePath)
		}
	})

	t.Run("in-place: unknown instance_id and is_public filled from state by name", func(t *testing.T) {
		state := build("1.0.0", map[string]tftypes.Value{"instances": instList(mkInst("Prod", "https://a", boolptr(true), strptr("i-1")))})
		plan := build("1.0.0", map[string]tftypes.Value{"instances": instList(mkInst("Prod", "https://a", nil, nil))})
		is := instsOf(t, run(t, state, plan))
		if len(is) != 1 {
			t.Fatalf("want 1 instance, got %d", len(is))
		}
		if is[0].InstanceID.IsUnknown() || is[0].InstanceID.ValueString() != "i-1" {
			t.Fatalf("instance_id not filled: %v", is[0].InstanceID)
		}
		if is[0].IsPublic.IsUnknown() || is[0].IsPublic.ValueBool() != true {
			t.Fatalf("is_public not filled: %v", is[0].IsPublic)
		}
	})

	t.Run("in-place: an explicitly configured is_public is NOT overwritten", func(t *testing.T) {
		state := build("1.0.0", map[string]tftypes.Value{"instances": instList(mkInst("Prod", "https://a", boolptr(true), strptr("i-1")))})
		// User flips is_public to a KNOWN false; only instance_id is unknown/computed.
		plan := build("1.0.0", map[string]tftypes.Value{"instances": instList(mkInst("Prod", "https://a", boolptr(false), nil))})
		is := instsOf(t, run(t, state, plan))
		if is[0].InstanceID.ValueString() != "i-1" {
			t.Fatalf("instance_id not filled: %v", is[0].InstanceID)
		}
		if is[0].IsPublic.IsUnknown() || is[0].IsPublic.ValueBool() != false {
			t.Fatalf("explicit is_public=false must be preserved, got %v", is[0].IsPublic)
		}
	})

	t.Run("CREATE (null state): page_path stays unknown (no stale fill)", func(t *testing.T) {
		plan := build("1.0.0", map[string]tftypes.Value{"pages": pageList(mkPage("overview", "o", nil))})
		ps := pagesOf(t, run(t, nullObj, plan))
		if !ps[0].PagePath.IsUnknown() {
			t.Fatalf("on create page_path must remain unknown, got %v", ps[0].PagePath)
		}
	})

	t.Run("REPLACE (version bump): page_path stays unknown (old version's value must NOT copy forward)", func(t *testing.T) {
		state := build("1.0.0", map[string]tftypes.Value{"pages": pageList(mkPage("overview", "o", strptr("p/over-v1")))})
		plan := build("2.0.0", map[string]tftypes.Value{"pages": pageList(mkPage("overview", "o", nil))})
		ps := pagesOf(t, run(t, state, plan))
		if !ps[0].PagePath.IsUnknown() {
			t.Fatalf("on replace the new version's page_path must stay unknown, got %v (stale copy would crash apply)", ps[0].PagePath)
		}
	})
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

// TestAssetResource_Lifecycle_Hermetic drives the resource's REAL Create → Read →
// Update → Delete → ImportState methods end-to-end against an in-process, stateful
// fake Exchange server — the closest thing to a `terraform apply` we can run without
// the network or a live org. It is the hermetic self-test for task #28: it proves the
// four CRUD verbs plus import agree with the client's URL/verb/header/status contracts
// and, crucially, that the sequence produces NO drift and NO error diagnostics.
//
// The subject is a type="custom", file_path=null asset on purpose. That combination is
// the one code path with zero disk I/O (buildAssetMultipart skips the file part when
// FilePath=="", asset.go client ~line 1210) and zero plan-time file guard (custom is not
// in assetTypeRequiringFile), so the test needs no fixture files and exercises the
// metadata-only lifecycle that every asset type shares.
//
// The fake is STATEFUL: the POST handler records the multipart name/description/status
// the provider actually sent, and every subsequent GET echoes them back. That is what
// lets us assert the post-create PATCH is correctly SKIPPED (the create readback already
// matches the plan, so needsPatch stays false) — a silent spurious PATCH here would mean
// the Create path can't tell "already correct" from "needs fixup". The PATCH handler then
// mutates the stored description so the Update readback observes the change, exactly as
// the live metadata endpoint would.
func TestAssetResource_Lifecycle_Hermetic(t *testing.T) {
	const (
		orgID   = "test-org-id"
		groupID = "g"
		assetID = "a"
		version = "1.0.0"
	)

	// fakeAsset is the server's mutable backing store. The mutex guards it because
	// httptest serves each request on its own goroutine; the CRUD calls are serial in
	// practice, but the lock keeps the test clean under `go test -race`.
	type fakeAsset struct {
		mu          sync.Mutex
		name        string
		description string
		status      string
		posted      bool
		patched     bool
		deleted     bool
		deleteType  string
	}
	fa := &fakeAsset{}

	// assetJSON renders the current stored state as the Exchange GET body. Empty
	// collections keep all five non-fatal read helpers on their clean, drift-free path.
	assetJSON := func() map[string]interface{} {
		return map[string]interface{}{
			"groupId":       groupID,
			"assetId":       assetID,
			"version":       version,
			"name":          fa.name,
			"description":   fa.description,
			"type":          "custom",
			"status":        fa.status,
			"isPublic":      false,
			"isSnapshot":    false,
			"minorVersion":  "1.0",
			"versionGroup":  version,
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
		}
	}

	postPath := "/exchange/api/v2/organizations/" + orgID + "/assets/" + groupID + "/" + assetID + "/" + version
	versionPath := "/exchange/api/v2/assets/" + groupID + "/" + assetID + "/" + version
	metaPath := "/exchange/api/v2/assets/" + groupID + "/" + assetID
	pagesPath := versionPath + "/portal/pages"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		// Publish (multipart POST). Capture exactly what the provider sent so the GET
		// readback mirrors the plan and the post-create PATCH is proven unnecessary.
		postPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want POST")
				return
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				testutil.ErrorResponse(w, http.StatusBadRequest, "bad multipart: "+err.Error())
				return
			}
			fa.mu.Lock()
			fa.name = r.FormValue("name")
			fa.description = r.FormValue("description")
			fa.status = r.FormValue("status")
			fa.posted = true
			fa.mu.Unlock()
			w.WriteHeader(http.StatusCreated)
		},
		// Version-scoped GET (readback) + DELETE (hard delete).
		versionPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodGet:
				fa.mu.Lock()
				body := assetJSON()
				fa.mu.Unlock()
				testutil.JSONResponse(w, http.StatusOK, body)
			case http.MethodDelete:
				fa.mu.Lock()
				fa.deleted = true
				fa.deleteType = r.Header.Get("x-delete-type")
				fa.mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want GET/DELETE")
			}
		},
		// Asset-scoped metadata PATCH (name/description/contact). Mutate the store so the
		// Update readback observes the new description.
		metaPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want PATCH")
				return
			}
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				testutil.ErrorResponse(w, http.StatusBadRequest, "bad json: "+err.Error())
				return
			}
			fa.mu.Lock()
			if v, ok := body["name"].(string); ok {
				fa.name = v
			}
			if v, ok := body["description"].(string); ok {
				fa.description = v
			}
			fa.patched = true
			fa.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
		},
		// Portal pages: empty list keeps readPagesIntoState on its clean path (and .terms
		// stays unregistered → 404 → T&C "" via the non-fatal readTermsIntoState).
		pagesPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []interface{}{})
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

	// --- CREATE ---------------------------------------------------------------
	createPlanRaw := assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, orgID),
		"group_id":        tftypes.NewValue(tftypes.String, groupID),
		"asset_id":        tftypes.NewValue(tftypes.String, assetID),
		"version":         tftypes.NewValue(tftypes.String, version),
		"name":            tftypes.NewValue(tftypes.String, "lifecycle asset"),
		"type":            tftypes.NewValue(tftypes.String, "custom"),
		"status":          tftypes.NewValue(tftypes.String, "published"),
		"description":     tftypes.NewValue(tftypes.String, "initial description"),
	})

	createReq := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: createPlanRaw}}
	createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, createReq, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors: %v", createResp.Diagnostics.Errors())
	}
	if !fa.posted {
		t.Fatal("Create() never issued the publish POST")
	}
	if fa.patched {
		t.Error("Create() issued a spurious post-create metadata PATCH; the GET readback already matched the plan, so needsPatch must have stayed false")
	}

	var created AssetResourceModel
	if diags := createResp.State.Get(ctx, &created); diags.HasError() {
		t.Fatalf("Create State.Get errors: %v", diags.Errors())
	}
	if created.ID.ValueString() != "g/a/1.0.0" {
		t.Errorf("Create: ID = %q, want %q", created.ID.ValueString(), "g/a/1.0.0")
	}
	if created.Name.ValueString() != "lifecycle asset" {
		t.Errorf("Create: Name = %q, want %q", created.Name.ValueString(), "lifecycle asset")
	}
	if created.Description.ValueString() != "initial description" {
		t.Errorf("Create: Description = %q, want %q", created.Description.ValueString(), "initial description")
	}
	if created.Status.ValueString() != "published" {
		t.Errorf("Create: Status = %q, want %q", created.Status.ValueString(), "published")
	}
	// Computed collections must settle to concrete (never unknown) after apply.
	if created.Tags.IsNull() || created.Tags.IsUnknown() || len(created.Tags.Elements()) != 0 {
		t.Errorf("Create: Tags should be a concrete empty list, got %v", created.Tags)
	}
	if created.TermsAndConditions.ValueString() != "" {
		t.Errorf("Create: TermsAndConditions = %q, want empty", created.TermsAndConditions.ValueString())
	}

	// --- READ (drift gate) ----------------------------------------------------
	// Feed Create's ACTUAL output state into Read. The fake is unchanged, so a correct
	// Read must reproduce byte-identical raw state: this is the zero-drift assertion.
	createStateRaw := createResp.State.Raw
	readReq := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: createStateRaw}}
	readResp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: createStateRaw}}
	res.Read(ctx, readReq, readResp)

	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", readResp.Diagnostics.Errors())
	}
	if readResp.State.Raw.IsNull() {
		t.Fatal("Read() unexpectedly removed the resource from state")
	}
	if !createStateRaw.Equal(readResp.State.Raw) {
		t.Errorf("Read() drifted from Create state:\n create = %v\n read   = %v", createStateRaw, readResp.State.Raw)
	}

	// --- UPDATE (description change) ------------------------------------------
	updatePlanRaw := assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
		"organization_id": tftypes.NewValue(tftypes.String, orgID),
		"group_id":        tftypes.NewValue(tftypes.String, groupID),
		"asset_id":        tftypes.NewValue(tftypes.String, assetID),
		"version":         tftypes.NewValue(tftypes.String, version),
		"name":            tftypes.NewValue(tftypes.String, "lifecycle asset"),
		"type":            tftypes.NewValue(tftypes.String, "custom"),
		"status":          tftypes.NewValue(tftypes.String, "published"),
		"description":     tftypes.NewValue(tftypes.String, "updated description"),
	})
	updateReq := resource.UpdateRequest{
		Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: updatePlanRaw},
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: readResp.State.Raw},
	}
	updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: readResp.State.Raw}}
	res.Update(ctx, updateReq, updateResp)

	if updateResp.Diagnostics.HasError() {
		t.Fatalf("Update() reported errors: %v", updateResp.Diagnostics.Errors())
	}
	if !fa.patched {
		t.Error("Update() never issued the metadata PATCH for the changed description")
	}
	var updated AssetResourceModel
	if diags := updateResp.State.Get(ctx, &updated); diags.HasError() {
		t.Fatalf("Update State.Get errors: %v", diags.Errors())
	}
	if updated.Description.ValueString() != "updated description" {
		t.Errorf("Update: Description = %q, want %q", updated.Description.ValueString(), "updated description")
	}
	// Unchanged fields must remain stable across the update.
	if updated.Name.ValueString() != "lifecycle asset" {
		t.Errorf("Update: Name = %q, want %q (unchanged)", updated.Name.ValueString(), "lifecycle asset")
	}
	if updated.ID.ValueString() != "g/a/1.0.0" {
		t.Errorf("Update: ID = %q, want %q (unchanged)", updated.ID.ValueString(), "g/a/1.0.0")
	}
	if updated.Status.ValueString() != "published" {
		t.Errorf("Update: Status = %q, want %q (unchanged)", updated.Status.ValueString(), "published")
	}

	// --- DELETE ---------------------------------------------------------------
	deleteReq := resource.DeleteRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: updateResp.State.Raw}}
	deleteResp := &resource.DeleteResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: updateResp.State.Raw}}
	res.Delete(ctx, deleteReq, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete() reported errors: %v", deleteResp.Diagnostics.Errors())
	}
	if !fa.deleted {
		t.Fatal("Delete() never issued the version DELETE")
	}
	if fa.deleteType != "hard-delete" {
		t.Errorf("Delete: x-delete-type header = %q, want %q (soft-delete leaves a tombstone that blocks recreation)", fa.deleteType, "hard-delete")
	}

	// --- IMPORTSTATE ----------------------------------------------------------
	importReq := resource.ImportStateRequest{ID: "g/a/1.0.0"}
	importResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: assetRawValue(ctx, schemaResp.Schema, nil)}}
	res.ImportState(ctx, importReq, importResp)

	if importResp.Diagnostics.HasError() {
		t.Fatalf("ImportState() reported errors: %v", importResp.Diagnostics.Errors())
	}
	var imported AssetResourceModel
	if diags := importResp.State.Get(ctx, &imported); diags.HasError() {
		t.Fatalf("Import State.Get errors: %v", diags.Errors())
	}
	if imported.GroupID.ValueString() != groupID {
		t.Errorf("Import: group_id = %q, want %q", imported.GroupID.ValueString(), groupID)
	}
	if imported.AssetID.ValueString() != assetID {
		t.Errorf("Import: asset_id = %q, want %q", imported.AssetID.ValueString(), assetID)
	}
	if imported.Version.ValueString() != version {
		t.Errorf("Import: version = %q, want %q", imported.Version.ValueString(), version)
	}
	if imported.ID.ValueString() != "g/a/1.0.0" {
		t.Errorf("Import: id = %q, want %q", imported.ID.ValueString(), "g/a/1.0.0")
	}
	if imported.OrganizationID.ValueString() != groupID {
		t.Errorf("Import: organization_id = %q, want %q (import seeds org_id from group_id)", imported.OrganizationID.ValueString(), groupID)
	}
}

// TestAssetResource_ReadPagesIntoState_HomePageFiltering locks in the READ-side contract
// for the Exchange portal "home" landing page — the tractable half of the #66 pages gaps,
// established by live-probing the portal-pages API (2026-07-23):
//
//   - Exchange auto-provisions a "home" page on EVERY asset version. Until a user writes
//     content to it, the API reports it as {"name":"home","synthetic":true}. A synthetic
//     home is platform-owned, not user-managed, so it MUST be filtered out — otherwise
//     every asset would show phantom `pages` drift on Read/import.
//   - Once content is published to "home", the API flips it to non-synthetic. A
//     non-synthetic "home" IS user-managed and MUST surface in state (with its content)
//     so a configured `pages { page_name = "home" ... }` block round-trips without drift.
//
// This is the READ counterpart to TestAssetResource_SyncPages_AdoptsAutoProvisionedHomePage
// (the WRITE side); together they cover both directions of home-page management. It also
// documents the platform limitations found live: pages are FLAT (no nesting) and FIXED to
// creation order (no reorder endpoint), so there is deliberately no schema surface for
// hierarchy or explicit ordering to exercise here.
func TestAssetResource_ReadPagesIntoState_HomePageFiltering(t *testing.T) {
	listPath := "/exchange/api/v2/assets/g/a/1.0.0/portal/pages"

	tests := []struct {
		name      string
		pages     []map[string]interface{}
		content   map[string]string // page path -> markdown body served by GET pages/{path}
		wantNames []string          // expected page_name values, in order
	}{
		{
			name:      "synthetic home is filtered out (phantom-drift guard)",
			pages:     []map[string]interface{}{{"name": "home", "path": "home", "synthetic": true}},
			wantNames: nil,
		},
		{
			name:      "non-synthetic (managed) home surfaces with content",
			pages:     []map[string]interface{}{{"name": "home", "path": "home"}},
			content:   map[string]string{"home": "<h2>Welcome</h2>"},
			wantNames: []string{"home"},
		},
		{
			name: "synthetic home filtered; sibling user page kept",
			pages: []map[string]interface{}{
				{"name": "home", "path": "home", "synthetic": true},
				{"name": "guide", "path": "tok-grf/guide"},
			},
			content:   map[string]string{"tok-grf/guide": "# Guide"},
			wantNames: []string{"guide"},
		},
		{
			name: "managed home + user page both surface in API listing (creation) order",
			pages: []map[string]interface{}{
				{"name": "home", "path": "home"},
				{"name": "guide", "path": "tok-grf/guide"},
			},
			content:   map[string]string{"home": "<h2>Welcome</h2>", "tok-grf/guide": "# Guide"},
			wantNames: []string{"home", "guide"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				listPath: func(w http.ResponseWriter, r *http.Request) {
					testutil.JSONResponse(w, http.StatusOK, tt.pages)
				},
			}
			// Register a content handler for every listed page path.
			for _, p := range tt.pages {
				pth, _ := p["path"].(string)
				body := tt.content[pth]
				handlers[listPath+"/"+pth] = func(w http.ResponseWriter, r *http.Request) {
					w.Header().Set("Content-Type", "text/markdown")
					w.WriteHeader(http.StatusOK)
					_, _ = w.Write([]byte(body))
				}
			}
			server := testutil.MockHTTPServer(t, handlers)

			res := NewAssetResource().(*AssetResource)
			res.client = &exchange.AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

			// readPagesIntoState reads GroupID/AssetID/Version + prior Pages off the model.
			// A null prior Pages means no reorder key set, so pages surface in API listing
			// order — which is exactly the platform's fixed creation order.
			state := &AssetResourceModel{
				GroupID: types.StringValue("g"),
				AssetID: types.StringValue("a"),
				Version: types.StringValue("1.0.0"),
				Pages:   types.ListNull(pageObjectType()),
			}
			res.readPagesIntoState(context.Background(), state)

			if state.Pages.IsNull() || state.Pages.IsUnknown() {
				t.Fatalf("readPagesIntoState left Pages null/unknown; want a known list")
			}
			var got []PageModel
			if diags := state.Pages.ElementsAs(context.Background(), &got, false); diags.HasError() {
				t.Fatalf("ElementsAs errors: %v", diags.Errors())
			}
			if len(got) != len(tt.wantNames) {
				t.Fatalf("page count = %d, want %d (got=%+v)", len(got), len(tt.wantNames), got)
			}
			for i, want := range tt.wantNames {
				if got[i].PageName.ValueString() != want {
					t.Errorf("page[%d].page_name = %q, want %q", i, got[i].PageName.ValueString(), want)
				}
				// Content must be read back verbatim for every surfaced page.
				if wantContent, ok := tt.content[got[i].PagePath.ValueString()]; ok {
					if got[i].Content.ValueString() != wantContent {
						t.Errorf("page[%d] (%s) content = %q, want %q", i, want, got[i].Content.ValueString(), wantContent)
					}
				}
			}
		})
	}
}

// mvFake is a stateful, multi-version-aware fake Exchange server backing
// TestAssetResource_MultiVersion_ForEach_ImportRoundTrip. Unlike the single-version
// fake in TestAssetResource_Lifecycle_Hermetic, it stores a MAP of versions for ONE
// (group, asset) pair so the test can prove the three properties the `for_each`-over-
// versions design (#75/#122) actually rests on:
//
//	(1) VERSION scope is independent — each version keeps its own file classifier,
//	    mainFile, tags, and status. This is why N map entries == N genuinely distinct
//	    objects and the import of v2 must NOT bleed v1's metadata (or vice versa).
//	(2) GROUP scope is shared — name/description live once per (group, asset), so the
//	    PATCH endpoint mutates BOTH versions' group metadata at once. The fake models
//	    this by storing name/description on the parent, not per version.
//	(3) DELETE is version-scoped — removing one map key hard-deletes ONLY that GAV; the
//	    sibling version survives and still reads back cleanly.
//
// The subject is type="rest-api", classifier="oas" — the file-backed path the example
// (examples/exchange/multi_version.tf) actually ships — so the fake also exercises the
// files[] → classifier/main_file extraction that the import Read path depends on to
// seed state without forcing a spurious replacement. Files are written to a temp dir so
// the real multipart upload (buildAssetMultipart → os.ReadFile) runs end to end.
type mvFake struct {
	mu sync.Mutex
	// Group-scoped (shared across ALL versions of this asset).
	name        string
	description string
	// Version-scoped store, keyed by version string.
	versions map[string]*mvVersion
	// Audit trail.
	deletedVersions []string
	deleteTypes     map[string]string
}

type mvVersion struct {
	classifier string // as the API stores it — bundled "fat-oas" for oas uploads
	mainFile   string
	status     string
	tags       []string
	posted     bool
}

// TestAssetResource_MultiVersion_ForEach_ImportRoundTrip is the end-to-end hermetic
// proof for multi-version version handling. It closes the coverage gap that the
// single-version TestAssetResource_Lifecycle_Hermetic left open: the `for_each` lifecycle
// (two versions of one asset side by side) and the per-version import round-trip that
// underpins `terraform import` / `-generate-config-out`.
//
// It runs, in order:
//
//	CREATE v1 (1.0.0, oas, tags=[a,v1], published)   — like for_each publishing key "v1"
//	CREATE v2 (2.0.0, oas, tags=[a,v2], published)   — like for_each publishing key "v2"
//	  → also asserts CREATE with status="deprecated" is REJECTED (create accepts only
//	    development/published — the #68 create/update asymmetry).
//	UPDATE v2 status published → deprecated              — via the real PUT /status path
//	  → the only faithful route to "deprecated"; proves status is version-scoped on update.
//	  → asserts the two GAVs coexist and each settles to ITS OWN version-scoped state,
//	    while sharing the one group-scoped name/description.
//	IMPORT v2 by "g/a/2.0.0", then READ                — the round-trip that feeds generated config
//	  → asserts the imported+read state reproduces v2's classifier/main_file/tags/status
//	    (NOT v1's) and the shared group name/description, with ZERO drift on the immutable
//	    identity (group_id/asset_id/version/id/type). This is exactly what a user would get
//	    from `terraform import anypoint_exchange_asset.petstore[\"v2\"] g/a/2.0.0` followed
//	    by a plan, so "does the generated config match" reduces to "does this state match".
//	DELETE v1                                          — like removing the "v1" map key
//	  → asserts a version-scoped hard delete of ONLY 1.0.0; v2 still reads back intact.
func TestAssetResource_MultiVersion_ForEach_ImportRoundTrip(t *testing.T) {
	const (
		orgID   = "test-org-id"
		groupID = "g"
		assetID = "a"
		v1      = "1.0.0"
		v2      = "2.0.0"
		// Group-scoped: identical for every version (the example factors these into
		// locals precisely so they cannot drift apart — caveat #2 in multi_version.tf).
		groupName = "TF Demo Petstore API (multi-version)"
		groupDesc = "Petstore API managed across versions with for_each."
	)

	fa := &mvFake{
		name:        groupName,
		description: groupDesc,
		versions:    map[string]*mvVersion{},
		deleteTypes: map[string]string{},
	}

	// Two genuinely different local spec files so the multipart upload + files[]
	// extraction is real, not stubbed. main_file differs per version to prove the
	// import read surfaces the RIGHT version's file metadata.
	tmp := t.TempDir()
	v1File := tmp + "/petstore.json"
	v2File := tmp + "/petstore-v2.json"
	if err := os.WriteFile(v1File, []byte(`{"openapi":"3.0.0","info":{"title":"petstore","version":"1.0.0"}}`), 0o600); err != nil {
		t.Fatalf("write v1 spec: %v", err)
	}
	if err := os.WriteFile(v2File, []byte(`{"openapi":"3.0.0","info":{"title":"petstore","version":"2.0.0"},"paths":{"/vaccinations":{}}}`), 0o600); err != nil {
		t.Fatalf("write v2 spec: %v", err)
	}

	// assetJSON renders the GET body for one stored version. The files[] array carries
	// the API-bundled "fat-oas" classifier so the Read path's normalizeClassifier /
	// apiClassifierToUserClassifier ("fat-" strip) is exercised on the import branch.
	assetJSON := func(version string, vv *mvVersion) map[string]interface{} {
		return map[string]interface{}{
			"groupId":      groupID,
			"assetId":      assetID,
			"version":      version,
			"name":         fa.name,        // group-scoped
			"description":  fa.description, // group-scoped
			"type":         "rest-api",
			"status":       vv.status, // version-scoped
			"isPublic":     false,
			"isSnapshot":   false,
			"minorVersion": version[:strings.LastIndex(version, ".")],
			"versionGroup": version[:strings.Index(version, ".")], // major → shared instance scope
			"createdDate":  "2024-01-01T00:00:00Z",
			"updatedDate":  "2024-01-01T00:00:00Z",
			"contactName":  nil,
			"contactEmail": nil,
			"manager":      nil,
			"labels":       toIfaceSlice(vv.tags), // version-scoped
			"categories":   []interface{}{},
			"customFields": []interface{}{},
			"files": []interface{}{
				// user-uploaded file — the one extractFileMetadata should pick up.
				map[string]interface{}{
					"classifier":  vv.classifier,
					"packaging":   "json",
					"mainFile":    vv.mainFile,
					"isGenerated": false,
				},
				// an auto-generated file that extractFileMetadata must SKIP.
				map[string]interface{}{
					"classifier":  "fat-oas",
					"packaging":   "json",
					"mainFile":    vv.mainFile,
					"isGenerated": true,
				},
			},
			"dependencies": []interface{}{},
			"instances":    []interface{}{},
			"attributes":   []interface{}{},
		}
	}

	base := "/exchange/api/v2"
	// Per-version POST publish endpoints (org-scoped).
	postV1 := base + "/organizations/" + orgID + "/assets/" + groupID + "/" + assetID + "/" + v1
	postV2 := base + "/organizations/" + orgID + "/assets/" + groupID + "/" + assetID + "/" + v2
	// A throwaway publish endpoint used ONLY to prove the create-side status guard fires
	// (a real route so the 400 comes from the status check, not a missing handler / 404).
	postReject := base + "/organizations/" + orgID + "/assets/" + groupID + "/" + assetID + "/9.9.9"
	// Per-version GET/DELETE + tags + pages endpoints.
	verV1 := base + "/assets/" + groupID + "/" + assetID + "/" + v1
	verV2 := base + "/assets/" + groupID + "/" + assetID + "/" + v2
	metaPath := base + "/assets/" + groupID + "/" + assetID // group-scoped PATCH

	// publishHandler records a multipart publish for the given version.
	publishHandler := func(version string) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want POST")
				return
			}
			if err := r.ParseMultipartForm(10 << 20); err != nil {
				testutil.ErrorResponse(w, http.StatusBadRequest, "bad multipart: "+err.Error())
				return
			}
			fa.mu.Lock()
			defer fa.mu.Unlock()
			// The multipart type must be rest-api for BOTH versions (same asset type).
			if got := r.FormValue("type"); got != "rest-api" {
				testutil.ErrorResponse(w, http.StatusBadRequest, "unexpected type: "+got)
				return
			}
			// Live create/update asymmetry (#68, version-scoping-live-proof.md §4):
			// multipart CREATE accepts only {development, published}. "deprecated" 400s at
			// create and is reachable ONLY via the PUT /status update path. Model that
			// faithfully so the test cannot silently rely on an impossible create.
			if st := r.FormValue("status"); st != "" && st != "development" && st != "published" {
				testutil.ErrorResponse(w, http.StatusBadRequest,
					"INVALID_ASSET_STATUS: "+st+" doesn't have a valid value (development, published).")
				return
			}
			// Discover the uploaded file field name: files.oas.json (from writeAssetFilePart).
			// Its presence proves the version-scoped file upload actually happened.
			mainFile := ""
			if r.MultipartForm != nil {
				for field, fhs := range r.MultipartForm.File {
					if strings.HasPrefix(field, "files.oas.") && len(fhs) > 0 {
						mainFile = fhs[0].Filename
					}
				}
			}
			vv := &mvVersion{
				// API bundles an oas upload as "fat-oas"; the Read path must strip it.
				classifier: "fat-oas",
				mainFile:   mainFile,
				status:     r.FormValue("status"),
				posted:     true,
			}
			fa.versions[version] = vv
			// Group metadata is shared: a publish may (re)assert name/description.
			if n := r.FormValue("name"); n != "" {
				fa.name = n
			}
			if d := r.FormValue("description"); d != "" {
				fa.description = d
			}
			w.WriteHeader(http.StatusCreated)
		}
	}

	// versionHandler serves GET (readback) + DELETE (version-scoped hard delete) + the
	// tags sub-path (PUT) for one version.
	versionHandler := func(version string) func(http.ResponseWriter, *http.Request) {
		return func(w http.ResponseWriter, r *http.Request) {
			// tags sub-path: /assets/g/a/{version}/tags
			if strings.HasSuffix(r.URL.Path, "/tags") {
				if r.Method != http.MethodPut {
					testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want PUT for tags")
					return
				}
				var tagReqs []map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&tagReqs); err != nil {
					testutil.ErrorResponse(w, http.StatusBadRequest, "bad tags json: "+err.Error())
					return
				}
				fa.mu.Lock()
				if vv := fa.versions[version]; vv != nil {
					vv.tags = vv.tags[:0]
					for _, tr := range tagReqs {
						if val, ok := tr["value"].(string); ok {
							vv.tags = append(vv.tags, val)
						}
					}
				}
				fa.mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// status sub-path: /assets/g/a/{version}/status. This is the ONLY route to
			// "deprecated" — the create multipart rejects it (see publishHandler). The PUT
			// mutates ONLY this version's status, proving status is version-scoped on the
			// UPDATE path too, and it completes the create/update asymmetry by REJECTING
			// "development" on update (#68 / version-scoping-live-proof.md §4: PUT /status
			// accepts only {published, deprecated}).
			if strings.HasSuffix(r.URL.Path, "/status") {
				if r.Method != http.MethodPut {
					testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want PUT for status")
					return
				}
				var statusReq map[string]interface{}
				if err := json.NewDecoder(r.Body).Decode(&statusReq); err != nil {
					testutil.ErrorResponse(w, http.StatusBadRequest, "bad status json: "+err.Error())
					return
				}
				newStatus, _ := statusReq["status"].(string)
				if newStatus != "published" && newStatus != "deprecated" {
					testutil.ErrorResponse(w, http.StatusBadRequest,
						"INVALID_ASSET_STATUS: "+newStatus+" not allowed on update (published, deprecated).")
					return
				}
				fa.mu.Lock()
				if vv := fa.versions[version]; vv != nil {
					vv.status = newStatus
				}
				fa.mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
				return
			}
			// pages sub-path: keep readPagesIntoState on its clean, empty path.
			if strings.HasSuffix(r.URL.Path, "/portal/pages") {
				testutil.JSONResponse(w, http.StatusOK, []interface{}{})
				return
			}
			switch r.Method {
			case http.MethodGet:
				fa.mu.Lock()
				vv := fa.versions[version]
				var body map[string]interface{}
				if vv != nil {
					body = assetJSON(version, vv)
				}
				fa.mu.Unlock()
				if vv == nil {
					testutil.ErrorResponse(w, http.StatusNotFound, "version not found")
					return
				}
				testutil.JSONResponse(w, http.StatusOK, body)
			case http.MethodDelete:
				fa.mu.Lock()
				delete(fa.versions, version)
				fa.deletedVersions = append(fa.deletedVersions, version)
				fa.deleteTypes[version] = r.Header.Get("x-delete-type")
				fa.mu.Unlock()
				w.WriteHeader(http.StatusNoContent)
			default:
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want GET/DELETE")
			}
		}
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		postV1:     publishHandler(v1),
		postV2:     publishHandler(v2),
		postReject: publishHandler("9.9.9"),
		// A ServeMux longest-prefix match sends /tags and /portal/pages here too.
		verV1 + "/": versionHandler(v1),
		verV2 + "/": versionHandler(v2),
		verV1:       versionHandler(v1),
		verV2:       versionHandler(v2),
		metaPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPatch {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "want PATCH")
				return
			}
			var body map[string]interface{}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				testutil.ErrorResponse(w, http.StatusBadRequest, "bad json: "+err.Error())
				return
			}
			fa.mu.Lock()
			if v, ok := body["name"].(string); ok {
				fa.name = v // group-scoped: mutates the parent, affecting BOTH versions
			}
			if v, ok := body["description"].(string); ok {
				fa.description = v
			}
			fa.mu.Unlock()
			w.WriteHeader(http.StatusNoContent)
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

	// planRawFor builds the config the way for_each would render ONE map entry: all
	// VERSION-scoped fields from the entry, the GROUP-scoped name/description shared.
	planRawFor := func(version, filePath, status string, tags []string) tftypes.Value {
		tagVals := make([]tftypes.Value, len(tags))
		for i, tg := range tags {
			tagVals[i] = tftypes.NewValue(tftypes.String, tg)
		}
		return assetRawValue(ctx, schemaResp.Schema, map[string]tftypes.Value{
			"organization_id": tftypes.NewValue(tftypes.String, orgID),
			"group_id":        tftypes.NewValue(tftypes.String, groupID),
			"asset_id":        tftypes.NewValue(tftypes.String, assetID),
			"version":         tftypes.NewValue(tftypes.String, version),
			"type":            tftypes.NewValue(tftypes.String, "rest-api"),
			"classifier":      tftypes.NewValue(tftypes.String, "oas"),
			"file_path":       tftypes.NewValue(tftypes.String, filePath),
			"main_file":       tftypes.NewValue(tftypes.String, filePathBase(filePath)),
			"name":            tftypes.NewValue(tftypes.String, groupName),
			"description":     tftypes.NewValue(tftypes.String, groupDesc),
			"status":          tftypes.NewValue(tftypes.String, status),
			"tags": tftypes.NewValue(
				tftypes.List{ElementType: tftypes.String}, tagVals,
			),
		})
	}

	// createVersion runs one Create the way for_each would for a single map entry.
	createVersion := func(version, filePath, status string, tags []string) AssetResourceModel {
		t.Helper()
		planRaw := planRawFor(version, filePath, status, tags)
		createReq := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw}}
		createResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		res.Create(ctx, createReq, createResp)
		if createResp.Diagnostics.HasError() {
			t.Fatalf("Create(%s) errors: %v", version, createResp.Diagnostics.Errors())
		}
		var m AssetResourceModel
		if diags := createResp.State.Get(ctx, &m); diags.HasError() {
			t.Fatalf("Create(%s) State.Get errors: %v", version, diags.Errors())
		}
		return m
	}

	// updateStatusVersion drives the REAL Update path (asset.go Update → client.UpdateStatus
	// → PUT /status) to transition one version's status. This is the only faithful way to
	// reach "deprecated": the create multipart rejects it (create/update asymmetry, #68).
	// Only `status` differs from prior state, so Update fires UpdateStatus and nothing else.
	updateStatusVersion := func(prior AssetResourceModel, version, filePath, newStatus string, tags []string) AssetResourceModel {
		t.Helper()
		planRaw := planRawFor(version, filePath, newStatus, tags)
		priorState := tfsdk.State{Schema: schemaResp.Schema}
		if diags := priorState.Set(ctx, &prior); diags.HasError() {
			t.Fatalf("seed update state(%s): %v", version, diags.Errors())
		}
		updateReq := resource.UpdateRequest{
			Plan:  tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
			State: priorState,
		}
		updateResp := &resource.UpdateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
		res.Update(ctx, updateReq, updateResp)
		if updateResp.Diagnostics.HasError() {
			t.Fatalf("Update(%s → %s) errors: %v", version, newStatus, updateResp.Diagnostics.Errors())
		}
		var m AssetResourceModel
		if diags := updateResp.State.Get(ctx, &m); diags.HasError() {
			t.Fatalf("Update(%s) State.Get errors: %v", version, diags.Errors())
		}
		return m
	}

	// --- CREATE both versions (for_each over {v1, v2}) ------------------------
	// Both are created "published" — the ONLY multi-version-friendly create status the
	// platform accepts (development/published). v2 is then transitioned to "deprecated"
	// via the update path below, exactly as a user would flip a version's status.
	created1 := createVersion(v1, v1File, "published", []string{"petstore", "v1"})
	created2Pub := createVersion(v2, v2File, "published", []string{"petstore", "v2"})
	// Guard: "deprecated" at CREATE must be rejected by the platform (and our fake). This
	// asserts the create-side of the asymmetry so the test can't silently regress into
	// relying on an impossible create.
	rejectReq := resource.CreateRequest{Plan: tfsdk.Plan{Schema: schemaResp.Schema,
		Raw: planRawFor("9.9.9", v2File, "deprecated", []string{"nope"})}}
	rejectResp := &resource.CreateResponse{State: tfsdk.State{Schema: schemaResp.Schema}}
	res.Create(ctx, rejectReq, rejectResp)
	if !rejectResp.Diagnostics.HasError() {
		t.Fatal("Create with status=deprecated must fail (create accepts only development/published)")
	}
	// Now flip v2 → deprecated through the real Update → PUT /status path.
	created2 := updateStatusVersion(created2Pub, v2, v2File, "deprecated", []string{"petstore", "v2"})

	// Both GAVs must coexist in the fake — additive, NOT destroy-and-recreate.
	fa.mu.Lock()
	if len(fa.versions) != 2 || fa.versions[v1] == nil || fa.versions[v2] == nil {
		t.Fatalf("expected both versions to coexist, got %d: %v", len(fa.versions), fa.versions)
	}
	if len(fa.deletedVersions) != 0 {
		t.Fatalf("Create must never delete a version; deleted=%v", fa.deletedVersions)
	}
	fa.mu.Unlock()

	// Distinct composite IDs.
	if created1.ID.ValueString() != "g/a/1.0.0" {
		t.Errorf("v1 ID = %q, want g/a/1.0.0", created1.ID.ValueString())
	}
	if created2.ID.ValueString() != "g/a/2.0.0" {
		t.Errorf("v2 ID = %q, want g/a/2.0.0", created2.ID.ValueString())
	}
	// VERSION-scoped fields must be independent per version.
	if created1.Status.ValueString() != "published" {
		t.Errorf("v1 status = %q, want published", created1.Status.ValueString())
	}
	if created2.Status.ValueString() != "deprecated" {
		t.Errorf("v2 status = %q, want deprecated", created2.Status.ValueString())
	}
	if got := tagStrings(t, ctx, created1.Tags); !equalStringSets(got, []string{"petstore", "v1"}) {
		t.Errorf("v1 tags = %v, want [petstore v1]", got)
	}
	if got := tagStrings(t, ctx, created2.Tags); !equalStringSets(got, []string{"petstore", "v2"}) {
		t.Errorf("v2 tags = %v, want [petstore v2]", got)
	}
	// classifier round-trips back to the user-facing "oas" (API stored "fat-oas").
	if created1.Classifier.ValueString() != "oas" {
		t.Errorf("v1 classifier = %q, want oas (fat- must be stripped)", created1.Classifier.ValueString())
	}
	// main_file must be the version's own uploaded file, independent per version.
	if created1.MainFile.ValueString() != "petstore.json" {
		t.Errorf("v1 main_file = %q, want petstore.json", created1.MainFile.ValueString())
	}
	if created2.MainFile.ValueString() != "petstore-v2.json" {
		t.Errorf("v2 main_file = %q, want petstore-v2.json", created2.MainFile.ValueString())
	}
	// GROUP-scoped fields must be identical across versions.
	if created1.Name.ValueString() != groupName || created2.Name.ValueString() != groupName {
		t.Errorf("group name diverged: v1=%q v2=%q", created1.Name.ValueString(), created2.Name.ValueString())
	}
	if created1.Description.ValueString() != groupDesc || created2.Description.ValueString() != groupDesc {
		t.Errorf("group description diverged: v1=%q v2=%q", created1.Description.ValueString(), created2.Description.ValueString())
	}

	// --- IMPORT v2, then READ (the generated-config round-trip) ---------------
	// This is the crux: `terraform import ...["v2"] g/a/2.0.0` seeds identity, then a
	// Read fills the rest. The resulting state is exactly what -generate-config-out
	// would render, so it MUST reflect v2 — never v1 — for every version-scoped field.
	importReq := resource.ImportStateRequest{ID: "g/a/2.0.0"}
	importResp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: assetRawValue(ctx, schemaResp.Schema, nil)}}
	res.ImportState(ctx, importReq, importResp)
	if importResp.Diagnostics.HasError() {
		t.Fatalf("ImportState(v2) errors: %v", importResp.Diagnostics.Errors())
	}

	readReq := resource.ReadRequest{State: importResp.State}
	readResp := &resource.ReadResponse{State: importResp.State}
	res.Read(ctx, readReq, readResp)
	if readResp.Diagnostics.HasError() {
		t.Fatalf("Read after import(v2) errors: %v", readResp.Diagnostics.Errors())
	}
	if readResp.State.Raw.IsNull() {
		t.Fatal("Read after import removed the resource from state")
	}
	var imported AssetResourceModel
	if diags := readResp.State.Get(ctx, &imported); diags.HasError() {
		t.Fatalf("imported State.Get errors: %v", diags.Errors())
	}

	// Identity: the imported GAV is v2, not v1.
	if imported.Version.ValueString() != v2 {
		t.Errorf("import: version = %q, want %q", imported.Version.ValueString(), v2)
	}
	if imported.ID.ValueString() != "g/a/2.0.0" {
		t.Errorf("import: id = %q, want g/a/2.0.0", imported.ID.ValueString())
	}
	if imported.GroupID.ValueString() != groupID || imported.AssetID.ValueString() != assetID {
		t.Errorf("import: group/asset = %q/%q, want g/a", imported.GroupID.ValueString(), imported.AssetID.ValueString())
	}
	if imported.Type.ValueString() != "rest-api" {
		t.Errorf("import: type = %q, want rest-api", imported.Type.ValueString())
	}
	// VERSION-scoped fields reflect v2 specifically (the bleed-guard: NOT v1's values).
	if imported.Status.ValueString() != "deprecated" {
		t.Errorf("import: status = %q, want deprecated (v2's own status, not v1's published)", imported.Status.ValueString())
	}
	if imported.MainFile.ValueString() != "petstore-v2.json" {
		t.Errorf("import: main_file = %q, want petstore-v2.json (v2's file, not v1's petstore.json)", imported.MainFile.ValueString())
	}
	if imported.Classifier.ValueString() != "oas" {
		t.Errorf("import: classifier = %q, want oas (fat- stripped on import)", imported.Classifier.ValueString())
	}
	if got := tagStrings(t, ctx, imported.Tags); !equalStringSets(got, []string{"petstore", "v2"}) {
		t.Errorf("import: tags = %v, want [petstore v2] (v2's own tags)", got)
	}
	// GROUP-scoped fields present and shared.
	if imported.Name.ValueString() != groupName {
		t.Errorf("import: name = %q, want %q", imported.Name.ValueString(), groupName)
	}
	if imported.Description.ValueString() != groupDesc {
		t.Errorf("import: description = %q, want %q", imported.Description.ValueString(), groupDesc)
	}

	// --- DELETE v1 only (removing the "v1" map key) ---------------------------
	// Uses created1's state, exactly as Terraform would when the "v1" entry is dropped.
	delState := tfsdk.State{Schema: schemaResp.Schema}
	if diags := delState.Set(ctx, &created1); diags.HasError() {
		t.Fatalf("seed delete state: %v", diags.Errors())
	}
	deleteReq := resource.DeleteRequest{State: delState}
	deleteResp := &resource.DeleteResponse{State: delState}
	res.Delete(ctx, deleteReq, deleteResp)
	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete(v1) errors: %v", deleteResp.Diagnostics.Errors())
	}

	fa.mu.Lock()
	defer fa.mu.Unlock()
	// ONLY v1 was deleted, and as a HARD delete (tombstone-free, reusable GAV).
	if len(fa.deletedVersions) != 1 || fa.deletedVersions[0] != v1 {
		t.Errorf("delete: deletedVersions = %v, want [1.0.0] only", fa.deletedVersions)
	}
	if fa.deleteTypes[v1] != "hard-delete" {
		t.Errorf("delete: v1 x-delete-type = %q, want hard-delete", fa.deleteTypes[v1])
	}
	// v2 MUST survive — version-scoped delete never touches the sibling.
	if fa.versions[v2] == nil {
		t.Error("delete(v1) collaterally removed v2; version-scoped delete must leave siblings intact")
	}
	if fa.versions[v1] != nil {
		t.Error("delete(v1) did not remove v1 from the store")
	}
}

// --- helpers for the multi-version test ---

// filePathBase returns the trailing file name of a path (avoids importing path/filepath
// solely for basename in the test; the resource itself uses filepath.Base at runtime).
func filePathBase(p string) string {
	if i := strings.LastIndexByte(p, '/'); i >= 0 {
		return p[i+1:]
	}
	return p
}

// toIfaceSlice converts a []string to []interface{} for JSON body construction.
func toIfaceSlice(ss []string) []interface{} {
	out := make([]interface{}, len(ss))
	for i, s := range ss {
		out[i] = s
	}
	return out
}

// tagStrings extracts a []string from a types.List of strings.
func tagStrings(t *testing.T, ctx context.Context, l types.List) []string {
	t.Helper()
	if l.IsNull() || l.IsUnknown() {
		return nil
	}
	var out []string
	if diags := l.ElementsAs(ctx, &out, false); diags.HasError() {
		t.Fatalf("tagStrings ElementsAs: %v", diags.Errors())
	}
	return out
}

// equalStringSets compares two string slices as unordered sets (tags carry
// UseStateForUnknown + reorder, so order is not part of the contract here).
func equalStringSets(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	m := make(map[string]int, len(a))
	for _, s := range a {
		m[s]++
	}
	for _, s := range b {
		m[s]--
		if m[s] < 0 {
			return false
		}
	}
	return true
}
