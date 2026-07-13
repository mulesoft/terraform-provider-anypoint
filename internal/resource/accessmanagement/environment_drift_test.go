package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// TestEnvironmentResource_Read_DetectsBackendNameDrift validates that when an
// environment is renamed in the Anypoint UI (backend), a subsequent Terraform
// refresh surfaces the drift by updating state.Name to the server-reported
// value. This regression guards the reported bug where renaming an environment
// via the UI did not produce any plan diff on `terraform apply`.
func TestEnvironmentResource_Read_DetectsBackendNameDrift(t *testing.T) {
	const (
		orgID      = "test-org-id"
		envID      = "test-env-id"
		oldName    = "original-name"
		backendNew = "renamed-via-ui"
	)

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/" + orgID + "/environments/" + envID: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "unexpected method "+r.Method)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             envID,
				"name":           backendNew,
				"type":           "sandbox",
				"isProduction":   false,
				"organizationId": orgID,
				"clientId":       "client-id-abc",
			})
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	envClient := &accessmanagement.EnvironmentClient{
		UserAnypointClient: &client.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      orgID,
		},
	}

	envResource := &EnvironmentResource{client: envClient}

	ctx := context.Background()

	// Build the schema and the tftypes representation of prior state.
	schemaResp := &resource.SchemaResponse{}
	envResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	if schemaResp.Diagnostics.HasError() {
		t.Fatalf("schema errors: %v", schemaResp.Diagnostics.Errors())
	}
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, envID),
		"name":            tftypes.NewValue(tftypes.String, oldName),
		"type":            tftypes.NewValue(tftypes.String, "sandbox"),
		"is_production":   tftypes.NewValue(tftypes.Bool, false),
		"organization_id": tftypes.NewValue(tftypes.String, orgID),
		"client_id":       tftypes.NewValue(tftypes.String, "client-id-abc"),
		"arc_namespace":   tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    priorStateRaw,
		},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    priorStateRaw,
		},
	}

	envResource.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got EnvironmentResourceModel
	diags := resp.State.Get(ctx, &got)
	if diags.HasError() {
		t.Fatalf("resp.State.Get errors: %v", diags.Errors())
	}

	if got.Name.ValueString() != backendNew {
		t.Fatalf("expected state.name to reflect backend rename %q, got %q (prior was %q)",
			backendNew, got.Name.ValueString(), oldName)
	}

	if got.ID.ValueString() != envID {
		t.Errorf("expected state.id %q, got %q", envID, got.ID.ValueString())
	}
}

// TestEnvironmentResource_Read_DetectsBackendTypeAndIsProductionDrift confirms
// that backend changes to the environment `type` and `is_production` flags
// also propagate into refreshed state, matching the same contract as `name`.
func TestEnvironmentResource_Read_DetectsBackendTypeAndIsProductionDrift(t *testing.T) {
	const (
		orgID = "test-org-id"
		envID = "test-env-id"
	)

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/" + orgID + "/environments/" + envID: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodGet {
				testutil.ErrorResponse(w, http.StatusMethodNotAllowed, "unexpected method "+r.Method)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(http.StatusOK)
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"id":             envID,
				"name":           "keep-same-name",
				"type":           "production",
				"isProduction":   true,
				"organizationId": orgID,
				"clientId":       "client-id-abc",
			})
		},
		"/accounts/api/v2/oauth2/token": testutil.StandardMockHandlers()["/accounts/api/v2/oauth2/token"],
		"/accounts/api/me":              testutil.StandardMockHandlers()["/accounts/api/me"],
	}

	server := testutil.MockHTTPServer(t, handlers)

	envClient := &accessmanagement.EnvironmentClient{
		UserAnypointClient: &client.UserAnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      orgID,
		},
	}

	envResource := &EnvironmentResource{client: envClient}

	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	envResource.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, envID),
		"name":            tftypes.NewValue(tftypes.String, "keep-same-name"),
		"type":            tftypes.NewValue(tftypes.String, "sandbox"),
		"is_production":   tftypes.NewValue(tftypes.Bool, false),
		"organization_id": tftypes.NewValue(tftypes.String, orgID),
		"client_id":       tftypes.NewValue(tftypes.String, "client-id-abc"),
		"arc_namespace":   tftypes.NewValue(tftypes.String, nil),
	})

	req := resource.ReadRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}
	resp := &resource.ReadResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw},
	}

	envResource.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got EnvironmentResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("resp.State.Get errors: %v", diags.Errors())
	}

	if got.Type.ValueString() != "production" {
		t.Errorf("expected state.type to reflect backend change to production, got %q", got.Type.ValueString())
	}
	if !got.IsProduction.ValueBool() {
		t.Error("expected state.is_production to reflect backend change to true, got false")
	}
}
