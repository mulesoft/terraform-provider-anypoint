package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTeamResource(t *testing.T) {
	r := NewTeamResource()

	if r == nil {
		t.Error("NewTeamResource() returned nil")
	}

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

func TestTeamResource_Read(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/teams/test-team-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"team_id":    "test-team-id",
				"team_name":  "My Team",
				"team_type":  "internal",
				"org_id":     "test-org-id",
				"created_at": "2024-01-01T00:00:00Z",
				"updated_at": "2024-01-01T00:00:00Z",
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTeamResource().(*TeamResource)
	res.client = &accessmanagement.TeamClient{
		AnypointClient: &client.AnypointClient{
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
		"id":              tftypes.NewValue(tftypes.String, "test-team-id"),
		"team_name":       tftypes.NewValue(tftypes.String, "My Team"),
		"parent_team_id":  tftypes.NewValue(tftypes.String, ""),
		"team_type":       tftypes.NewValue(tftypes.String, "internal"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"created_at":      tftypes.NewValue(tftypes.String, ""),
		"updated_at":      tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got TeamResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.TeamName.ValueString() != "My Team" {
		t.Errorf("Expected TeamName 'My Team', got %s", got.TeamName.ValueString())
	}
}

func TestTeamResource_Read_NotFound(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/teams/test-team-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTeamResource().(*TeamResource)
	res.client = &accessmanagement.TeamClient{
		AnypointClient: &client.AnypointClient{
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
		"id":              tftypes.NewValue(tftypes.String, "test-team-id"),
		"team_name":       tftypes.NewValue(tftypes.String, "My Team"),
		"parent_team_id":  tftypes.NewValue(tftypes.String, ""),
		"team_type":       tftypes.NewValue(tftypes.String, "internal"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"created_at":      tftypes.NewValue(tftypes.String, ""),
		"updated_at":      tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
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
