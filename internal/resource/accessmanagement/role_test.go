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

func TestNewRoleResource(t *testing.T) {
	r := NewRoleResource()

	if r == nil {
		t.Error("NewRoleResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("RoleResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("RoleResource should implement ResourceWithImportState")
	}
}

func TestRoleResource_Metadata(t *testing.T) {
	r := NewRoleResource()
	testutil.TestResourceMetadata(t, r, "_role")
}

func TestRoleResource_Schema(t *testing.T) {
	res := NewRoleResource()

	requiredAttrs := []string{"name"}
	optionalAttrs := []string{"description", "organization_id"}
	computedAttrs := []string{"id", "editable", "external_names", "created_at", "updated_at"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestRoleResource_Configure(t *testing.T) {
	res := NewRoleResource().(*RoleResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
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

func TestRoleResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewRoleResource().(*RoleResource)

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

func TestRoleResource_ImportState(t *testing.T) {
	r := NewRoleResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestRoleResourceModel_Validation(t *testing.T) {
	model := RoleResourceModel{}
	_ = model.ID
}

func TestRoleResource_Read(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-role-group-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"role_group_id":  "test-role-group-id",
				"name":           "Test Role Group",
				"description":    "A test role group",
				"org_id":         "test-org-id",
				"editable":       true,
				"external_names": []string{},
				"created_at":     "2024-01-01T00:00:00Z",
				"updated_at":     "2024-01-01T00:00:00Z",
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRoleResource().(*RoleResource)
	res.client = &accessmanagement.RoleClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	permObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-role-group-id"),
		"name":            tftypes.NewValue(tftypes.String, "Test Role Group"),
		"description":     tftypes.NewValue(tftypes.String, "A test role group"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"editable":        tftypes.NewValue(tftypes.Bool, true),
		"external_names":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
		// permissions/members null => Read leaves them unmanaged (no reconcile calls).
		"permissions": tftypes.NewValue(tftypes.Set{ElementType: permObjType}, nil),
		"members":     tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		"created_at":  tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
		"updated_at":  tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got RoleResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "Test Role Group" {
		t.Errorf("Expected Name 'Test Role Group', got %s", got.Name.ValueString())
	}
	if got.Description.ValueString() != "A test role group" {
		t.Errorf("Expected Description 'A test role group', got %s", got.Description.ValueString())
	}
	if !got.Editable.ValueBool() {
		t.Errorf("Expected Editable true, got false")
	}
}

func TestRoleResource_Read_NotFound(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-role-group-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusNotFound)
			_, _ = w.Write([]byte("Role group not found"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRoleResource().(*RoleResource)
	res.client = &accessmanagement.RoleClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	permObjType := tftypes.Object{AttributeTypes: map[string]tftypes.Type{
		"name":           tftypes.String,
		"context_params": tftypes.Map{ElementType: tftypes.String},
	}}
	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-role-group-id"),
		"name":            tftypes.NewValue(tftypes.String, "Test Role Group"),
		"description":     tftypes.NewValue(tftypes.String, ""),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"editable":        tftypes.NewValue(tftypes.Bool, true),
		"external_names":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, []tftypes.Value{}),
		"permissions":     tftypes.NewValue(tftypes.Set{ElementType: permObjType}, nil),
		"members":         tftypes.NewValue(tftypes.Set{ElementType: tftypes.String}, nil),
		"created_at":      tftypes.NewValue(tftypes.String, ""),
		"updated_at":      tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	// Should not have errors - it should just remove the resource from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() should not have errors on 404, got: %v", resp.Diagnostics.Errors())
	}

	// State should be empty (resource removed)
	if !resp.State.Raw.IsNull() {
		t.Error("Read() should remove resource from state on 404")
	}
}
