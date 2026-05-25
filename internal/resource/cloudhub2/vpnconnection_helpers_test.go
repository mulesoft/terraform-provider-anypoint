package cloudhub2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/attr"
	"github.com/hashicorp/terraform-plugin-framework/types"

	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// --- getVPNAttrTypes / getVPNTunnelAttrTypes ---

func TestGetVPNAttrTypes(t *testing.T) {
	attrTypes := getVPNAttrTypes()
	required := []string{
		"local_asn", "remote_asn", "remote_ip_address", "static_routes",
		"vpn_tunnels", "name", "connection_name", "vpn_connection_status",
		"vpn_id", "connection_id",
	}
	for _, key := range required {
		if _, ok := attrTypes[key]; !ok {
			t.Errorf("getVPNAttrTypes() missing key %q", key)
		}
	}
}

func TestGetVPNTunnelAttrTypes(t *testing.T) {
	attrTypes := getVPNTunnelAttrTypes()
	required := []string{"psk", "ptp_cidr", "startup_action", "is_logs_enabled"}
	for _, key := range required {
		if _, ok := attrTypes[key]; !ok {
			t.Errorf("getVPNTunnelAttrTypes() missing key %q", key)
		}
	}
}

// --- vpnConnectionAPIResponseToResourceModel ---

func TestVPNConnectionAPIResponseToResourceModel(t *testing.T) {
	ctx := context.Background()

	makeEmptyPlan := func(privateSpaceID string) *VPNConnectionResourceModel {
		vpnsList, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: getVPNAttrTypes()},
			[]VPNResourceModel{},
		)
		return &VPNConnectionResourceModel{
			ID:             types.StringValue(""),
			PrivateSpaceID: types.StringValue(privateSpaceID),
			OrganizationID: types.StringValue("org-1"),
			Name:           types.StringValue("my-vpn"),
			VPNs:           vpnsList,
		}
	}

	t.Run("empty vpn list from API is flattened", func(t *testing.T) {
		vpnConn := &ch2client.VPNConnection{
			ID:   "vpn-conn-id",
			Name: "my-vpn",
			VPNs: []ch2client.VPNResponse{},
		}
		plan := makeEmptyPlan("ps-1")
		model, diags := vpnConnectionAPIResponseToResourceModel(ctx, vpnConn, plan)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags.Errors())
		}
		if model.ID.ValueString() != "vpn-conn-id" {
			t.Errorf("ID = %q, want vpn-conn-id", model.ID.ValueString())
		}
		if model.Name.ValueString() != "my-vpn" {
			t.Errorf("Name = %q, want my-vpn", model.Name.ValueString())
		}
		if model.PrivateSpaceID.ValueString() != "ps-1" {
			t.Errorf("PrivateSpaceID = %q, want ps-1", model.PrivateSpaceID.ValueString())
		}
		var vpns []VPNResourceModel
		model.VPNs.ElementsAs(ctx, &vpns, false)
		if len(vpns) != 0 {
			t.Errorf("VPNs len = %d, want 0", len(vpns))
		}
	})

	t.Run("VPN with tunnels is mapped correctly", func(t *testing.T) {
		vpnConn := &ch2client.VPNConnection{
			ID:   "vpn-id-1",
			Name: "test-vpn",
			VPNs: []ch2client.VPNResponse{
				{
					Name:                "vpn-a",
					VPNID:               "vpn-a-id",
					ConnectionID:        "conn-a-id",
					ConnectionName:      "conn-a",
					VPNConnectionStatus: "AVAILABLE",
					LocalASN:            65000,
					RemoteASN:           65001,
					RemoteIPAddress:     "1.2.3.4",
					VPNTunnels: []ch2client.VPNConnectionTunnelResponse{
						{PSK: "psk1", PTPCidr: "169.254.0.0/30", StartupAction: "start", IsLogsEnabled: false},
					},
				},
			},
		}
		// plan should have corresponding VPN for static routes lookup
		tunnelObj, _ := types.ObjectValue(getVPNTunnelAttrTypes(), map[string]attr.Value{
			"psk":             types.StringValue("psk1"),
			"ptp_cidr":        types.StringValue("169.254.0.0/30"),
			"startup_action":  types.StringValue("start"),
			"is_logs_enabled": types.BoolValue(false),
		})
		tunnelsList, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: getVPNTunnelAttrTypes()},
			[]interface{}{tunnelObj},
		)
		staticRoutes, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/8"})
		vpnModel := VPNResourceModel{
			LocalASN:        types.StringValue("65000"),
			RemoteASN:       types.StringValue("65001"),
			RemoteIPAddress: types.StringValue("1.2.3.4"),
			StaticRoutes:    staticRoutes,
			VPNTunnels:      tunnelsList,
			Name:            types.StringValue(""),
			ConnectionName:  types.StringValue(""),
			VPNConnectionStatus: types.StringValue(""),
			VPNID:           types.StringValue(""),
			ConnectionID:    types.StringValue(""),
		}
		vpnsList, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: getVPNAttrTypes()},
			[]VPNResourceModel{vpnModel},
		)
		plan := &VPNConnectionResourceModel{
			ID:             types.StringValue(""),
			PrivateSpaceID: types.StringValue("ps-1"),
			OrganizationID: types.StringValue("org-1"),
			Name:           types.StringValue("test-vpn"),
			VPNs:           vpnsList,
		}

		model, diags := vpnConnectionAPIResponseToResourceModel(ctx, vpnConn, plan)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags.Errors())
		}

		var vpns []VPNResourceModel
		model.VPNs.ElementsAs(ctx, &vpns, false)
		if len(vpns) != 1 {
			t.Fatalf("VPNs len = %d, want 1", len(vpns))
		}
		vpn := vpns[0]
		if vpn.RemoteIPAddress.ValueString() != "1.2.3.4" {
			t.Errorf("RemoteIPAddress = %q, want 1.2.3.4", vpn.RemoteIPAddress.ValueString())
		}
		if vpn.VPNID.ValueString() != "vpn-a-id" {
			t.Errorf("VPNID = %q, want vpn-a-id", vpn.VPNID.ValueString())
		}
		if vpn.VPNConnectionStatus.ValueString() != "AVAILABLE" {
			t.Errorf("VPNConnectionStatus = %q, want AVAILABLE", vpn.VPNConnectionStatus.ValueString())
		}

		// static routes should be carried from plan
		var routes []string
		vpn.StaticRoutes.ElementsAs(ctx, &routes, false)
		if len(routes) != 1 || routes[0] != "10.0.0.0/8" {
			t.Errorf("StaticRoutes = %v, want [10.0.0.0/8]", routes)
		}
	})
}

// --- vpnConnectionResourceModelToAPIRequest ---

func TestVPNConnectionResourceModelToAPIRequest(t *testing.T) {
	ctx := context.Background()

	t.Run("basic model is converted to API request", func(t *testing.T) {
		tunnelModel := VPNTunnelResourceModel{
			PSK:           types.StringValue("my-psk"),
			PTPCidr:       types.StringValue("169.254.1.0/30"),
			StartupAction: types.StringValue("start"),
			IsLogsEnabled: types.BoolValue(false),
		}
		tunnelsList, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: getVPNTunnelAttrTypes()},
			[]VPNTunnelResourceModel{tunnelModel},
		)
		staticRoutes, _ := types.ListValueFrom(ctx, types.StringType, []string{"192.168.0.0/16"})
		vpnModel := VPNResourceModel{
			LocalASN:            types.StringValue("65000"),
			RemoteASN:           types.StringValue("65001"),
			RemoteIPAddress:     types.StringValue("203.0.113.1"),
			StaticRoutes:        staticRoutes,
			VPNTunnels:          tunnelsList,
			Name:                types.StringValue(""),
			ConnectionName:      types.StringValue(""),
			VPNConnectionStatus: types.StringValue(""),
			VPNID:               types.StringValue(""),
			ConnectionID:        types.StringValue(""),
		}
		vpnsList, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: getVPNAttrTypes()},
			[]VPNResourceModel{vpnModel},
		)
		data := &VPNConnectionResourceModel{
			ID:             types.StringValue(""),
			PrivateSpaceID: types.StringValue("ps-1"),
			OrganizationID: types.StringValue("org-1"),
			Name:           types.StringValue("my-vpn-conn"),
			VPNs:           vpnsList,
		}

		req, diags := vpnConnectionResourceModelToAPIRequest(ctx, data)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags.Errors())
		}
		if req.Name != "my-vpn-conn" {
			t.Errorf("Name = %q, want my-vpn-conn", req.Name)
		}
		if len(req.VPNs) != 1 {
			t.Fatalf("VPNs len = %d, want 1", len(req.VPNs))
		}
		vpn := req.VPNs[0]
		if vpn.LocalASN != "65000" {
			t.Errorf("LocalASN = %q, want 65000", vpn.LocalASN)
		}
		if vpn.RemoteIPAddress != "203.0.113.1" {
			t.Errorf("RemoteIPAddress = %q, want 203.0.113.1", vpn.RemoteIPAddress)
		}
		if len(vpn.StaticRoutes) != 1 || vpn.StaticRoutes[0] != "192.168.0.0/16" {
			t.Errorf("StaticRoutes = %v, want [192.168.0.0/16]", vpn.StaticRoutes)
		}
		if len(vpn.VPNTunnels) != 1 {
			t.Fatalf("VPNTunnels len = %d, want 1", len(vpn.VPNTunnels))
		}
		if vpn.VPNTunnels[0].PSK != "my-psk" {
			t.Errorf("Tunnel PSK = %q, want my-psk", vpn.VPNTunnels[0].PSK)
		}
		if vpn.VPNTunnels[0].StartupAction != "start" {
			t.Errorf("Tunnel StartupAction = %q, want start", vpn.VPNTunnels[0].StartupAction)
		}
	})

	t.Run("null static routes are omitted", func(t *testing.T) {
		tunnelsList, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: getVPNTunnelAttrTypes()},
			[]VPNTunnelResourceModel{},
		)
		vpnModel := VPNResourceModel{
			LocalASN:            types.StringValue("65000"),
			RemoteASN:           types.StringValue("65001"),
			RemoteIPAddress:     types.StringValue("1.1.1.1"),
			StaticRoutes:        types.ListNull(types.StringType),
			VPNTunnels:          tunnelsList,
			Name:                types.StringValue(""),
			ConnectionName:      types.StringValue(""),
			VPNConnectionStatus: types.StringValue(""),
			VPNID:               types.StringValue(""),
			ConnectionID:        types.StringValue(""),
		}
		vpnsList, _ := types.ListValueFrom(ctx,
			types.ObjectType{AttrTypes: getVPNAttrTypes()},
			[]VPNResourceModel{vpnModel},
		)
		data := &VPNConnectionResourceModel{
			ID:             types.StringValue(""),
			PrivateSpaceID: types.StringValue("ps-1"),
			OrganizationID: types.StringValue("org-1"),
			Name:           types.StringValue("test"),
			VPNs:           vpnsList,
		}
		req, diags := vpnConnectionResourceModelToAPIRequest(ctx, data)
		if diags.HasError() {
			t.Fatalf("unexpected diags: %v", diags.Errors())
		}
		if len(req.VPNs[0].StaticRoutes) != 0 {
			t.Errorf("StaticRoutes = %v, want empty", req.VPNs[0].StaticRoutes)
		}
	})
}
