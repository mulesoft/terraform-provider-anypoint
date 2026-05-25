package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestManagedOmniGatewaySingleDataSource_Read(t *testing.T) {
	basePath := "/gatewaymanager/xapi/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-abc"

	mockGW := &apimgmtclient.ManagedOmniGateway{
		ID:             "gw-abc",
		Name:           "my-gateway",
		TargetID:       "target-1",
		TargetName:     "Private Space 1",
		TargetType:     "private-space",
		RuntimeVersion: "1.6.0",
		ReleaseChannel: "LTS",
		Size:           "small",
		Status:         "Running",
		DesiredStatus:  "Running",
		StatusMessage:  "",
		DateCreated:    "2024-01-01",
		LastUpdated:    "2024-06-01",
		APILimit:       100,
		Configuration: apimgmtclient.ManagedOmniGatewayConfig{
			Ingress: apimgmtclient.IngressConfig{
				PublicURL:         "https://pub.example.com",
				InternalURL:       "https://int1.example.com, https://int2.example.com",
				ForwardSSLSession: true,
				LastMileSecurity:  false,
			},
			Properties: apimgmtclient.PropertiesConfig{
				UpstreamResponseTimeout: 30000,
				ConnectionIdleTimeout:   60000,
			},
			Logging: apimgmtclient.LoggingConfig{
				Level:       "INFO",
				ForwardLogs: false,
			},
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, mockGW)
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewManagedOmniGatewaySingleDataSource().(*ManagedOmniGatewaySingleDataSource)
	ds.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "gw-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"target_id":       tftypes.NewValue(tftypes.String, nil),
		"target_name":     tftypes.NewValue(tftypes.String, nil),
		"target_type":     tftypes.NewValue(tftypes.String, nil),
		"runtime_version": tftypes.NewValue(tftypes.String, nil),
		"release_channel": tftypes.NewValue(tftypes.String, nil),
		"size":            tftypes.NewValue(tftypes.String, nil),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"desired_status":  tftypes.NewValue(tftypes.String, nil),
		"status_message":  tftypes.NewValue(tftypes.String, nil),
		"date_created":    tftypes.NewValue(tftypes.String, nil),
		"last_updated":    tftypes.NewValue(tftypes.String, nil),
		"api_limit":       tftypes.NewValue(tftypes.Number, nil),
		"ingress":         tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["ingress"], nil),
		"properties":      tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["properties"], nil),
		"logging":         tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["logging"], nil),
		"port_configuration": tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["port_configuration"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got ManagedOmniGatewaySingleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "my-gateway" {
		t.Errorf("Name = %q, want my-gateway", got.Name.ValueString())
	}
	if got.Status.ValueString() != "Running" {
		t.Errorf("Status = %q, want Running", got.Status.ValueString())
	}
	if got.APILimit.ValueInt64() != 100 {
		t.Errorf("APILimit = %d, want 100", got.APILimit.ValueInt64())
	}
}

func TestManagedOmniGatewaySingleDataSource_Read_Error(t *testing.T) {
	basePath := "/apimanager/xapi/v1/organizations/test-org-id/environments/test-env-id/flexGateway/gw-bad"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewManagedOmniGatewaySingleDataSource().(*ManagedOmniGatewaySingleDataSource)
	ds.client = &apimgmtclient.ManagedOmniGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "gw-bad"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"target_id":       tftypes.NewValue(tftypes.String, nil),
		"target_name":     tftypes.NewValue(tftypes.String, nil),
		"target_type":     tftypes.NewValue(tftypes.String, nil),
		"runtime_version": tftypes.NewValue(tftypes.String, nil),
		"release_channel": tftypes.NewValue(tftypes.String, nil),
		"size":            tftypes.NewValue(tftypes.String, nil),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"desired_status":  tftypes.NewValue(tftypes.String, nil),
		"status_message":  tftypes.NewValue(tftypes.String, nil),
		"date_created":    tftypes.NewValue(tftypes.String, nil),
		"last_updated":    tftypes.NewValue(tftypes.String, nil),
		"api_limit":       tftypes.NewValue(tftypes.Number, nil),
		"ingress":         tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["ingress"], nil),
		"properties":      tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["properties"], nil),
		"logging":         tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["logging"], nil),
		"port_configuration": tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["port_configuration"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report errors on API failure")
	}
}
