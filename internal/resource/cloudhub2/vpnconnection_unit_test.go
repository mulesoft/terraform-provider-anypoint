package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/hashicorp/terraform-plugin-framework/types"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// --- ImportState ---

func TestVPNConnectionResource_ImportState_IDParsing(t *testing.T) {
	r := NewVPNConnectionResource().(*VPNConnectionResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	rawState := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"private_space_id": tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, nil),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"vpns":            tftypes.NewValue(objType.AttributeTypes["vpns"], nil),
	})

	t.Run("valid 2-part ID sets private_space_id and id", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "ps-123/vpn-conn-456"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		var got VPNConnectionResourceModel
		if diags := resp.State.Get(ctx, &got); diags.HasError() {
			t.Fatalf("State.Get errors: %v", diags.Errors())
		}
		if got.PrivateSpaceID.ValueString() != "ps-123" {
			t.Errorf("PrivateSpaceID = %q, want ps-123", got.PrivateSpaceID.ValueString())
		}
		if got.ID.ValueString() != "vpn-conn-456" {
			t.Errorf("ID = %q, want vpn-conn-456", got.ID.ValueString())
		}
	})

	t.Run("invalid ID (3-part) produces error", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "ps/conn/extra"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for 3-part ID")
		}
	})

	t.Run("empty parts produce error", func(t *testing.T) {
		req := resource.ImportStateRequest{ID: "/conn-id"}
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("ImportState() should error for empty parts")
		}
	})
}

// --- Read with error ---

func TestVPNConnectionResource_Read_Error(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/ps-1/connections/conn-1"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewVPNConnectionResource().(*VPNConnectionResource)
	res.client = &ch2client.VPNConnectionClient{
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
	objType := stateType.(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "conn-1"),
		"private_space_id": tftypes.NewValue(tftypes.String, "ps-1"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":            tftypes.NewValue(tftypes.String, "my-vpn"),
		"vpns":            tftypes.NewValue(objType.AttributeTypes["vpns"], nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// --- ModifyPlan ---

func TestVPNConnectionResource_ModifyPlan(t *testing.T) {
	r := NewVPNConnectionResource().(*VPNConnectionResource)
	ctx := context.Background()

	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)

	vpnsListType := objType.AttributeTypes["vpns"]
	buildState := func(vpnsVal tftypes.Value) tfsdk.State {
		raw := tftypes.NewValue(stateType, map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "conn-1"),
			"private_space_id": tftypes.NewValue(tftypes.String, "ps-1"),
			"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
			"name":            tftypes.NewValue(tftypes.String, "vpn"),
			"vpns":            vpnsVal,
		})
		return tfsdk.State{Schema: schemaResp.Schema, Raw: raw}
	}
	buildPlan := func(vpnsVal tftypes.Value) tfsdk.Plan {
		raw := tftypes.NewValue(stateType, map[string]tftypes.Value{
			"id":              tftypes.NewValue(tftypes.String, "conn-1"),
			"private_space_id": tftypes.NewValue(tftypes.String, "ps-1"),
			"organization_id": tftypes.NewValue(tftypes.String, "org-1"),
			"name":            tftypes.NewValue(tftypes.String, "vpn"),
			"vpns":            vpnsVal,
		})
		return tfsdk.Plan{Schema: schemaResp.Schema, Raw: raw}
	}

	t.Run("null state (create) is a no-op", func(t *testing.T) {
		nullState := tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)}
		req := resource.ModifyPlanRequest{
			State: nullState,
			Plan:  buildPlan(tftypes.NewValue(vpnsListType, nil)),
		}
		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ModifyPlan() unexpected errors on create: %v", resp.Diagnostics.Errors())
		}
		if len(resp.RequiresReplace) != 0 {
			t.Errorf("RequiresReplace should be empty on create, got %d", len(resp.RequiresReplace))
		}
	})

	t.Run("plan null vpns with state non-empty vpns requires replace", func(t *testing.T) {
		// Build a state with one VPN
		vpnsElemType := vpnsListType.(tftypes.List).ElementType
		stateVPNs := tftypes.NewValue(vpnsListType, []tftypes.Value{
			tftypes.NewValue(vpnsElemType, map[string]tftypes.Value{
				"local_asn":             tftypes.NewValue(tftypes.String, "65000"),
				"remote_asn":            tftypes.NewValue(tftypes.String, "65001"),
				"remote_ip_address":     tftypes.NewValue(tftypes.String, "1.2.3.4"),
				"static_routes":         tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
				"vpn_tunnels":           tftypes.NewValue(vpnsElemType.(tftypes.Object).AttributeTypes["vpn_tunnels"], nil),
				"name":                  tftypes.NewValue(tftypes.String, nil),
				"connection_name":       tftypes.NewValue(tftypes.String, nil),
				"vpn_connection_status": tftypes.NewValue(tftypes.String, nil),
				"vpn_id":                tftypes.NewValue(tftypes.String, nil),
				"connection_id":         tftypes.NewValue(tftypes.String, nil),
			}),
		})

		req := resource.ModifyPlanRequest{
			State: buildState(stateVPNs),
			Plan:  buildPlan(tftypes.NewValue(vpnsListType, nil)),
		}
		resp := &resource.ModifyPlanResponse{}
		r.ModifyPlan(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("ModifyPlan() unexpected errors: %v", resp.Diagnostics.Errors())
		}
		if len(resp.RequiresReplace) == 0 {
			t.Error("RequiresReplace should be set when plan removes all VPNs from non-empty state")
		}
	})
}

// --- vpnConnectionAPIResponseToResourceModel with port_config ---

func TestVPNConnectionAPIResponseToResourceModel_WithPortConfig(t *testing.T) {
	ctx := context.Background()
	vpnConn := &ch2client.VPNConnection{
		ID:   "vpn-id",
		Name: "test-vpn",
		VPNs: []ch2client.VPNResponse{},
	}
	vpnsList, _ := types.ListValueFrom(ctx,
		types.ObjectType{AttrTypes: getVPNAttrTypes()},
		[]VPNResourceModel{},
	)
	plan := &VPNConnectionResourceModel{
		ID:             types.StringValue("vpn-id"),
		PrivateSpaceID: types.StringValue("ps-1"),
		OrganizationID: types.StringValue("org-1"),
		Name:           types.StringValue("test-vpn"),
		VPNs:           vpnsList,
	}

	model, diags := vpnConnectionAPIResponseToResourceModel(ctx, vpnConn, plan)
	if diags.HasError() {
		t.Fatalf("unexpected diags: %v", diags.Errors())
	}
	if model.ID.ValueString() != "vpn-id" {
		t.Errorf("ID = %q, want vpn-id", model.ID.ValueString())
	}
}
