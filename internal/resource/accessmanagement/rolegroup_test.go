package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRoleGroupResource(t *testing.T) {
	r := NewRoleGroupResource()

	if r == nil {
		t.Error("NewRoleGroupResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("RoleGroupResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("RoleGroupResource should implement ResourceWithImportState")
	}
}

func TestRoleGroupResource_Metadata(t *testing.T) {
	r := NewRoleGroupResource()
	testutil.TestResourceMetadata(t, r, "_rolegroup")
}

func TestRoleGroupResource_Schema(t *testing.T) {
	res := NewRoleGroupResource()

	requiredAttrs := []string{"name", "description"}
	optionalAttrs := []string{"external_names", "organization_id"}
	computedAttrs := []string{"id", "editable", "created_at", "updated_at"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestRoleGroupResource_Configure(t *testing.T) {
	res := NewRoleGroupResource().(*RoleGroupResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.ClientConfig{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestRoleGroupResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewRoleGroupResource().(*RoleGroupResource)

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

func TestRoleGroupResource_ImportState(t *testing.T) {
	r := NewRoleGroupResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestRoleGroupResourceModel_Validation(t *testing.T) {
	model := RoleGroupResourceModel{}
	_ = model.ID
}

func BenchmarkRoleGroupResource_Schema(b *testing.B) {
	res := NewRoleGroupResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
