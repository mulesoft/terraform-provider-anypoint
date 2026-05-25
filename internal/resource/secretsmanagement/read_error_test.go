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

// nullSMState builds a tfsdk.State filled with null values for a resource.
func nullSMState(t *testing.T, r resource.Resource) (resource.SchemaResponse, tfsdk.State) {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	return schemaResp, tfsdk.State{Schema: schemaResp.Schema, Raw: nullSMValue(t, stateType)}
}

func nullSMValue(t *testing.T, tfType tftypes.Type) tftypes.Value {
	t.Helper()
	switch typ := tfType.(type) {
	case tftypes.Object:
		vals := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for k, at := range typ.AttributeTypes {
			vals[k] = nullSMValue(t, at)
		}
		return tftypes.NewValue(typ, vals)
	default:
		return tftypes.NewValue(tfType, nil)
	}
}

// buildSMReadState creates a populated state map with org/env/sg/id set.
func buildSMReadState(t *testing.T, r resource.Resource, orgID, envID, sgID, id string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for k, at := range objType.AttributeTypes {
		vals[k] = nullSMValue(t, at)
	}
	set := func(key, val string) {
		if _, ok := objType.AttributeTypes[key]; ok {
			vals[key] = tftypes.NewValue(tftypes.String, val)
		}
	}
	set("id", id)
	set("organization_id", orgID)
	set("environment_id", envID)
	set("secret_group_id", sgID)
	raw := tftypes.NewValue(stateType, vals)
	return tfsdk.State{Schema: schemaResp.Schema, Raw: raw}
}

// ── SecretGroupResource.Read error paths ─────────────────────────────────────

func TestSecretGroupResource_Read_Error(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/org-1/environments/env-1/secretGroups/sg-abc"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewSecretGroupResource().(*SecretGroupResource)
	res.client = &secretsmgmt.SecretGroupClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "tok",
			HTTPClient: &http.Client{},
			OrgID:      "org-1",
		},
	}

	ctx := context.Background()
	schemaResp, _ := nullSMState(t, res)
	state := buildSMReadState(t, res, "org-1", "env-1", "", "sg-abc")

	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// ── CertificateResource.Read error paths ─────────────────────────────────────

func TestCertificateResource_Read_Error(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/org-1/environments/env-1/secretGroups/sg-1/certificates/cert-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewCertificateResource().(*CertificateResource)
	res.client = &secretsmgmt.CertificateClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "tok",
			HTTPClient: &http.Client{},
			OrgID:      "org-1",
		},
	}

	ctx := context.Background()
	schemaResp, _ := nullSMState(t, res)
	state := buildSMReadState(t, res, "org-1", "env-1", "sg-1", "cert-1")

	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// ── CertificatePinsetResource.Read error paths ───────────────────────────────

func TestCertificatePinsetResource_Read_Error(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/org-1/environments/env-1/secretGroups/sg-1/certificatePinsets/pin-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewCertificatePinsetResource().(*CertificatePinsetResource)
	res.client = &secretsmgmt.CertificatePinsetClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "tok",
			HTTPClient: &http.Client{},
			OrgID:      "org-1",
		},
	}

	ctx := context.Background()
	schemaResp, _ := nullSMState(t, res)
	state := buildSMReadState(t, res, "org-1", "env-1", "sg-1", "pin-1")

	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// ── TLSContextResource (secretsmanagement).Read error paths ──────────────────

func TestTLSContextResource_SecretsManagement_Read_Error(t *testing.T) {
	basePath := "/secrets-manager/api/v1/organizations/org-1/environments/env-1/secretGroups/sg-1/tlsContexts/tls-1"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewTLSContextResource().(*TLSContextResource)
	res.client = &secretsmgmt.TLSContextClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "tok",
			HTTPClient: &http.Client{},
			OrgID:      "org-1",
		},
	}

	ctx := context.Background()
	schemaResp, _ := nullSMState(t, res)
	state := buildSMReadState(t, res, "org-1", "env-1", "sg-1", "tls-1")

	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}
