package agentstools

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	agentsclient "github.com/mulesoft/terraform-provider-anypoint/internal/client/agentstools"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/apimanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const (
	brOrg = "test-org"
	brEnv = "test-env"
	brID  = 21058094
)

func newBridgeResourceWithMock(t *testing.T, extra map[string]func(w http.ResponseWriter, r *http.Request)) *MCPBridgeResource {
	t.Helper()
	handlers := testutil.StandardMockHandlers()
	for k, v := range extra {
		handlers[k] = v
	}
	server := testutil.MockHTTPServer(t, handlers)
	cfg := &anypointclient.Config{BaseURL: server.URL, ClientID: "cid", ClientSecret: "cs"}
	c, err := agentsclient.NewMCPBridgeClient(cfg)
	if err != nil {
		t.Fatalf("NewMCPBridgeClient() error = %v", err)
	}
	res := NewMCPBridgeResource().(*MCPBridgeResource)
	res.client = c
	return res
}

// petstoreSources mirrors the two source APIs of the live MCP-bridge-test capture.
func petstoreSources() []bridgeSource {
	return []bridgeSource{
		{Label: "tf-fresh-rest-api", UpstreamURI: "https://sandbox.example.com/petstore/v1",
			AssetID: "tf-fresh-rest-api", GroupID: brOrg, Version: "1.0.0", Tools: []bridgeTool{
				{Method: "GET", Path: "/pets", Description: "Get pets"},
				{Method: "POST", Path: "/pets", HasBody: true},
				{Method: "GET", Path: "/pets/{petId}"},
			}},
		{Label: "tf-ref-petstore-mv", UpstreamURI: "http://example.com",
			AssetID: "tf-ref-petstore-mv", GroupID: brOrg, Version: "2.0.0", Tools: []bridgeTool{
				{Method: "GET", Path: "/pets/{petId}/vaccinations"},
			}},
	}
}

func liveUpstreams() []agentsclient.MCPBridgeUpstreamDetail {
	c1 := &agentsclient.MCPBridgeConnection{AssetID: "tf-fresh-rest-api", GroupID: brOrg, Version: "1.0.0"}
	c2 := &agentsclient.MCPBridgeConnection{AssetID: "tf-ref-petstore-mv", GroupID: brOrg, Version: "2.0.0"}
	return []agentsclient.MCPBridgeUpstreamDetail{
		{ID: "dde8b791", URI: "https://sandbox.example.com/petstore/v1", Connection: c1},
		{ID: "cdbd6d67", URI: "http://example.com", Connection: c2},
	}
}

// --- attachBridgePolicies: the 5-policy layout matches the live capture ---

func TestMCPBridge_attachBridgePolicies(t *testing.T) {
	inboundPath := "/apimanager/api/v1/organizations/" + brOrg + "/environments/" + brEnv + "/apis/21058094/policies"
	outboundPath := "/apimanager/xapi/v1/organizations/" + brOrg + "/environments/" + brEnv + "/apis/21058094/policies/outbound-policies"

	var inbound []map[string]interface{}
	var outbound []map[string]interface{}

	res := newBridgeResourceWithMock(t, map[string]func(w http.ResponseWriter, r *http.Request){
		inboundPath: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			inbound = append(inbound, body)
			testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{"id": 1000 + len(inbound)})
		},
		outboundPath: func(w http.ResponseWriter, r *http.Request) {
			var body map[string]interface{}
			_ = json.NewDecoder(r.Body).Decode(&body)
			outbound = append(outbound, body)
			testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{"id": 2000 + len(outbound)})
		},
	})

	err := res.attachBridgePolicies(context.Background(), brOrg, brEnv, brID, petstoreSources(), liveUpstreams())
	if err != nil {
		t.Fatalf("attachBridgePolicies() error = %v", err)
	}

	// 3 inbound: mcp-support, mcp-schema-validation, mcp-transcoding-router.
	if len(inbound) != 3 {
		t.Fatalf("inbound policy count = %d, want 3", len(inbound))
	}
	if inbound[0]["assetId"] != "mcp-support" || inbound[0]["assetVersion"] != "1.0.1" {
		t.Errorf("policy 1 = %v, want mcp-support 1.0.1", inbound[0])
	}
	if inbound[1]["assetId"] != "mcp-schema-validation" || inbound[1]["assetVersion"] != "1.1.1" {
		t.Errorf("policy 2 = %v, want mcp-schema-validation 1.1.1", inbound[1])
	}
	if cfg := inbound[1]["configurationData"].(map[string]interface{}); cfg["validateToolSchema"] != true {
		t.Errorf("schema-validation config = %v, want validateToolSchema:true", cfg)
	}
	if inbound[2]["assetId"] != "mcp-transcoding-router" || inbound[2]["assetVersion"] != "1.0.1" {
		t.Errorf("policy 3 = %v, want mcp-transcoding-router 1.0.1", inbound[2])
	}
	routerCfg := inbound[2]["configurationData"].(map[string]interface{})
	if routerCfg["transcodingPath"] != "/mcp" {
		t.Errorf("router transcodingPath = %v", routerCfg["transcodingPath"])
	}
	if routes := routerCfg["routes"].([]interface{}); len(routes) != 2 {
		t.Errorf("router routes len = %d, want 2", len(routes))
	}

	// 2 outbound mcp-transcoding, one per upstream, each with the correct upstreamId.
	if len(outbound) != 2 {
		t.Fatalf("outbound policy count = %d, want 2", len(outbound))
	}
	for _, o := range outbound {
		if o["assetId"] != "mcp-transcoding" || o["assetVersion"] != "1.0.0" {
			t.Errorf("outbound policy = %v, want mcp-transcoding 1.0.0", o)
		}
		if apiVer, _ := o["apiVersionId"].(float64); int(apiVer) != brID {
			t.Errorf("outbound apiVersionId = %v, want %d", o["apiVersionId"], brID)
		}
		ups := o["upstreamIds"].([]interface{})
		if len(ups) != 1 || (ups[0] != "dde8b791" && ups[0] != "cdbd6d67") {
			t.Errorf("outbound upstreamIds = %v", ups)
		}
	}
}

// --- parseBridgeProxyURI (import port/base_path recovery) ---

func TestParseBridgeProxyURI(t *testing.T) {
	cases := []struct {
		raw      string
		port     int64
		basePath string
		ok       bool
	}{
		{"http://0.0.0.0:8081/tflife", 8081, "tflife", true},
		{"http://0.0.0.0:8081/", 8081, "", true},
		{"http://0.0.0.0:9090/a/b", 9090, "a/b", true},
		{"http://0.0.0.0/nope", 0, "", false}, // no port
	}
	for _, c := range cases {
		port, bp, ok := parseBridgeProxyURI(c.raw)
		if ok != c.ok || (ok && (port != c.port || bp != c.basePath)) {
			t.Errorf("parseBridgeProxyURI(%q) = (%d,%q,%v), want (%d,%q,%v)", c.raw, port, bp, ok, c.port, c.basePath, c.ok)
		}
	}
}

// --- bridgeToolSignature (ModifyPlan trigger) ---

func TestBridgeToolSignature(t *testing.T) {
	base := petstoreSources()
	if bridgeToolSignature(base) != bridgeToolSignature(petstoreSources()) {
		t.Fatal("signature should be stable for identical tool sets")
	}
	// Adding a tool changes the signature (triggers asset_version re-plan).
	withExtra := petstoreSources()
	withExtra[0].Tools = append(withExtra[0].Tools, bridgeTool{Method: "DELETE", Path: "/pets/{petId}"})
	if bridgeToolSignature(base) == bridgeToolSignature(withExtra) {
		t.Error("adding a tool must change the signature")
	}
	// Editing only a description also changes the signature.
	descEdit := petstoreSources()
	descEdit[0].Tools[0].Description = "changed"
	if bridgeToolSignature(base) == bridgeToolSignature(descEdit) {
		t.Error("editing a tool description must change the signature")
	}
	// Structural-only change (upstream_uri) must NOT change the tool signature.
	structOnly := petstoreSources()
	structOnly[0].UpstreamURI = "https://elsewhere.example.com"
	if bridgeToolSignature(base) != bridgeToolSignature(structOnly) {
		t.Error("changing only upstream_uri must not change the tool signature")
	}
}

// --- backfillBridgeImportFields ---

func TestBackfillBridgeImportFields(t *testing.T) {
	proxy := "http://0.0.0.0:8081/tflife"
	inst := &agentsclient.MCPBridge{
		AssetID:    "tf-bridge-lifecycle",
		Endpoint:   &agentsclient.MCPBridgeEndpoint{ProxyURI: &proxy},
		Deployment: &agentsclient.MCPBridgeDeployment{TargetID: "gw-uuid-123"},
	}
	data := &MCPBridgeResourceModel{
		GatewayID:    types.StringNull(),
		MCPAssetName: types.StringNull(),
		Port:         types.Int64Null(),
		BasePath:     types.StringNull(),
	}
	backfillBridgeImportFields(data, inst)
	if data.GatewayID.ValueString() != "gw-uuid-123" {
		t.Errorf("gateway_id = %q, want gw-uuid-123", data.GatewayID.ValueString())
	}
	if data.MCPAssetName.ValueString() != "tf-bridge-lifecycle" {
		t.Errorf("mcp_asset_name = %q, want tf-bridge-lifecycle", data.MCPAssetName.ValueString())
	}
	if data.Port.ValueInt64() != 8081 {
		t.Errorf("port = %d, want 8081", data.Port.ValueInt64())
	}
	if data.BasePath.ValueString() != "tflife" {
		t.Errorf("base_path = %q, want tflife", data.BasePath.ValueString())
	}

	// Existing (non-empty) values must be preserved, not overwritten.
	data2 := &MCPBridgeResourceModel{
		GatewayID:    types.StringValue("keep-gw"),
		MCPAssetName: types.StringValue("keep-name"),
		Port:         types.Int64Value(9000),
		BasePath:     types.StringValue("keep"),
	}
	backfillBridgeImportFields(data2, inst)
	if data2.GatewayID.ValueString() != "keep-gw" || data2.MCPAssetName.ValueString() != "keep-name" ||
		data2.Port.ValueInt64() != 9000 || data2.BasePath.ValueString() != "keep" {
		t.Errorf("backfill overwrote existing values: %+v", data2)
	}
}

// --- matchUpstreamID ---

func TestMatchUpstreamID(t *testing.T) {
	ups := liveUpstreams()
	src := bridgeSource{Label: "x", UpstreamURI: "http://example.com", AssetID: "tf-ref-petstore-mv", Version: "2.0.0"}
	if got := matchUpstreamID(ups, src); got != "cdbd6d67" {
		t.Errorf("matchUpstreamID by connection = %q, want cdbd6d67", got)
	}
	// URI fallback when connection asset doesn't match.
	src2 := bridgeSource{UpstreamURI: "https://sandbox.example.com/petstore/v1", AssetID: "unknown", Version: "9"}
	if got := matchUpstreamID(ups, src2); got != "dde8b791" {
		t.Errorf("matchUpstreamID by uri fallback = %q, want dde8b791", got)
	}
}

// --- flattenBridge ---

func TestMCPBridge_flattenBridge(t *testing.T) {
	r := &MCPBridgeResource{}
	inst := &agentsclient.MCPBridge{
		ID:             brID,
		AssetID:        "MCP-bridge-test",
		AssetVersion:   "1.0.0",
		ProductVersion: "v1.0",
		Status:         "inactive",
		Deployment: &agentsclient.MCPBridgeDeployment{
			EnvironmentID: brEnv, Type: "HY", ExpectedStatus: "deployed",
			TargetID: "c3709c68", TargetName: "tf-smg-onefile", GatewayVersion: "1.13.3",
		},
	}
	data := &MCPBridgeResourceModel{
		OrganizationID: types.StringNull(),
		ProductVersion: types.StringUnknown(),
		Status:         types.StringUnknown(),
	}
	r.flattenBridge(inst, data, brOrg, brEnv)

	if data.ID.ValueString() != "21058094" {
		t.Errorf("ID = %q", data.ID.ValueString())
	}
	if data.Technology.ValueString() != "flexGateway" {
		t.Errorf("Technology = %q, want flexGateway", data.Technology.ValueString())
	}
	if data.AssetVersion.ValueString() != "1.0.0" {
		t.Errorf("AssetVersion = %q", data.AssetVersion.ValueString())
	}
	if data.OrganizationID.ValueString() != brOrg {
		t.Errorf("OrganizationID = %q, want %s", data.OrganizationID.ValueString(), brOrg)
	}
	if data.Deployment.IsNull() {
		t.Fatal("Deployment should not be null")
	}
	depAttrs := data.Deployment.Attributes()
	if depAttrs["target_name"].(types.String).ValueString() != "tf-smg-onefile" {
		t.Errorf("deployment target_name = %v", depAttrs["target_name"])
	}
}

// --- ImportState ---

func TestMCPBridge_ImportState(t *testing.T) {
	r := NewMCPBridgeResource().(*MCPBridgeResource)
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	rawState := tftypes.NewValue(stateType, nil)

	t.Run("valid 3-part", func(t *testing.T) {
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, resource.ImportStateRequest{ID: "org-1/env-2/21058094"}, resp)
		if resp.Diagnostics.HasError() {
			t.Fatalf("ImportState errors: %v", resp.Diagnostics.Errors())
		}
		var got MCPBridgeResourceModel
		resp.State.Get(ctx, &got)
		if got.ID.ValueString() != "21058094" || got.EnvironmentID.ValueString() != "env-2" {
			t.Errorf("import parsed wrong: id=%q env=%q", got.ID.ValueString(), got.EnvironmentID.ValueString())
		}
	})

	t.Run("invalid 2-part", func(t *testing.T) {
		resp := &resource.ImportStateResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: rawState}}
		r.ImportState(ctx, resource.ImportStateRequest{ID: "only/two"}, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected error for 2-part import ID")
		}
	})
}

// --- ValidateConfig ---

func buildBridgeValidateReq(t *testing.T, r *MCPBridgeResource, sourceAPIs tftypes.Value) resource.ValidateConfigRequest {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)

	vals := map[string]tftypes.Value{
		"id":                tftypes.NewValue(tftypes.String, nil),
		"organization_id":   tftypes.NewValue(tftypes.String, nil),
		"environment_id":    tftypes.NewValue(tftypes.String, "env-1"),
		"gateway_id":        tftypes.NewValue(tftypes.String, "gw-1"),
		"mcp_asset_name":    tftypes.NewValue(tftypes.String, "bridge"),
		"port":              tftypes.NewValue(tftypes.Number, nil),
		"base_path":         tftypes.NewValue(tftypes.String, nil),
		"asset_id":          tftypes.NewValue(tftypes.String, nil),
		"asset_version":     tftypes.NewValue(tftypes.String, nil),
		"product_version":   tftypes.NewValue(tftypes.String, nil),
		"consumer_endpoint": tftypes.NewValue(tftypes.String, nil),
		"status":            tftypes.NewValue(tftypes.String, nil),
		"technology":        tftypes.NewValue(tftypes.String, nil),
		"deployment":        tftypes.NewValue(objType.AttributeTypes["deployment"], nil),
		"source_apis":       sourceAPIs,
	}
	raw := tftypes.NewValue(stateType, vals)
	return resource.ValidateConfigRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: raw}}
}

func sourceAPIsValue(t *testing.T, r *MCPBridgeResource, labels []string, toolsPerSource int) tftypes.Value {
	t.Helper()
	ctx := context.Background()
	schemaResp := &resource.SchemaResponse{}
	r.Schema(ctx, resource.SchemaRequest{}, schemaResp)
	objType := schemaResp.Schema.Type().TerraformType(ctx).(tftypes.Object)
	srcElem := objType.AttributeTypes["source_apis"].(tftypes.List).ElementType.(tftypes.Object)
	toolElem := srcElem.AttributeTypes["tools"].(tftypes.List).ElementType.(tftypes.Object)

	buildTool := func(method, p string) tftypes.Value {
		return tftypes.NewValue(toolElem, map[string]tftypes.Value{
			"method":        tftypes.NewValue(tftypes.String, method),
			"path":          tftypes.NewValue(tftypes.String, p),
			"name":          tftypes.NewValue(tftypes.String, nil),
			"description":   tftypes.NewValue(tftypes.String, nil),
			"query_params":  tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"header_params": tftypes.NewValue(tftypes.List{ElementType: tftypes.String}, nil),
			"has_body":      tftypes.NewValue(tftypes.Bool, false),
		})
	}

	srcVals := make([]tftypes.Value, 0, len(labels))
	for _, lbl := range labels {
		toolVals := make([]tftypes.Value, 0, toolsPerSource)
		for i := 0; i < toolsPerSource; i++ {
			toolVals = append(toolVals, buildTool("GET", "/pets"))
		}
		var toolsList tftypes.Value
		if toolsPerSource == 0 {
			toolsList = tftypes.NewValue(tftypes.List{ElementType: toolElem}, []tftypes.Value{})
		} else {
			toolsList = tftypes.NewValue(tftypes.List{ElementType: toolElem}, toolVals)
		}
		srcVals = append(srcVals, tftypes.NewValue(srcElem, map[string]tftypes.Value{
			"label":        tftypes.NewValue(tftypes.String, lbl),
			"upstream_uri": tftypes.NewValue(tftypes.String, "https://backend"),
			"asset_id":     tftypes.NewValue(tftypes.String, lbl),
			"group_id":     tftypes.NewValue(tftypes.String, nil),
			"version":      tftypes.NewValue(tftypes.String, "1.0.0"),
			"tools":        toolsList,
		}))
	}
	return tftypes.NewValue(tftypes.List{ElementType: srcElem}, srcVals)
}

func TestMCPBridge_ValidateConfig(t *testing.T) {
	r := NewMCPBridgeResource().(*MCPBridgeResource)
	ctx := context.Background()

	t.Run("clean config passes", func(t *testing.T) {
		req := buildBridgeValidateReq(t, r, sourceAPIsValue(t, r, []string{"a", "b"}, 2))
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if resp.Diagnostics.HasError() {
			t.Errorf("unexpected errors: %v", resp.Diagnostics.Errors())
		}
	})

	t.Run("duplicate labels error", func(t *testing.T) {
		req := buildBridgeValidateReq(t, r, sourceAPIsValue(t, r, []string{"dup", "dup"}, 1))
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected duplicate-label error")
		}
	})

	t.Run("source with no tools error", func(t *testing.T) {
		req := buildBridgeValidateReq(t, r, sourceAPIsValue(t, r, []string{"a"}, 0))
		resp := &resource.ValidateConfigResponse{}
		r.ValidateConfig(ctx, req, resp)
		if !resp.Diagnostics.HasError() {
			t.Error("expected no-tools error")
		}
	})
}

// TestBridgeToolMappingFromPolicies_RouterWithoutUpstreamIDs locks the import
// reconstruction fix: the policies LIST endpoint returns the mcp-transcoding tool
// DEFINITIONS but drops upstreamIds for outbound policies, so tools must be mapped
// to their source via the mcp-transcoding-router `routes` (label -> tool names),
// NOT via upstreamIds. Before the fix, nil upstreamIds meant every transcoding
// policy was skipped and import generated `tools = []` (caught only by a live
// config-driven import). Two source APIs verify per-label routing, not a fallback.
func TestBridgeToolMappingFromPolicies_RouterWithoutUpstreamIDs(t *testing.T) {
	toolDef := func(name, method, path string, q ...string) map[string]interface{} {
		qp := []interface{}{}
		for _, k := range q {
			qp = append(qp, map[string]interface{}{"key": k, "value": "#[vars.params['" + k + "']]"})
		}
		return map[string]interface{}{"name": name, "method": method, "path": path, "queryParams": qp}
	}
	policies := []apimanagement.APIPolicy{
		// Inbound policies carry no tool data and must be ignored.
		{AssetID: "mcp-support", ConfigurationData: map[string]interface{}{}},
		{AssetID: "mcp-schema-validation", ConfigurationData: map[string]interface{}{"validateToolSchema": true}},
		// Router: authoritative label -> tool-name mapping (no upstreamIds needed).
		{AssetID: "mcp-transcoding-router", ConfigurationData: map[string]interface{}{
			"transcodingPath": "/mcp",
			"routes": []interface{}{
				map[string]interface{}{"upstreamName": "svc-a", "tools": []interface{}{"get_pets", "post_pets"}},
				map[string]interface{}{"upstreamName": "svc-b", "tools": []interface{}{"get_orders"}},
			},
		}},
		// Transcoding tool defs — note upstreamIds is nil (the live LIST behaviour).
		{AssetID: "mcp-transcoding", UpstreamIDs: nil, ConfigurationData: map[string]interface{}{
			"tools": []interface{}{
				toolDef("get_pets", "GET", "/pets", "limit"),
				toolDef("post_pets", "POST", "/pets"),
				toolDef("get_orders", "GET", "/orders"),
			},
		}},
	}

	byName, ordered, byLabel := bridgeToolMappingFromPolicies(policies)

	if len(byName) != 3 {
		t.Fatalf("expected 3 tool defs by name, got %d", len(byName))
	}
	if len(ordered) != 3 {
		t.Fatalf("expected 3 ordered tool defs, got %d", len(ordered))
	}
	if got := byLabel["svc-a"]; len(got) != 2 || got[0] != "get_pets" || got[1] != "post_pets" {
		t.Errorf("svc-a tools = %v, want [get_pets post_pets]", got)
	}
	if got := byLabel["svc-b"]; len(got) != 1 || got[0] != "get_orders" {
		t.Errorf("svc-b tools = %v, want [get_orders]", got)
	}
	if _, ok := byName["get_pets"]; !ok {
		t.Error("get_pets def missing from byName map")
	}
}

// --- orphan self-heal: publish 409 -> hard-delete -> republish ---

func TestIsAssetConflict(t *testing.T) {
	cases := []struct {
		name string
		err  error
		want bool
	}{
		{"nil", nil, false},
		{"unrelated", errString("some other failure"), false},
		{"status 409 message", errString("failed to create asset with status 409: {...}"), true},
		{"platform code", errString(`{"code":"ASSET_PRE_CONDITIONS_FAILED"}`), true},
		{"typed conflict", anypointclient.NewConflictError("asset x"), true},
	}
	for _, tc := range cases {
		if got := isAssetConflict(tc.err); got != tc.want {
			t.Errorf("%s: isAssetConflict = %v, want %v", tc.name, got, tc.want)
		}
	}
}

func TestPublishBridgeAsset_SelfHealsOrphan(t *testing.T) {
	assetPath := "/exchange/api/v2/organizations/" + brOrg + "/assets/" + brOrg + "/my-bridge/1.0.0"
	getPath := "/exchange/api/v2/assets/" + brOrg + "/my-bridge/1.0.0"

	var publishCalls, deleteCalls int
	res := newBridgeResourceWithMock(t, map[string]func(w http.ResponseWriter, r *http.Request){
		assetPath: func(w http.ResponseWriter, r *http.Request) {
			switch r.Method {
			case http.MethodPost:
				publishCalls++
				if publishCalls == 1 {
					// First publish 409s on the orphaned version.
					testutil.ErrorResponse(w, http.StatusConflict, `{"code":"ASSET_PRE_CONDITIONS_FAILED"}`)
					return
				}
				testutil.JSONResponse(w, http.StatusCreated, map[string]string{"assetId": "my-bridge"})
			case http.MethodDelete:
				deleteCalls++
				w.WriteHeader(http.StatusNoContent)
			}
		},
		getPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodDelete {
				deleteCalls++
				w.WriteHeader(http.StatusNoContent)
				return
			}
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"groupId": brOrg, "assetId": "my-bridge", "version": "1.0.0", "type": "mcp",
			})
		},
	})

	err := res.publishBridgeAsset(context.Background(), brOrg, &agentsclient.PublishBridgeAssetInput{
		OrganizationID: brOrg, GroupID: brOrg, AssetID: "my-bridge", Version: "1.0.0",
		Name: "My Bridge", MetadataJSON: []byte(`{"tools":[]}`),
	})
	if err != nil {
		t.Fatalf("publishBridgeAsset() error = %v", err)
	}
	if publishCalls != 2 {
		t.Errorf("publish attempts = %d, want 2 (409 then success)", publishCalls)
	}
	if deleteCalls == 0 {
		t.Error("expected a hard-delete of the orphaned asset between publish attempts")
	}
}

type errString string

func (e errString) Error() string { return string(e) }
