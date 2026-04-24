package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewSharedSecretResource(t *testing.T) {
	r := NewSharedSecretResource()

	if r == nil {
		t.Error("NewSharedSecretResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("SharedSecretResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("SharedSecretResource should implement ResourceWithImportState")
	}
}

func TestSharedSecretResource_Metadata(t *testing.T) {
	r := NewSharedSecretResource()
	testutil.TestResourceMetadata(t, r, "_secret_group_shared_secret")
}

func TestSharedSecretResource_Schema(t *testing.T) {
	res := NewSharedSecretResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id", "type"}
	optionalAttrs := []string{"organization_id", "expiration_date", "username", "password", "access_key_id", "secret_access_key", "key", "content"}
	computedAttrs := []string{"id", "expiration_date", "organization_id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestSharedSecretResource_Configure(t *testing.T) {
	res := NewSharedSecretResource().(*SharedSecretResource)

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

func TestSharedSecretResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewSharedSecretResource().(*SharedSecretResource)

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

func TestSharedSecretResource_ImportState(t *testing.T) {
	r := NewSharedSecretResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestSharedSecretResourceModel_Validation(t *testing.T) {
	model := SharedSecretResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.Type
	_ = model.ExpirationDate
	_ = model.Username
	_ = model.Password
	_ = model.AccessKeyID
	_ = model.SecretAccessKey
	_ = model.Key
	_ = model.Content
}

func BenchmarkSharedSecretResource_Schema(b *testing.B) {
	res := NewSharedSecretResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
