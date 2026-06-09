package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTLSContextResource(t *testing.T) {
	r := NewTLSContextResource()

	if r == nil {
		t.Error("NewTLSContextResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestTLSContextResource_Metadata(t *testing.T) {
	r := NewTLSContextResource()
	testutil.TestResourceMetadata(t, r, "_tls_context")
}

func TestTLSContextResource_Schema(t *testing.T) {
	res := NewTLSContextResource()

	requiredAttrs := []string{"private_space_id", "name", "ciphers"}
	optionalAttrs := []string{"organization_id", "keystore_type"}
	computedAttrs := []string{"id", "type", "keystore_type"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTLSContextResource_Configure(t *testing.T) {
	res := NewTLSContextResource().(*TLSContextResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestTLSContextResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTLSContextResource().(*TLSContextResource)

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

func TestTLSContextResource_ImportState(t *testing.T) {
	res := NewTLSContextResource()

	ctx := context.Background()

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{
		ID: "test-space:test-tls",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(stateType, nil),
		},
	}

	if importableResource, ok := res.(resource.ResourceWithImportState); ok {
		importableResource.ImportState(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("ImportState() has errors: %v", resp.Diagnostics.Errors())
		}
	} else {
		t.Error("Resource does not implement ResourceWithImportState")
	}
}

func TestTLSContextResource_ImportState_SimpleID(t *testing.T) {
	res := NewTLSContextResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{ID: "test-space-id:test-tls-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)},
	}

	res.(resource.ResourceWithImportState).ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with simple ID reported errors: %v", resp.Diagnostics.Errors())
	}

	var model TLSContextResourceModel
	resp.State.Get(ctx, &model)
	if model.ID.ValueString() != "test-tls-id" {
		t.Errorf("expected id = %q, got %q", "test-tls-id", model.ID.ValueString())
	}
	if model.PrivateSpaceID.ValueString() != "test-space-id" {
		t.Errorf("expected private_space_id = %q, got %q", "test-space-id", model.PrivateSpaceID.ValueString())
	}
	if !model.OrganizationID.IsNull() {
		t.Errorf("expected organization_id to be null for simple import, got %q", model.OrganizationID.ValueString())
	}
}

func TestTLSContextResource_ImportState_CompositeID(t *testing.T) {
	res := NewTLSContextResource()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{ID: "test-org-id:test-space-id:test-tls-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)},
	}

	res.(resource.ResourceWithImportState).ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() with composite ID reported errors: %v", resp.Diagnostics.Errors())
	}

	var model TLSContextResourceModel
	resp.State.Get(ctx, &model)
	if model.ID.ValueString() != "test-tls-id" {
		t.Errorf("expected id = %q, got %q", "test-tls-id", model.ID.ValueString())
	}
	if model.PrivateSpaceID.ValueString() != "test-space-id" {
		t.Errorf("expected private_space_id = %q, got %q", "test-space-id", model.PrivateSpaceID.ValueString())
	}
	if model.OrganizationID.ValueString() != "test-org-id" {
		t.Errorf("expected organization_id = %q, got %q", "test-org-id", model.OrganizationID.ValueString())
	}
}

func TestTLSContextResourceModel_Validation(t *testing.T) {
	model := TLSContextResourceModel{}
	_ = model.ID
}

func TestTLSContextResource_Read(t *testing.T) {
	// GetTLSContext uses the list endpoint — mock /tlsContexts (no ID suffix)
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/tlsContexts"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{
				{
					"id":   "test-tls-id",
					"name": "test-tls",
					"type": "PEM",
					"keyStore": map[string]interface{}{
						"type": "PEM",
						"cn":   "test.example.com",
						"san":  []string{},
					},
					"ciphers": map[string]interface{}{
						"aes128GcmSha256":            false,
						"aes128Sha256":               false,
						"aes256GcmSha384":            false,
						"aes256Sha256":               false,
						"dheRsaAes128Sha256":         false,
						"dheRsaAes256GcmSha384":      false,
						"dheRsaAes256Sha256":         false,
						"ecdheEcdsaAes128GcmSha256":  false,
						"ecdheEcdsaAes256GcmSha384":  false,
						"ecdheRsaAes128GcmSha256":    false,
						"ecdheRsaAes256GcmSha384":    false,
						"ecdheEcdsaChacha20Poly1305":  false,
						"ecdheRsaChacha20Poly1305":   false,
						"dheRsaChacha20Poly1305":     false,
						"tlsAes256GcmSha384":         false,
						"tlsChacha20Poly1305Sha256":  false,
						"tlsAes128GcmSha256":         false,
					},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTLSContextResource().(*TLSContextResource)
	res.client = &ch2client.TLSContextClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ciphersObjType := objType.AttributeTypes["ciphers"].(tftypes.Object)
	trustStoreObjType := objType.AttributeTypes["trust_store"].(tftypes.Object)
	keyStoreObjType := objType.AttributeTypes["key_store"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "test-tls-id"),
		"private_space_id":     tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":                 tftypes.NewValue(tftypes.String, "test-tls"),
		"keystore_type":        tftypes.NewValue(tftypes.String, "PEM"),
		"certificate":          tftypes.NewValue(tftypes.String, nil),
		"key":                  tftypes.NewValue(tftypes.String, nil),
		"key_filename":         tftypes.NewValue(tftypes.String, nil),
		"certificate_filename": tftypes.NewValue(tftypes.String, nil),
		"keystore_base64":      tftypes.NewValue(tftypes.String, nil),
		"store_passphrase":     tftypes.NewValue(tftypes.String, nil),
		"alias":                tftypes.NewValue(tftypes.String, nil),
		"keystore_filename":    tftypes.NewValue(tftypes.String, nil),
		"key_passphrase":       tftypes.NewValue(tftypes.String, nil),
		"ciphers":              tftypes.NewValue(ciphersObjType, nil),
		"type":                 tftypes.NewValue(tftypes.String, nil),
		"trust_store":          tftypes.NewValue(trustStoreObjType, nil),
		"key_store":            tftypes.NewValue(keyStoreObjType, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got TLSContextResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "test-tls" {
		t.Errorf("Expected Name 'test-tls', got %s", got.Name.ValueString())
	}
}

func TestTLSContextResource_Read_NotFound(t *testing.T) {
	// GetTLSContext uses the list endpoint; return an empty list to simulate not found
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/tlsContexts"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []map[string]interface{}{})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTLSContextResource().(*TLSContextResource)
	res.client = &ch2client.TLSContextClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	ciphersObjType := objType.AttributeTypes["ciphers"].(tftypes.Object)
	trustStoreObjType := objType.AttributeTypes["trust_store"].(tftypes.Object)
	keyStoreObjType := objType.AttributeTypes["key_store"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "test-tls-id"),
		"private_space_id":     tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":                 tftypes.NewValue(tftypes.String, "test-tls"),
		"keystore_type":        tftypes.NewValue(tftypes.String, "PEM"),
		"certificate":          tftypes.NewValue(tftypes.String, nil),
		"key":                  tftypes.NewValue(tftypes.String, nil),
		"key_filename":         tftypes.NewValue(tftypes.String, nil),
		"certificate_filename": tftypes.NewValue(tftypes.String, nil),
		"keystore_base64":      tftypes.NewValue(tftypes.String, nil),
		"store_passphrase":     tftypes.NewValue(tftypes.String, nil),
		"alias":                tftypes.NewValue(tftypes.String, nil),
		"keystore_filename":    tftypes.NewValue(tftypes.String, nil),
		"key_passphrase":       tftypes.NewValue(tftypes.String, nil),
		"ciphers":              tftypes.NewValue(ciphersObjType, nil),
		"type":                 tftypes.NewValue(tftypes.String, nil),
		"trust_store":          tftypes.NewValue(trustStoreObjType, nil),
		"key_store":            tftypes.NewValue(keyStoreObjType, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}

func BenchmarkTLSContextResource_Schema(b *testing.B) {
	res := NewTLSContextResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
