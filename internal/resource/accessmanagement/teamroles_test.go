package accessmanagement

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTeamRolesResource(t *testing.T) {
	r := NewTeamRolesResource()

	if r == nil {
		t.Error("NewTeamRolesResource() returned nil")
	}

	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("TeamRolesResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("TeamRolesResource should implement ResourceWithImportState")
	}
}

func TestTeamRolesResource_Metadata(t *testing.T) {
	r := NewTeamRolesResource()
	testutil.TestResourceMetadata(t, r, "_team_roles")
}

func TestTeamRolesResource_Schema(t *testing.T) {
	res := NewTeamRolesResource()

	requiredAttrs := []string{"team_id", "roles"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestTeamRolesResource_Configure(t *testing.T) {
	res := NewTeamRolesResource().(*TeamRolesResource)

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

func TestTeamRolesResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewTeamRolesResource().(*TeamRolesResource)

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

func TestTeamRolesResource_ImportState(t *testing.T) {
	r := NewTeamRolesResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestTeamRolesResourceModel_Validation(t *testing.T) {
	model := TeamRolesResourceModel{}
	_ = model.ID
}

func BenchmarkTeamRolesResource_Schema(b *testing.B) {
	res := NewTeamRolesResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
