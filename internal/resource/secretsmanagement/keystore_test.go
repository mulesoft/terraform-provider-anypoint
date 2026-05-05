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
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewKeystoreResource(t *testing.T) {
	r := NewKeystoreResource()

	if r == nil {
		t.Error("NewKeystoreResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("KeystoreResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("KeystoreResource should implement ResourceWithImportState")
	}
}

func TestKeystoreResource_Metadata(t *testing.T) {
	r := NewKeystoreResource()
	testutil.TestResourceMetadata(t, r, "_secret_group_keystore")
}

func TestKeystoreResource_Schema(t *testing.T) {
	res := NewKeystoreResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id"}
	optionalAttrs := []string{"type", "organization_id", "certificate_base64", "key_base64", "keystore_file_base64", "store_passphrase", "key_passphrase", "alias", "ca_path_base64"}
	computedAttrs := []string{"id", "expiration_date", "algorithm", "type", "organization_id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestKeystoreResource_Configure(t *testing.T) {
	res := NewKeystoreResource().(*KeystoreResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestKeystoreResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewKeystoreResource().(*KeystoreResource)

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

func TestKeystoreResource_ImportState(t *testing.T) {
	r := NewKeystoreResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestKeystoreResourceModel_Validation(t *testing.T) {
	model := KeystoreResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.Type
	_ = model.CertificateB64
	_ = model.KeyB64
	_ = model.KeystoreFileB64
	_ = model.StorePassphrase
	_ = model.KeyPassphrase
	_ = model.Alias
	_ = model.CaPathB64
	_ = model.ExpirationDate
	_ = model.Algorithm
}

func TestKeystoreResource_Read(t *testing.T) {
	mockKS := &secretsmgmt.KeystoreResponse{
		Name: "test-keystore",
		Type: "PEM",
		Meta: secretsmgmt.SecretGroupMeta{ID: "ks-id-1"},
	}

	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/keystores/ks-id-1"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				testutil.JSONResponse(w, http.StatusOK, mockKS)
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewKeystoreResource().(*KeystoreResource)
	res.client = &secretsmgmt.KeystoreClient{
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "ks-id-1"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":       tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":      tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":                 tftypes.NewValue(tftypes.String, "old-name"),
		"type":                 tftypes.NewValue(tftypes.String, "PEM"),
		"certificate_base64":   tftypes.NewValue(tftypes.String, nil),
		"key_base64":           tftypes.NewValue(tftypes.String, nil),
		"keystore_file_base64": tftypes.NewValue(tftypes.String, nil),
		"store_passphrase":     tftypes.NewValue(tftypes.String, nil),
		"key_passphrase":       tftypes.NewValue(tftypes.String, nil),
		"alias":                tftypes.NewValue(tftypes.String, nil),
		"ca_path_base64":       tftypes.NewValue(tftypes.String, nil),
		"expiration_date":      tftypes.NewValue(tftypes.String, ""),
		"algorithm":            tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got KeystoreResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "test-keystore" {
		t.Errorf("Expected Name test-keystore, got %s", got.Name.ValueString())
	}
}

func TestKeystoreResource_Read_NotFound(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/keystores/missing-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewKeystoreResource().(*KeystoreResource)
	res.client = &secretsmgmt.KeystoreClient{
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, "missing-id"),
		"organization_id":      tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":       tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":      tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":                 tftypes.NewValue(tftypes.String, "ks"),
		"type":                 tftypes.NewValue(tftypes.String, "PEM"),
		"certificate_base64":   tftypes.NewValue(tftypes.String, nil),
		"key_base64":           tftypes.NewValue(tftypes.String, nil),
		"keystore_file_base64": tftypes.NewValue(tftypes.String, nil),
		"store_passphrase":     tftypes.NewValue(tftypes.String, nil),
		"key_passphrase":       tftypes.NewValue(tftypes.String, nil),
		"alias":                tftypes.NewValue(tftypes.String, nil),
		"ca_path_base64":       tftypes.NewValue(tftypes.String, nil),
		"expiration_date":      tftypes.NewValue(tftypes.String, ""),
		"algorithm":            tftypes.NewValue(tftypes.String, ""),
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

func TestKeystoreResource_ImportState_Valid(t *testing.T) {
	res := NewKeystoreResource().(*KeystoreResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, nil),
		"organization_id":      tftypes.NewValue(tftypes.String, nil),
		"environment_id":       tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":      tftypes.NewValue(tftypes.String, nil),
		"name":                 tftypes.NewValue(tftypes.String, nil),
		"type":                 tftypes.NewValue(tftypes.String, nil),
		"certificate_base64":   tftypes.NewValue(tftypes.String, nil),
		"key_base64":           tftypes.NewValue(tftypes.String, nil),
		"keystore_file_base64": tftypes.NewValue(tftypes.String, nil),
		"store_passphrase":     tftypes.NewValue(tftypes.String, nil),
		"key_passphrase":       tftypes.NewValue(tftypes.String, nil),
		"alias":                tftypes.NewValue(tftypes.String, nil),
		"ca_path_base64":       tftypes.NewValue(tftypes.String, nil),
		"expiration_date":      tftypes.NewValue(tftypes.String, nil),
		"algorithm":            tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ImportStateRequest{ID: "org-id/env-id/sg-id/ks-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestKeystoreResource_ImportState_Invalid(t *testing.T) {
	res := NewKeystoreResource().(*KeystoreResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                   tftypes.NewValue(tftypes.String, nil),
		"organization_id":      tftypes.NewValue(tftypes.String, nil),
		"environment_id":       tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":      tftypes.NewValue(tftypes.String, nil),
		"name":                 tftypes.NewValue(tftypes.String, nil),
		"type":                 tftypes.NewValue(tftypes.String, nil),
		"certificate_base64":   tftypes.NewValue(tftypes.String, nil),
		"key_base64":           tftypes.NewValue(tftypes.String, nil),
		"keystore_file_base64": tftypes.NewValue(tftypes.String, nil),
		"store_passphrase":     tftypes.NewValue(tftypes.String, nil),
		"key_passphrase":       tftypes.NewValue(tftypes.String, nil),
		"alias":                tftypes.NewValue(tftypes.String, nil),
		"ca_path_base64":       tftypes.NewValue(tftypes.String, nil),
		"expiration_date":      tftypes.NewValue(tftypes.String, nil),
		"algorithm":            tftypes.NewValue(tftypes.String, nil),
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

func BenchmarkKeystoreResource_Schema(b *testing.B) {
	res := NewKeystoreResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
