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

func TestNewSharedSecretResource(t *testing.T) {
	r := NewSharedSecretResource()

	if r == nil {
		t.Error("NewSharedSecretResource() returned nil")
	}

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

func TestSharedSecretResource_Read(t *testing.T) {
	mockSS := &secretsmgmt.SharedSecretResponse{
		Name: "test-secret",
		Type: "UsernamePassword",
		Meta: secretsmgmt.SecretGroupMeta{ID: "ss-id-1"},
	}

	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/sharedSecrets/ss-id-1"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				testutil.JSONResponse(w, http.StatusOK, mockSS)
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewSharedSecretResource().(*SharedSecretResource)
	res.client = &secretsmgmt.SharedSecretClient{
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
		"id":                tftypes.NewValue(tftypes.String, "ss-id-1"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":   tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":              tftypes.NewValue(tftypes.String, "old-name"),
		"type":              tftypes.NewValue(tftypes.String, "UsernamePassword"),
		"expiration_date":   tftypes.NewValue(tftypes.String, ""),
		"username":          tftypes.NewValue(tftypes.String, nil),
		"password":          tftypes.NewValue(tftypes.String, nil),
		"access_key_id":     tftypes.NewValue(tftypes.String, nil),
		"secret_access_key": tftypes.NewValue(tftypes.String, nil),
		"key":               tftypes.NewValue(tftypes.String, nil),
		"content":           tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got SharedSecretResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "test-secret" {
		t.Errorf("Expected Name test-secret, got %s", got.Name.ValueString())
	}
}

func TestSharedSecretResource_Read_NotFound(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/sharedSecrets/missing-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewSharedSecretResource().(*SharedSecretResource)
	res.client = &secretsmgmt.SharedSecretClient{
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
		"id":                tftypes.NewValue(tftypes.String, "missing-id"),
		"organization_id":   tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":    tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":   tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":              tftypes.NewValue(tftypes.String, "ss"),
		"type":              tftypes.NewValue(tftypes.String, "UsernamePassword"),
		"expiration_date":   tftypes.NewValue(tftypes.String, ""),
		"username":          tftypes.NewValue(tftypes.String, nil),
		"password":          tftypes.NewValue(tftypes.String, nil),
		"access_key_id":     tftypes.NewValue(tftypes.String, nil),
		"secret_access_key": tftypes.NewValue(tftypes.String, nil),
		"key":               tftypes.NewValue(tftypes.String, nil),
		"content":           tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() on not-found should remove resource: %v", resp.Diagnostics.Errors())
	}
	if !resp.State.Raw.IsNull() {
		t.Error("Read() on not-found should remove resource from state")
	}
}

func TestSharedSecretResource_ImportState_Valid(t *testing.T) {
	res := NewSharedSecretResource().(*SharedSecretResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, nil),
		"organization_id":   tftypes.NewValue(tftypes.String, nil),
		"environment_id":    tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":   tftypes.NewValue(tftypes.String, nil),
		"name":              tftypes.NewValue(tftypes.String, nil),
		"type":              tftypes.NewValue(tftypes.String, nil),
		"expiration_date":   tftypes.NewValue(tftypes.String, nil),
		"username":          tftypes.NewValue(tftypes.String, nil),
		"password":          tftypes.NewValue(tftypes.String, nil),
		"access_key_id":     tftypes.NewValue(tftypes.String, nil),
		"secret_access_key": tftypes.NewValue(tftypes.String, nil),
		"key":               tftypes.NewValue(tftypes.String, nil),
		"content":           tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ImportStateRequest{ID: "org-id/env-id/sg-id/ss-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestSharedSecretResource_ImportState_Invalid(t *testing.T) {
	res := NewSharedSecretResource().(*SharedSecretResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, nil),
		"organization_id":   tftypes.NewValue(tftypes.String, nil),
		"environment_id":    tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":   tftypes.NewValue(tftypes.String, nil),
		"name":              tftypes.NewValue(tftypes.String, nil),
		"type":              tftypes.NewValue(tftypes.String, nil),
		"expiration_date":   tftypes.NewValue(tftypes.String, nil),
		"username":          tftypes.NewValue(tftypes.String, nil),
		"password":          tftypes.NewValue(tftypes.String, nil),
		"access_key_id":     tftypes.NewValue(tftypes.String, nil),
		"secret_access_key": tftypes.NewValue(tftypes.String, nil),
		"key":               tftypes.NewValue(tftypes.String, nil),
		"content":           tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ImportStateRequest{ID: "invalid/short"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("ImportState() with invalid ID should produce errors")
	}
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
