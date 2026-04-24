package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTeamResource(t *testing.T) {
	r := NewTeamResource()

	if r == nil {
		t.Error("NewTeamResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("TeamResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("TeamResource should implement ResourceWithImportState")
	}
}

func TestTeamResource_Metadata(t *testing.T) {
	r := NewTeamResource()
	testutil.TestResourceMetadata(t, r, "_team")
}

func TestTeamResource_Schema(t *testing.T) {
	res := NewTeamResource()

	requiredAttrs := []string{"team_name", "parent_team_id", "team_type"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id", "created_at", "updated_at"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTeamResource_Configure(t *testing.T) {
	res := NewTeamResource().(*TeamResource)

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

func TestTeamResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTeamResource().(*TeamResource)

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

func TestTeamResource_ImportState(t *testing.T) {
	r := NewTeamResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTeamResourceModel_Validation(t *testing.T) {
	model := TeamResourceModel{}
	_ = model.ID
}

func BenchmarkTeamResource_Schema(b *testing.B) {
	res := NewTeamResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
