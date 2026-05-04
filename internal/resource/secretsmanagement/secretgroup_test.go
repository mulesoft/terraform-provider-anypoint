package secretsmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewSecretGroupResource(t *testing.T) {
	r := NewSecretGroupResource()

	if r == nil {
		t.Error("NewSecretGroupResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("SecretGroupResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("SecretGroupResource should implement ResourceWithImportState")
	}
}

func TestSecretGroupResource_Metadata(t *testing.T) {
	r := NewSecretGroupResource()
	testutil.TestResourceMetadata(t, r, "_secret_group")
}

func TestSecretGroupResource_Schema(t *testing.T) {
	res := NewSecretGroupResource()

	requiredAttrs := []string{"name", "environment_id"}
	optionalAttrs := []string{"downloadable", "organization_id"}
	computedAttrs := []string{"id", "current_state", "downloadable", "organization_id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestSecretGroupResource_Configure(t *testing.T) {
	res := NewSecretGroupResource().(*SecretGroupResource)

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

func TestSecretGroupResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewSecretGroupResource().(*SecretGroupResource)

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

func TestSecretGroupResource_ImportState(t *testing.T) {
	r := NewSecretGroupResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestSecretGroupResourceModel_Validation(t *testing.T) {
	model := SecretGroupResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.OrganizationID
	_ = model.Downloadable
	_ = model.CurrentState
}

func BenchmarkSecretGroupResource_Schema(b *testing.B) {
	res := NewSecretGroupResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
