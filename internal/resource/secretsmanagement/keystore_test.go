package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

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
