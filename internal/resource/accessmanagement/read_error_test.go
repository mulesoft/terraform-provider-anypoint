package accessmanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	accessmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// buildAMReadState builds a tfsdk.State with the given string fields patched in.
func buildAMReadState(t *testing.T, r resource.Resource, fields map[string]string) tfsdk.State {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	vals := map[string]tftypes.Value{}
	for k, at := range objType.AttributeTypes {
		vals[k] = nullAMValue(t, at)
	}
	for k, v := range fields {
		if _, ok := objType.AttributeTypes[k]; ok {
			vals[k] = tftypes.NewValue(tftypes.String, v)
		}
	}
	return tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, vals)}
}

func nullAMValue(t *testing.T, tfType tftypes.Type) tftypes.Value {
	t.Helper()
	switch typ := tfType.(type) {
	case tftypes.Object:
		vals := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for k, at := range typ.AttributeTypes {
			vals[k] = nullAMValue(t, at)
		}
		return tftypes.NewValue(typ, vals)
	default:
		return tftypes.NewValue(tfType, nil)
	}
}

// ── ConnectedAppScopesResource.Read error paths ──────────────────────────────

func TestConnectedAppScopesResource_Read_ServerError(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/connectedApplications/app-1/scopes": func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "server error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewConnectedAppScopesResource().(*ConnectedAppScopesResource)
	res.client = &accessmgmt.ConnectedAppScopesClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "tok",
			HTTPClient: &http.Client{},
			OrgID:      "org-1",
		},
	}

	ctx := context.Background()
	state := buildAMReadState(t, res, map[string]string{
		"id":               "app-1",
		"connected_app_id": "app-1",
	})

	schemaResp := resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// ── OrganizationResource.Read error path ─────────────────────────────────────

func TestOrganizationResource_Read_Error(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/accounts/api/organizations/org-err": func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "server error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewOrganizationResource().(*OrganizationResource)
	res.client = &accessmgmt.OrganizationClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "tok",
			HTTPClient: &http.Client{},
			OrgID:      "org-err",
		},
	}

	ctx := context.Background()
	state := buildAMReadState(t, res, map[string]string{"id": "org-err"})

	schemaResp := resource.SchemaResponse{}
	res.Schema(ctx, resource.SchemaRequest{}, &schemaResp)

	req := resource.ReadRequest{State: state}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: state.Raw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}
