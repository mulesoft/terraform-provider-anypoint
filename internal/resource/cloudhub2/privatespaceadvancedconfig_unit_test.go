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

// --- buildRequest ---

func TestPrivateSpaceAdvancedConfigResource_buildRequest(t *testing.T) {
	r := &PrivateSpaceAdvancedConfigResource{}
	ctx := context.Background()

	t.Run("null ingress config produces minimal request", func(t *testing.T) {
		data := &PrivateSpaceAdvancedConfigResourceModel{
			EnableIAMRole:        types.BoolValue(true),
			IngressConfiguration: types.ObjectNull(nil),
		}
		req := r.buildRequest(ctx, data)
		if !req.EnableIAMRole {
			t.Error("EnableIAMRole should be true")
		}
	})

	t.Run("enable_iam_role false is preserved", func(t *testing.T) {
		data := &PrivateSpaceAdvancedConfigResourceModel{
			EnableIAMRole:        types.BoolValue(false),
			IngressConfiguration: types.ObjectNull(nil),
		}
		req := r.buildRequest(ctx, data)
		if req.EnableIAMRole {
			t.Error("EnableIAMRole should be false")
		}
	})
}

// --- Read with error ---

func TestPrivateSpaceAdvancedConfigResource_Read_Error(t *testing.T) {
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/ps-1"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewPrivateSpaceAdvancedConfigResource().(*PrivateSpaceAdvancedConfigResource)
	res.client = &ch2client.PrivateSpaceAdvancedConfigClient{
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
		"id":                    tftypes.NewValue(tftypes.String, "ps-1"),
		"private_space_id":      tftypes.NewValue(tftypes.String, "ps-1"),
		"organization_id":       tftypes.NewValue(tftypes.String, "test-org-id"),
		"ingress_configuration": tftypes.NewValue(objType.AttributeTypes["ingress_configuration"], nil),
		"enable_iam_role":       tftypes.NewValue(tftypes.Bool, false),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report error on 500")
	}
}

// --- mapPrivateSpaceToState ---

func TestPrivateSpaceAdvancedConfigResource_mapPrivateSpaceToState(t *testing.T) {
	r := &PrivateSpaceAdvancedConfigResource{}
	ctx := context.Background()

	t.Run("enable_iam_role is mapped", func(t *testing.T) {
		ps := &ch2client.PrivateSpace{
			EnableIAMRole: true,
			IngressConfiguration: ch2client.PrivateSpaceIngressConfig{
				ReadResponseTimeout: 300,
				Protocol:            "https-redirect",
				Logs: ch2client.PrivateSpaceIngressLogs{
					PortLogLevel: "ERROR",
					Filters:      []ch2client.PrivateSpaceLogFilter{},
				},
			},
		}
		data := &PrivateSpaceAdvancedConfigResourceModel{}
		r.mapPrivateSpaceToState(ctx, ps, data)
		if !data.EnableIAMRole.ValueBool() {
			t.Error("EnableIAMRole should be true")
		}
	})

	t.Run("ingress_configuration is populated", func(t *testing.T) {
		ps := &ch2client.PrivateSpace{
			EnableIAMRole: false,
			IngressConfiguration: ch2client.PrivateSpaceIngressConfig{
				ReadResponseTimeout: 60,
				Protocol:            "http",
				Logs: ch2client.PrivateSpaceIngressLogs{
					PortLogLevel: "DEBUG",
					Filters: []ch2client.PrivateSpaceLogFilter{
						{IP: "10.0.0.1", Level: "ERROR"},
					},
				},
			},
		}
		data := &PrivateSpaceAdvancedConfigResourceModel{}
		r.mapPrivateSpaceToState(ctx, ps, data)
		if data.IngressConfiguration.IsNull() {
			t.Error("IngressConfiguration should not be null")
		}
	})
}
