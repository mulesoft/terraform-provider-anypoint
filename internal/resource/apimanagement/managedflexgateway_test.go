package apimanagement

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	apimgmtclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewManagedFlexGatewayResource(t *testing.T) {
	r := NewManagedFlexGatewayResource()
	if r == nil {
		t.Error("NewManagedFlexGatewayResource() returned nil")
	}
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("ManagedFlexGatewayResource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("ManagedFlexGatewayResource should implement ResourceWithImportState")
	}
}

func TestManagedFlexGatewayResource_Metadata(t *testing.T) {
	r := NewManagedFlexGatewayResource()
	testutil.TestResourceMetadata(t, r, "_managed_flexgateway")
}

func TestManagedFlexGatewayResource_Schema(t *testing.T) {
	r := NewManagedFlexGatewayResource()
	requiredAttrs := []string{"name", "environment_id", "target_id"}
	optionalAttrs := []string{"organization_id", "runtime_version", "release_channel", "size", "ingress", "properties", "logging", "tracing"}
	computedAttrs := []string{"id", "status", "organization_id"}
	testutil.TestResourceSchema(t, r, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestManagedFlexGatewayResource_Configure(t *testing.T) {
	res := NewManagedFlexGatewayResource().(*ManagedFlexGatewayResource)
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

func TestManagedFlexGatewayResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewManagedFlexGatewayResource().(*ManagedFlexGatewayResource)
	ctx := context.Background()
	req := resource.ConfigureRequest{ProviderData: "invalid"}
	resp := &resource.ConfigureResponse{}
	res.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestManagedFlexGatewayResource_ImportState(t *testing.T) {
	r := NewManagedFlexGatewayResource()
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource does not implement ImportState")
	}
}

func TestManagedFlexGatewayResourceModel_Validation(t *testing.T) {
	model := ManagedFlexGatewayResourceModel{}
	_ = model.ID
	_ = model.Name
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.TargetID
	_ = model.RuntimeVersion
	_ = model.ReleaseChannel
	_ = model.Size
	_ = model.Status
	_ = model.Ingress
	_ = model.Properties
	_ = model.Logging
	_ = model.Tracing
}

func TestManagedFlexGatewayResource_Read(t *testing.T) {
	basePath := "/gatewaymanager/xapi/v1/organizations/test-org-id/environments/test-env-id/gateways/test-gw-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "test-gw-id",
				"name":           "test-gateway",
				"targetId":       "test-target-id",
				"runtimeVersion": "1.6.0",
				"releaseChannel": "LTS",
				"size":           "SMALL",
				"status":         "Active",
				"configuration": map[string]interface{}{
					"ingress":    map[string]interface{}{"publicUrl": "", "internalUrl": ""},
					"properties": map[string]interface{}{},
					"logging":    map[string]interface{}{},
					"tracing":    map[string]interface{}{},
				},
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedFlexGatewayResource().(*ManagedFlexGatewayResource)
	res.client = &apimgmtclient.ManagedFlexGatewayClient{
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
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-gw-id"),
		"name":            tftypes.NewValue(tftypes.String, "test-gateway"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"target_id":       tftypes.NewValue(tftypes.String, "test-target-id"),
		"runtime_version": tftypes.NewValue(tftypes.String, "1.6.0"),
		"release_channel": tftypes.NewValue(tftypes.String, "LTS"),
		"size":            tftypes.NewValue(tftypes.String, "SMALL"),
		"status":          tftypes.NewValue(tftypes.String, "Active"),
		"ingress":         tftypes.NewValue(ingressObjType, nil),
		"properties":      tftypes.NewValue(propertiesObjType, nil),
		"logging":         tftypes.NewValue(loggingObjType, nil),
		"tracing":         tftypes.NewValue(tracingObjType, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got ManagedFlexGatewayResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Name.ValueString() != "test-gateway" {
		t.Errorf("Expected Name 'test-gateway', got %s", got.Name.ValueString())
	}
}

func TestManagedFlexGatewayResource_Read_NotFound(t *testing.T) {
	basePath := "/gatewaymanager/xapi/v1/organizations/test-org-id/environments/test-env-id/gateways/test-gw-id"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewManagedFlexGatewayResource().(*ManagedFlexGatewayResource)
	res.client = &apimgmtclient.ManagedFlexGatewayClient{
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
	ingressObjType := objType.AttributeTypes["ingress"].(tftypes.Object)
	propertiesObjType := objType.AttributeTypes["properties"].(tftypes.Object)
	loggingObjType := objType.AttributeTypes["logging"].(tftypes.Object)
	tracingObjType := objType.AttributeTypes["tracing"].(tftypes.Object)

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "test-gw-id"),
		"name":            tftypes.NewValue(tftypes.String, "test-gateway"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"target_id":       tftypes.NewValue(tftypes.String, "test-target-id"),
		"runtime_version": tftypes.NewValue(tftypes.String, "1.6.0"),
		"release_channel": tftypes.NewValue(tftypes.String, "LTS"),
		"size":            tftypes.NewValue(tftypes.String, "SMALL"),
		"status":          tftypes.NewValue(tftypes.String, "Active"),
		"ingress":         tftypes.NewValue(ingressObjType, nil),
		"properties":      tftypes.NewValue(propertiesObjType, nil),
		"logging":         tftypes.NewValue(loggingObjType, nil),
		"tracing":         tftypes.NewValue(tracingObjType, nil),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if !resp.State.Raw.IsNull() {
		t.Error("Read() for 404 should remove resource (state should be null)")
	}
}
