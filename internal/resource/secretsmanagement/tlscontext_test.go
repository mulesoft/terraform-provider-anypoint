package secretsmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	secretsmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTLSContextResource(t *testing.T) {
	r := NewTLSContextResource()

	if r == nil {
		t.Error("NewTLSContextResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("TLSContextResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("TLSContextResource should implement ResourceWithImportState")
	}
}

func TestTLSContextResource_Metadata(t *testing.T) {
	r := NewTLSContextResource()
	testutil.TestResourceMetadata(t, r, "_flex_tls_context")
}

func TestTLSContextResource_Schema(t *testing.T) {
	res := NewTLSContextResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id"}
	optionalAttrs := []string{"organization_id", "keystore_id", "truststore_id", "min_tls_version", "max_tls_version", "alpn_protocols", "cipher_suites", "enable_client_cert_validation", "skip_server_cert_validation"}
	computedAttrs := []string{"id", "target", "min_tls_version", "max_tls_version", "enable_client_cert_validation", "skip_server_cert_validation", "organization_id", "expiration_date"}

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
	r := NewTLSContextResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTLSContextResourceModel_Validation(t *testing.T) {
	model := TLSContextResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.Target
	_ = model.KeystoreID
	_ = model.TruststoreID
	_ = model.MinTLSVersion
	_ = model.MaxTLSVersion
	_ = model.AlpnProtocols
	_ = model.CipherSuites
	_ = model.EnableClientCertValidation
	_ = model.SkipServerCertValidation
	_ = model.ExpirationDate
}

func TestTLSContextResource_Read(t *testing.T) {
	mockTLS := &secretsmgmt.TLSContextResponse{
		Name:   "test-tls",
		Target: "Mule",
		Meta:   secretsmgmt.SecretGroupMeta{ID: "tls-id-1"},
	}

	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/tlsContexts/tls-id-1"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				testutil.JSONResponse(w, http.StatusOK, mockTLS)
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTLSContextResource().(*TLSContextResource)
	res.client = &secretsmgmt.TLSContextClient{
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
	alpnElemType := objType.AttributeTypes["alpn_protocols"].(tftypes.List).ElementType
	cipherElemType := objType.AttributeTypes["cipher_suites"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                            tftypes.NewValue(tftypes.String, "tls-id-1"),
		"organization_id":               tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":                tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":               tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":                          tftypes.NewValue(tftypes.String, "old-name"),
		"target":                        tftypes.NewValue(tftypes.String, ""),
		"keystore_id":                   tftypes.NewValue(tftypes.String, nil),
		"truststore_id":                 tftypes.NewValue(tftypes.String, nil),
		"min_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"max_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"alpn_protocols":                tftypes.NewValue(tftypes.List{ElementType: alpnElemType}, nil),
		"cipher_suites":                 tftypes.NewValue(tftypes.List{ElementType: cipherElemType}, nil),
		"enable_client_cert_validation": tftypes.NewValue(tftypes.Bool, false),
		"skip_server_cert_validation":   tftypes.NewValue(tftypes.Bool, false),
		"expiration_date":               tftypes.NewValue(tftypes.String, ""),
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
		t.Errorf("Expected Name test-tls, got %s", got.Name.ValueString())
	}
}

func TestTLSContextResource_Read_NotFound(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/tlsContexts/missing-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTLSContextResource().(*TLSContextResource)
	res.client = &secretsmgmt.TLSContextClient{
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
	alpnElemType := objType.AttributeTypes["alpn_protocols"].(tftypes.List).ElementType
	cipherElemType := objType.AttributeTypes["cipher_suites"].(tftypes.List).ElementType

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                            tftypes.NewValue(tftypes.String, "missing-id"),
		"organization_id":               tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":                tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":               tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":                          tftypes.NewValue(tftypes.String, "tls"),
		"target":                        tftypes.NewValue(tftypes.String, ""),
		"keystore_id":                   tftypes.NewValue(tftypes.String, nil),
		"truststore_id":                 tftypes.NewValue(tftypes.String, nil),
		"min_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"max_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"alpn_protocols":                tftypes.NewValue(tftypes.List{ElementType: alpnElemType}, nil),
		"cipher_suites":                 tftypes.NewValue(tftypes.List{ElementType: cipherElemType}, nil),
		"enable_client_cert_validation": tftypes.NewValue(tftypes.Bool, false),
		"skip_server_cert_validation":   tftypes.NewValue(tftypes.Bool, false),
		"expiration_date":               tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() on not-found should remove resource: %v", resp.Diagnostics.Errors())
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Read() on not-found should remove resource from state")
	}
}

func TestTLSContextResource_ImportState_Valid(t *testing.T) {
	res := NewTLSContextResource().(*TLSContextResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	alpnElemType := objType.AttributeTypes["alpn_protocols"].(tftypes.List).ElementType
	cipherElemType := objType.AttributeTypes["cipher_suites"].(tftypes.List).ElementType

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                            tftypes.NewValue(tftypes.String, nil),
		"organization_id":               tftypes.NewValue(tftypes.String, nil),
		"environment_id":                tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":               tftypes.NewValue(tftypes.String, nil),
		"name":                          tftypes.NewValue(tftypes.String, nil),
		"target":                        tftypes.NewValue(tftypes.String, nil),
		"keystore_id":                   tftypes.NewValue(tftypes.String, nil),
		"truststore_id":                 tftypes.NewValue(tftypes.String, nil),
		"min_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"max_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"alpn_protocols":                tftypes.NewValue(tftypes.List{ElementType: alpnElemType}, nil),
		"cipher_suites":                 tftypes.NewValue(tftypes.List{ElementType: cipherElemType}, nil),
		"enable_client_cert_validation": tftypes.NewValue(tftypes.Bool, nil),
		"skip_server_cert_validation":   tftypes.NewValue(tftypes.Bool, nil),
		"expiration_date":               tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ImportStateRequest{ID: "org-id/env-id/sg-id/tls-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestTLSContextResource_ImportState_Invalid(t *testing.T) {
	res := NewTLSContextResource().(*TLSContextResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	alpnElemType := objType.AttributeTypes["alpn_protocols"].(tftypes.List).ElementType
	cipherElemType := objType.AttributeTypes["cipher_suites"].(tftypes.List).ElementType

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                            tftypes.NewValue(tftypes.String, nil),
		"organization_id":               tftypes.NewValue(tftypes.String, nil),
		"environment_id":                tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":               tftypes.NewValue(tftypes.String, nil),
		"name":                          tftypes.NewValue(tftypes.String, nil),
		"target":                        tftypes.NewValue(tftypes.String, nil),
		"keystore_id":                   tftypes.NewValue(tftypes.String, nil),
		"truststore_id":                 tftypes.NewValue(tftypes.String, nil),
		"min_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"max_tls_version":               tftypes.NewValue(tftypes.String, nil),
		"alpn_protocols":                tftypes.NewValue(tftypes.List{ElementType: alpnElemType}, nil),
		"cipher_suites":                 tftypes.NewValue(tftypes.List{ElementType: cipherElemType}, nil),
		"enable_client_cert_validation": tftypes.NewValue(tftypes.Bool, nil),
		"skip_server_cert_validation":   tftypes.NewValue(tftypes.Bool, nil),
		"expiration_date":               tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ImportStateRequest{ID: "invalid/short"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ImportState() with invalid ID should produce errors")
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
