package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewEnvironmentResource(t *testing.T) {
	r := NewEnvironmentResource()

	if r == nil {
		t.Error("NewEnvironmentResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("EnvironmentResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("EnvironmentResource should implement ResourceWithImportState")
	}
}

func TestEnvironmentResource_Metadata(t *testing.T) {
	r := NewEnvironmentResource()
	testutil.TestResourceMetadata(t, r, "_environment")
}

func TestEnvironmentResource_Schema(t *testing.T) {
	res := NewEnvironmentResource()

	requiredAttrs := []string{"name"}
	optionalAttrs := []string{"type", "is_production", "organization_id", "client_id", "arc_namespace"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestEnvironmentResource_Configure(t *testing.T) {
	res := NewEnvironmentResource().(*EnvironmentResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.ClientConfig{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestEnvironmentResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewEnvironmentResource().(*EnvironmentResource)

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

func TestEnvironmentResource_ImportState(t *testing.T) {
	r := NewEnvironmentResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestEnvironmentResourceModel_Validation(t *testing.T) {
	model := EnvironmentResourceModel{}
	_ = model.ID
}

func BenchmarkEnvironmentResource_Schema(b *testing.B) {
	res := NewEnvironmentResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
