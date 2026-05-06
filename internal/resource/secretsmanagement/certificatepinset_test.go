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
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewCertificatePinsetResource(t *testing.T) {
	r := NewCertificatePinsetResource()

	if r == nil {
		t.Error("NewCertificatePinsetResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("CertificatePinsetResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("CertificatePinsetResource should implement ResourceWithImportState")
	}
}

func TestCertificatePinsetResource_Metadata(t *testing.T) {
	r := NewCertificatePinsetResource()
	testutil.TestResourceMetadata(t, r, "_secret_group_certificate_pinset")
}

func TestCertificatePinsetResource_Schema(t *testing.T) {
	res := NewCertificatePinsetResource()

	requiredAttrs := []string{"name", "environment_id", "secret_group_id", "certificate_pinset_base64"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id", "expiration_date", "algorithm", "organization_id"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestCertificatePinsetResource_Configure(t *testing.T) {
	res := NewCertificatePinsetResource().(*CertificatePinsetResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestCertificatePinsetResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewCertificatePinsetResource().(*CertificatePinsetResource)

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

func TestCertificatePinsetResource_ImportState(t *testing.T) {
	r := NewCertificatePinsetResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestCertificatePinsetResourceModel_Validation(t *testing.T) {
	model := CertificatePinsetResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.EnvironmentID
	_ = model.SecretGroupID
	_ = model.OrganizationID
	_ = model.PinsetB64
	_ = model.ExpirationDate
	_ = model.Algorithm
}

func TestCertificatePinsetResource_Read(t *testing.T) {
	mockPin := &secretsmgmt.CertificatePinsetResponse{
		Name: "test-pinset",
		Meta: secretsmgmt.SecretGroupMeta{ID: "pin-id-1"},
	}

	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/certificatePinsets/pin-id-1"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "GET" {
				testutil.JSONResponse(w, http.StatusOK, mockPin)
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewCertificatePinsetResource().(*CertificatePinsetResource)
	res.client = &secretsmgmt.CertificatePinsetClient{
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
		"id":                        tftypes.NewValue(tftypes.String, "pin-id-1"),
		"organization_id":           tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":            tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":           tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":                      tftypes.NewValue(tftypes.String, "old-name"),
		"certificate_pinset_base64": tftypes.NewValue(tftypes.String, nil),
		"expiration_date":           tftypes.NewValue(tftypes.String, ""),
		"algorithm":                 tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got CertificatePinsetResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "test-pinset" {
		t.Errorf("Expected Name test-pinset, got %s", got.Name.ValueString())
	}
}

func TestCertificatePinsetResource_Read_NotFound(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/test-org-id/environments/test-env-id/secretGroups/test-sg-id/certificatePinsets/missing-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewCertificatePinsetResource().(*CertificatePinsetResource)
	res.client = &secretsmgmt.CertificatePinsetClient{
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
		"id":                        tftypes.NewValue(tftypes.String, "missing-id"),
		"organization_id":           tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":            tftypes.NewValue(tftypes.String, "test-env-id"),
		"secret_group_id":           tftypes.NewValue(tftypes.String, "test-sg-id"),
		"name":                      tftypes.NewValue(tftypes.String, "pin"),
		"certificate_pinset_base64": tftypes.NewValue(tftypes.String, nil),
		"expiration_date":           tftypes.NewValue(tftypes.String, ""),
		"algorithm":                 tftypes.NewValue(tftypes.String, ""),
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

func TestCertificatePinsetResource_ImportState_Valid(t *testing.T) {
	res := NewCertificatePinsetResource().(*CertificatePinsetResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                        tftypes.NewValue(tftypes.String, nil),
		"organization_id":           tftypes.NewValue(tftypes.String, nil),
		"environment_id":            tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":           tftypes.NewValue(tftypes.String, nil),
		"name":                      tftypes.NewValue(tftypes.String, nil),
		"certificate_pinset_base64": tftypes.NewValue(tftypes.String, nil),
		"expiration_date":           tftypes.NewValue(tftypes.String, nil),
		"algorithm":                 tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ImportStateRequest{ID: "org-id/env-id/sg-id/pin-id"}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: emptyStateRaw},
	}
	res.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() reported errors: %v", resp.Diagnostics.Errors())
	}
}

func TestCertificatePinsetResource_ImportState_Invalid(t *testing.T) {
	res := NewCertificatePinsetResource().(*CertificatePinsetResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	emptyStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":                        tftypes.NewValue(tftypes.String, nil),
		"organization_id":           tftypes.NewValue(tftypes.String, nil),
		"environment_id":            tftypes.NewValue(tftypes.String, nil),
		"secret_group_id":           tftypes.NewValue(tftypes.String, nil),
		"name":                      tftypes.NewValue(tftypes.String, nil),
		"certificate_pinset_base64": tftypes.NewValue(tftypes.String, nil),
		"expiration_date":           tftypes.NewValue(tftypes.String, nil),
		"algorithm":                 tftypes.NewValue(tftypes.String, nil),
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

func BenchmarkCertificatePinsetResource_Schema(b *testing.B) {
	res := NewCertificatePinsetResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}
