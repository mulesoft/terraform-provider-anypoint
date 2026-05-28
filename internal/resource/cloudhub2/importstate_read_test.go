package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// nullStateFor builds a tfsdk.State where every attribute is null.
func nullStateFor(t *testing.T, r resource.Resource) (resource.SchemaResponse, tftypes.Value) {
	t.Helper()
	ctx := context.Background()
	schemaResp := resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	return schemaResp, nullTFValue(t, stateType)
}

func nullTFValue(t *testing.T, tfType tftypes.Type) tftypes.Value {
	t.Helper()
	switch typ := tfType.(type) {
	case tftypes.Object:
		vals := make(map[string]tftypes.Value, len(typ.AttributeTypes))
		for k, at := range typ.AttributeTypes {
			vals[k] = nullTFValue(t, at)
		}
		return tftypes.NewValue(typ, vals)
	default:
		return tftypes.NewValue(tfType, nil)
	}
}

// ── PrivateSpaceConfigResource.ImportState ───────────────────────────────────

func TestPrivateSpaceConfigResource_ImportState_PassthroughID(t *testing.T) {
	r := NewPrivateSpaceConfigResource().(*PrivateSpaceConfigResource)
	ctx := context.Background()
	schemaResp, rawState := nullStateFor(t, r)

	req := resource.ImportStateRequest{ID: "ps-uuid-abc"}
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
	r.ImportState(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
	}
	var got PrivateSpaceConfigResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "ps-uuid-abc" {
		t.Errorf("ID = %q, want ps-uuid-abc", got.ID.ValueString())
	}
}

// ── PrivateSpaceConfigResource.Read ──────────────────────────────────────────

func TestPrivateSpaceConfigResource_Read_Success(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/ps-1"

	mockPS := ch2client.PrivateSpace{
		ID:     "ps-1",
		Name:   "my-space",
		Status: "Running",
		Network: ch2client.NetworkConfig{
			Region:    "us-east-1",
			CidrBlock: "10.0.0.0/16",
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockPS)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewPrivateSpaceConfigResource().(*PrivateSpaceConfigResource)
	res.spaceClient = &ch2client.PrivateSpacesClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}
	res.networkClient = &ch2client.PrivateNetworkClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}
	res.firewallClient = &ch2client.FirewallRulesClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp, _ := nullStateFor(t, res)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	// Build state with org and space id set (normal read, not import)
	priorRaw := tftypes.NewValue(stateType.(tftypes.Object), func() map[string]tftypes.Value {
		vals := map[string]tftypes.Value{}
		for k, at := range stateType.(tftypes.Object).AttributeTypes {
			vals[k] = nullTFValue(t, at)
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "ps-1")
		vals["organization_id"] = tftypes.NewValue(tftypes.String, "test-org-id")
		return vals
	}())

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got PrivateSpaceConfigResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "ps-1" {
		t.Errorf("ID = %q, want ps-1", got.ID.ValueString())
	}
	if got.Name.ValueString() != "my-space" {
		t.Errorf("Name = %q, want my-space", got.Name.ValueString())
	}
}

func TestPrivateSpaceConfigResource_Read_NotFound(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/ps-missing"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewPrivateSpaceConfigResource().(*PrivateSpaceConfigResource)
	res.spaceClient = &ch2client.PrivateSpacesClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}
	res.networkClient = &ch2client.PrivateNetworkClient{AnypointClient: &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id"}}
	res.firewallClient = &ch2client.FirewallRulesClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL: server.URL, Token: "mock-token",
			HTTPClient: &http.Client{}, OrgID: "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp, _ := nullStateFor(t, res)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorRaw := tftypes.NewValue(stateType.(tftypes.Object), func() map[string]tftypes.Value {
		vals := map[string]tftypes.Value{}
		for k, at := range stateType.(tftypes.Object).AttributeTypes {
			vals[k] = nullTFValue(t, at)
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "ps-missing")
		vals["organization_id"] = tftypes.NewValue(tftypes.String, "test-org-id")
		return vals
	}())

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource")
	}
}

func TestPrivateSpaceConfigResource_Read_Error(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/ps-err"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewPrivateSpaceConfigResource().(*PrivateSpaceConfigResource)
	res.spaceClient = &ch2client.PrivateSpacesClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}
	res.networkClient = &ch2client.PrivateNetworkClient{AnypointClient: &anypointclient.AnypointClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}, OrgID: "test-org-id"}}
	res.firewallClient = &ch2client.FirewallRulesClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL: server.URL, Token: "mock-token",
			HTTPClient: &http.Client{}, OrgID: "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp, _ := nullStateFor(t, res)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	priorRaw := tftypes.NewValue(stateType.(tftypes.Object), func() map[string]tftypes.Value {
		vals := map[string]tftypes.Value{}
		for k, at := range stateType.(tftypes.Object).AttributeTypes {
			vals[k] = nullTFValue(t, at)
		}
		vals["id"] = tftypes.NewValue(tftypes.String, "ps-err")
		vals["organization_id"] = tftypes.NewValue(tftypes.String, "test-org-id")
		return vals
	}())

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorRaw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// ── TLSContextResource (cloudhub2) ImportState – error branch ────────────────

func TestTLSContextResource_CloudHub2_ImportState_InvalidFormat(t *testing.T) {
	r := NewTLSContextResource().(*TLSContextResource)
	ctx := context.Background()
	schemaResp, rawState := nullStateFor(t, r)

	t.Run("valid 3-part colon ID sets org, private_space_id and id", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1:ps-1:ctx-1"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors for 3-part ID: %v", resp.Diagnostics.Errors())
		}
		var got TLSContextResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", got.OrganizationID.ValueString())
		}
		if got.PrivateSpaceID.ValueString() != "ps-1" {
			t.Errorf("PrivateSpaceID = %q, want ps-1", got.PrivateSpaceID.ValueString())
		}
		if got.ID.ValueString() != "ctx-1" {
			t.Errorf("ID = %q, want ctx-1", got.ID.ValueString())
		}
	})

	t.Run("4-part colon ID errors", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "org-1:ps-1:ctx-1:extra"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 4-part colon ID")
		}
	})

	t.Run("no colon ID errors", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "just-an-id-no-colon"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for ID without colon")
		}
	})

	t.Run("valid colon-separated ID sets attributes", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "ps-123:ctx-456"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got TLSContextResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.PrivateSpaceID.ValueString() != "ps-123" {
			t.Errorf("PrivateSpaceID = %q, want ps-123", got.PrivateSpaceID.ValueString())
		}
		if got.ID.ValueString() != "ctx-456" {
			t.Errorf("ID = %q, want ctx-456", got.ID.ValueString())
		}
	})
}

// ── PrivateSpaceAssociationResource.ImportState ───────────────────────────────

func TestPrivateSpaceAssociationResource_ImportState_SetsIDs(t *testing.T) {
	r := NewPrivateSpaceAssociationResource().(*PrivateSpaceAssociationResource)
	ctx := context.Background()
	schemaResp, rawState := nullStateFor(t, r)

	req := resource.ImportStateRequest{ID: "ps-abc"}
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
	r.ImportState(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
	}
	var got PrivateSpaceAssociationResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.PrivateSpaceID.ValueString() != "ps-abc" {
		t.Errorf("PrivateSpaceID = %q, want ps-abc", got.PrivateSpaceID.ValueString())
	}
	if got.ID.ValueString() != "ps-abc-associations" {
		t.Errorf("ID = %q, want ps-abc-associations", got.ID.ValueString())
	}
}

// ── PrivateSpaceAdvancedConfigResource.ImportState ────────────────────────────

func TestPrivateSpaceAdvancedConfigResource_ImportState_SetsIDs(t *testing.T) {
	r := NewPrivateSpaceAdvancedConfigResource().(*PrivateSpaceAdvancedConfigResource)
	ctx := context.Background()
	schemaResp, rawState := nullStateFor(t, r)

	req := resource.ImportStateRequest{ID: "ps-xyz"}
	resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
	r.ImportState(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
	}
	var got PrivateSpaceAdvancedConfigResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "ps-xyz" {
		t.Errorf("ID = %q, want ps-xyz", got.ID.ValueString())
	}
	if got.PrivateSpaceID.ValueString() != "ps-xyz" {
		t.Errorf("PrivateSpaceID = %q, want ps-xyz", got.PrivateSpaceID.ValueString())
	}
}
