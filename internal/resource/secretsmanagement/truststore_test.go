package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTruststoreResource(t *testing.T) {
	r := NewTruststoreResource()

	if r == nil {
		t.Error("NewTruststoreResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("TruststoreResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("TruststoreResource should implement ResourceWithImportState")
	}
}

func TestTruststoreResource_Metadata(t *testing.T) {
	r := NewTruststoreResource()
	testutil.TestResourceMetadata(t, r, "_secret_group_truststore")
}

func TestTruststoreResource_Schema(t *testing.T) {
	res := NewTruststoreResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id", "truststore_base64"}
	optionalAttrs := []string{"type", "organization_id", "passphrase"}
	computedAttrs := []string{"id", "expiration_date", "algorithm", "type", "organization_id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTruststoreResource_Configure(t *testing.T) {
	res := NewTruststoreResource().(*TruststoreResource)

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

func TestTruststoreResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTruststoreResource().(*TruststoreResource)

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

func TestTruststoreResource_ImportState(t *testing.T) {
	r := NewTruststoreResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTruststoreResourceModel_Validation(t *testing.T) {
	model := TruststoreResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.Type
	_ = model.TrustStoreB64
	_ = model.Passphrase
	_ = model.ExpirationDate
	_ = model.Algorithm
}

func BenchmarkTruststoreResource_Schema(b *testing.B) {
	res := NewTruststoreResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
