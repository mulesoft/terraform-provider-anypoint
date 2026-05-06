package secretsmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	secretsmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/secretsmanagement"
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

func TestSecretGroupResource_ImportState_Valid(t *testing.T) {
	res := NewSecretGroupResource().(*SecretGroupResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"environment_id":  tftypes.NewValue(tftypes.String, nil),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"downloadable":    tftypes.NewValue(tftypes.Bool, nil),
		"current_state":   tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ImportStateRequest{ID: "test-org/test-env/test-sg-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}

	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestSecretGroupResource_ImportState_Invalid(t *testing.T) {
	res := NewSecretGroupResource().(*SecretGroupResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)

	req := resource.ImportStateRequest{ID: "invalid-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema},
	}

	res.ImportState(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ImportState() with invalid ID should produce errors")
	}
}

func TestSecretGroupResource_Read(t *testing.T) {
	mockSG := &secretsmgmt.SecretGroupResponse{
		Name:         "test-group",
		Downloadable: true,
		Meta:         secretsmgmt.SecretGroupMeta{ID: "test-sg-id"},
		CurrentState: "ACTIVE",
	}

	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				testutil.JSONResponse(w, http.StatusOK, mockSG)
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewSecretGroupResource().(*SecretGroupResource)
	res.client = &secretsmgmt.SecretGroupClient{
		AnypointClient: &anypointclient.AnypointClient{
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
		"id":              tftypes.NewValue(tftypes.String, "test-sg-id"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"name":            tftypes.NewValue(tftypes.String, "old-name"),
		"downloadable":    tftypes.NewValue(tftypes.Bool, false),
		"current_state":   tftypes.NewValue(tftypes.String, "CLEAR"),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}

	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SecretGroupResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.ID.ValueString() != "test-sg-id" {
		t.Errorf("Expected ID test-sg-id, got %s", got.ID.ValueString())
	}
	if got.Name.ValueString() != "test-group" {
		t.Errorf("Expected Name test-group, got %s", got.Name.ValueString())
	}
	if got.CurrentState.ValueString() != "ACTIVE" {
		t.Errorf("Expected CurrentState ACTIVE, got %s", got.CurrentState.ValueString())
	}
}

func TestSecretGroupResource_Read_NotFound(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/missing-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewSecretGroupResource().(*SecretGroupResource)
	res.client = &secretsmgmt.SecretGroupClient{
		AnypointClient: &anypointclient.AnypointClient{
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
		"id":              tftypes.NewValue(tftypes.String, "missing-id"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"name":            tftypes.NewValue(tftypes.String, "group"),
		"downloadable":    tftypes.NewValue(tftypes.Bool, false),
		"current_state":   tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}

	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() on not-found should remove resource, not error: %v", resp.Diagnostics.Errors())
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Read() on not-found should set state to null (removed)")
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
