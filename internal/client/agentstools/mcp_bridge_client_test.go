package agentstools

import (
	"context"
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

const (
	testOrgID    = "test-org"
	testEnvID    = "test-env"
	testBridgeID = 20922790
)

// newTestBridgeClient builds an MCPBridgeClient wired to a mock server whose handlers
// are the standard auth endpoints merged with the caller's endpoint handlers. Building
// via the real NewMCPBridgeClient constructor exercises the compose path (all three
// sub-clients share one cached token) exactly as production does.
func newTestBridgeClient(t *testing.T, extra map[string]func(w http.ResponseWriter, r *http.Request)) *MCPBridgeClient {
	t.Helper()
	handlers := testutil.StandardMockHandlers()
	for k, v := range extra {
		handlers[k] = v
	}
	server := testutil.MockHTTPServer(t, handlers)

	cfg := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	c, err := NewMCPBridgeClient(cfg)
	if err != nil {
		t.Fatalf("NewMCPBridgeClient() error = %v", err)
	}
	return c
}

// --- Pure helpers ---

func TestBumpPatchVersion(t *testing.T) {
	tests := []struct {
		in   string
		want string
	}{
		{"1.0.0", "1.0.1"},
		{"1.0.9", "1.0.10"},
		{"2.3.4", "2.3.5"},
		{"1.0.0-20260414150102", "1.0.0-20260414150102.1"}, // non-numeric patch -> append .1
		{"1.0", "1.0.1"},     // not x.y.z -> append .1
		{"1.0.x", "1.0.x.1"}, // non-numeric patch -> append .1
		{"", ".1"},
	}
	for _, tt := range tests {
		t.Run(tt.in, func(t *testing.T) {
			if got := BumpPatchVersion(tt.in); got != tt.want {
				t.Errorf("BumpPatchVersion(%q) = %q, want %q", tt.in, got, tt.want)
			}
		})
	}
}

func TestNewMCPBridgeMetadata(t *testing.T) {
	t.Run("unmanaged envelope", func(t *testing.T) {
		tools := []interface{}{map[string]interface{}{"name": "get_inventory"}}
		md := NewMCPBridgeMetadata(tools, false, "")

		if md.ProtocolVersion != "2025-03-26" {
			t.Errorf("protocolVersion = %q, want 2025-03-26", md.ProtocolVersion)
		}
		if md.Transport.Kind != "streamableHttp" || md.Transport.Path != "/mcp" {
			t.Errorf("transport = %+v, want {streamableHttp /mcp}", md.Transport)
		}
		if md.Capabilities.Tools.ListChanged != false {
			t.Errorf("capabilities.tools.listChanged = %v, want false", md.Capabilities.Tools.ListChanged)
		}
		if _, ok := md.Capabilities.Experimental["flexTranscoding"]; !ok {
			t.Errorf("capabilities.experimental missing flexTranscoding: %+v", md.Capabilities.Experimental)
		}
		if _, ok := md.Capabilities.Experimental["trusted"]; ok {
			t.Errorf("unmanaged metadata should NOT carry experimental.trusted: %+v", md.Capabilities.Experimental)
		}
		if len(md.Tools) != 1 {
			t.Errorf("tools len = %d, want 1", len(md.Tools))
		}
	})

	t.Run("managed envelope with mcpUrl", func(t *testing.T) {
		md := NewMCPBridgeMetadata(nil, true, "https://gw.example/mcp/")
		trusted, ok := md.Capabilities.Experimental["trusted"].(map[string]interface{})
		if !ok {
			t.Fatalf("managed metadata missing experimental.trusted: %+v", md.Capabilities.Experimental)
		}
		if trusted["managed"] != true {
			t.Errorf("trusted.managed = %v, want true", trusted["managed"])
		}
		if trusted["mcpUrl"] != "https://gw.example/mcp/" {
			t.Errorf("trusted.mcpUrl = %v, want the passed url", trusted["mcpUrl"])
		}
		// nil tools must serialize as [] not null.
		if md.Tools == nil {
			t.Errorf("nil tools should be normalized to empty slice")
		}
	})

	t.Run("JSON serializes tools as array", func(t *testing.T) {
		md := NewMCPBridgeMetadata(nil, false, "")
		b, err := md.JSON()
		if err != nil {
			t.Fatalf("JSON() error = %v", err)
		}
		if !strings.Contains(string(b), `"tools":[]`) {
			t.Errorf("expected \"tools\":[] in JSON, got %s", string(b))
		}
		if !strings.Contains(string(b), `"protocolVersion":"2025-03-26"`) {
			t.Errorf("expected protocolVersion in JSON, got %s", string(b))
		}
	})
}

func TestNewMCPBridgeClient_InvalidConfig(t *testing.T) {
	if _, err := NewMCPBridgeClient(&client.Config{ClientSecret: "s"}); err == nil {
		t.Errorf("expected error for missing client_id")
	}
	if _, err := NewMCPBridgeClient(&client.Config{ClientID: "c"}); err == nil {
		t.Errorf("expected error for missing client_secret")
	}
}

// --- Exchange publish (composed AssetClient) ---

func TestPublishBridgeAsset(t *testing.T) {
	var sawType, sawPlatform string
	var sawMetadataFile bool

	postPath := "/exchange/api/v2/organizations/" + testOrgID + "/assets/" + testOrgID + "/my-bridge/1.0.0"
	getPath := "/exchange/api/v2/assets/" + testOrgID + "/my-bridge/1.0.0"

	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		postPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "POST", postPath)
			if got := r.Header.Get("x-sync-publication"); got != "true" {
				t.Errorf("x-sync-publication = %q, want true", got)
			}
			if err := r.ParseMultipartForm(1 << 20); err != nil {
				t.Fatalf("ParseMultipartForm: %v", err)
			}
			sawType = r.FormValue("type")
			sawPlatform = r.FormValue("properties.platform")
			if r.MultipartForm != nil {
				if _, ok := r.MultipartForm.File["files.mcp-metadata.json"]; ok {
					sawMetadataFile = true
				}
			}
			testutil.JSONResponse(w, http.StatusCreated, map[string]string{"assetId": "my-bridge"})
		},
		getPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"groupId": testOrgID, "assetId": "my-bridge", "version": "1.0.0",
				"type": "mcp", "status": "published",
			})
		},
	})

	md := NewMCPBridgeMetadata([]interface{}{map[string]interface{}{"name": "get_inventory"}}, false, "")
	metaJSON, err := md.JSON()
	if err != nil {
		t.Fatalf("metadata JSON: %v", err)
	}

	asset, err := c.PublishBridgeAsset(context.Background(), &PublishBridgeAssetInput{
		OrganizationID: testOrgID,
		GroupID:        testOrgID,
		AssetID:        "my-bridge",
		Version:        "1.0.0",
		Name:           "My Bridge",
		MetadataJSON:   metaJSON,
	})
	if err != nil {
		t.Fatalf("PublishBridgeAsset() error = %v", err)
	}
	if asset.AssetID != "my-bridge" || asset.Type != "mcp" {
		t.Errorf("readback asset = %+v, want assetId=my-bridge type=mcp", asset)
	}
	if sawType != "mcp" {
		t.Errorf("multipart type = %q, want mcp", sawType)
	}
	if sawPlatform != "mulesoft" {
		t.Errorf("multipart properties.platform = %q, want mulesoft", sawPlatform)
	}
	if !sawMetadataFile {
		t.Errorf("multipart missing files.mcp-metadata.json part")
	}
}

func TestDeleteBridgeAssetVersion_HardDelete(t *testing.T) {
	var sawDeleteType string
	delPath := "/exchange/api/v2/assets/" + testOrgID + "/my-bridge/1.0.0"
	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		delPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "DELETE", delPath)
			sawDeleteType = r.Header.Get("x-delete-type")
			w.WriteHeader(http.StatusNoContent)
		},
	})
	if err := c.DeleteBridgeAssetVersion(context.Background(), testOrgID, "my-bridge", "1.0.0"); err != nil {
		t.Fatalf("DeleteBridgeAssetVersion() error = %v", err)
	}
	if sawDeleteType != "hard-delete" {
		t.Errorf("x-delete-type = %q, want hard-delete", sawDeleteType)
	}
}

// --- Instance CRUD (inlined bridge transport) ---

func TestCreateBridgeInstance_Success(t *testing.T) {
	createPath := "/apimanager/xapi/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis"
	var body CreateBridgeInstanceRequest

	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		createPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "POST", createPath)
			if got := r.Header.Get("X-Web-App"); got != "api-manager-ui-lib" {
				t.Errorf("X-Web-App = %q, want api-manager-ui-lib", got)
			}
			if got := r.Header.Get("X-Envoy-Upstream-Rq-Timeout-Ms"); got != "30000" {
				t.Errorf("X-Envoy-Upstream-Rq-Timeout-Ms = %q, want 30000", got)
			}
			if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
				t.Fatalf("decode create body: %v", err)
			}
			testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{
				"id": testBridgeID, "technology": "flexGateway", "assetVersion": "1.0.0",
				"metadata": map[string]string{"generatedBy": "mcp_bridge"},
			})
		},
	})

	req := &CreateBridgeInstanceRequest{
		Spec:       &MCPBridgeSpec{AssetID: "my-bridge", GroupID: testOrgID, Version: "1.0.0"},
		Endpoint:   &MCPBridgeEndpoint{Type: "mcp"},
		Technology: "flexGateway",
		Metadata:   map[string]string{"generatedBy": "mcp_bridge"},
		Routing: []MCPBridgeRoute{{
			Label: "inventory-api",
			Rules: &MCPBridgeRules{Headers: map[string]string{"X-UPSTREAM-NAME": "inventory-api"}},
			Upstreams: []MCPBridgeRouteUpstream{{
				Weight: 100, URI: "https://backend.example",
				Connection: &MCPBridgeConnection{AssetID: "inventory-api", GroupID: testOrgID, Version: "1.0.0"},
			}},
		}},
	}
	bridge, err := c.CreateBridgeInstance(context.Background(), testOrgID, testEnvID, req)
	if err != nil {
		t.Fatalf("CreateBridgeInstance() error = %v", err)
	}
	if bridge.ID != testBridgeID {
		t.Errorf("bridge.ID = %d, want %d", bridge.ID, testBridgeID)
	}
	// Assert the wire body carried the bridge-defining fields.
	if body.Technology != "flexGateway" {
		t.Errorf("wire technology = %q, want flexGateway", body.Technology)
	}
	if body.Metadata["generatedBy"] != "mcp_bridge" {
		t.Errorf("wire metadata.generatedBy = %q, want mcp_bridge", body.Metadata["generatedBy"])
	}
	if len(body.Routing) != 1 || body.Routing[0].Upstreams[0].Connection == nil {
		t.Errorf("wire routing must carry upstream connection at create time: %+v", body.Routing)
	}
	if body.Routing[0].Upstreams[0].Connection.AssetID != "inventory-api" {
		t.Errorf("wire connection.assetId = %q, want inventory-api", body.Routing[0].Upstreams[0].Connection.AssetID)
	}
}

func TestCreateBridgeInstance_RetriesOnGatewayNotReady(t *testing.T) {
	// Shrink the retry backoff so the retry path does not stall the test.
	orig := bridgeGatewayRetryDelay
	bridgeGatewayRetryDelay = time.Millisecond
	t.Cleanup(func() { bridgeGatewayRetryDelay = orig })

	createPath := "/apimanager/xapi/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis"
	var calls int
	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		createPath: func(w http.ResponseWriter, r *http.Request) {
			calls++
			if calls == 1 {
				testutil.ErrorResponse(w, http.StatusBadRequest, "GatewayNotReadyError: gateway is starting")
				return
			}
			testutil.JSONResponse(w, http.StatusCreated, map[string]interface{}{"id": testBridgeID})
		},
	})

	bridge, err := c.CreateBridgeInstance(context.Background(), testOrgID, testEnvID, &CreateBridgeInstanceRequest{Technology: "flexGateway"})
	if err != nil {
		t.Fatalf("CreateBridgeInstance() error = %v", err)
	}
	if bridge.ID != testBridgeID {
		t.Errorf("bridge.ID = %d, want %d", bridge.ID, testBridgeID)
	}
	if calls != 2 {
		t.Errorf("expected 2 calls (1 retry), got %d", calls)
	}
}

func TestCreateBridgeInstance_NonRetryableError(t *testing.T) {
	createPath := "/apimanager/xapi/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis"
	var calls int
	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		createPath: func(w http.ResponseWriter, r *http.Request) {
			calls++
			testutil.ErrorResponse(w, http.StatusBadRequest, "InvalidSpecError: bad asset")
		},
	})
	if _, err := c.CreateBridgeInstance(context.Background(), testOrgID, testEnvID, &CreateBridgeInstanceRequest{}); err == nil {
		t.Fatalf("expected error for non-retryable 400")
	}
	if calls != 1 {
		t.Errorf("non-retryable 400 must NOT retry; got %d calls", calls)
	}
}

func TestGetBridge(t *testing.T) {
	getPath := "/apimanager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis/20922790"

	t.Run("success", func(t *testing.T) {
		c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
			getPath: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", getPath)
				if r.URL.Query().Get("includeProxyConfiguration") != "true" {
					t.Errorf("missing includeProxyConfiguration=true")
				}
				if r.URL.Query().Get("includeTlsContexts") != "true" {
					t.Errorf("missing includeTlsContexts=true")
				}
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"id": testBridgeID, "assetId": "MCP-Bridge-Testing", "technology": "flexGateway",
					"routing": []map[string]interface{}{
						{"label": "inventory-api", "upstreams": []map[string]interface{}{{"id": "u1", "weight": 100}}},
					},
					"metadata": map[string]string{"generatedBy": "mcp_bridge"},
				})
			},
		})
		bridge, err := c.GetBridge(context.Background(), testOrgID, testEnvID, testBridgeID)
		if err != nil {
			t.Fatalf("GetBridge() error = %v", err)
		}
		if bridge.AssetID != "MCP-Bridge-Testing" {
			t.Errorf("assetId = %q, want MCP-Bridge-Testing", bridge.AssetID)
		}
		if len(bridge.Routing) != 1 || bridge.Routing[0].Upstreams[0].ID != "u1" {
			t.Errorf("routing not decoded: %+v", bridge.Routing)
		}
	})

	t.Run("404 maps to NotFoundError", func(t *testing.T) {
		c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
			getPath: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
		})
		_, err := c.GetBridge(context.Background(), testOrgID, testEnvID, testBridgeID)
		if err == nil {
			t.Fatalf("expected error for 404")
		}
		if !client.IsNotFound(err) {
			t.Errorf("expected NotFoundError, got %v", err)
		}
	})
}

func TestGetBridgeUpstreams(t *testing.T) {
	upPath := "/apimanager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis/20922790/upstreams"
	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		upPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "GET", upPath)
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"total": 1,
				"upstreams": []map[string]interface{}{
					{
						"id": "f0c83e44", "uri": "https://inventory.example",
						"connection": map[string]interface{}{
							"assetId": "inventory-api", "groupId": testOrgID, "version": "1.0.0", "assetType": "raml",
						},
					},
				},
			})
		},
	})
	ups, err := c.GetBridgeUpstreams(context.Background(), testOrgID, testEnvID, testBridgeID)
	if err != nil {
		t.Fatalf("GetBridgeUpstreams() error = %v", err)
	}
	if len(ups) != 1 {
		t.Fatalf("upstreams len = %d, want 1", len(ups))
	}
	if ups[0].URI != "https://inventory.example" {
		t.Errorf("uri = %q, want https://inventory.example", ups[0].URI)
	}
	if ups[0].Connection == nil || ups[0].Connection.AssetID != "inventory-api" {
		t.Errorf("connection not decoded: %+v", ups[0].Connection)
	}
}

func TestUpdateBridgeAssetVersion(t *testing.T) {
	patchPath := "/apimanager/xapi/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis/20922790"
	var body map[string]string
	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		patchPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "PATCH", patchPath)
			if r.URL.Query().Get("checkAutomatedPolicies") != "true" {
				t.Errorf("missing checkAutomatedPolicies=true")
			}
			_ = json.NewDecoder(r.Body).Decode(&body)
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{"id": testBridgeID, "assetVersion": "1.0.1"})
		},
	})
	bridge, err := c.UpdateBridgeAssetVersion(context.Background(), testOrgID, testEnvID, testBridgeID, "1.0.1")
	if err != nil {
		t.Fatalf("UpdateBridgeAssetVersion() error = %v", err)
	}
	if bridge.AssetVersion != "1.0.1" {
		t.Errorf("assetVersion = %q, want 1.0.1", bridge.AssetVersion)
	}
	if body["assetVersion"] != "1.0.1" {
		t.Errorf("wire body assetVersion = %q, want 1.0.1", body["assetVersion"])
	}
}

func TestDeleteBridgeInstance(t *testing.T) {
	delPath := "/apimanager/api/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis/20922790"

	t.Run("204 success", func(t *testing.T) {
		c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
			delPath: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", delPath)
				w.WriteHeader(http.StatusNoContent)
			},
		})
		if err := c.DeleteBridgeInstance(context.Background(), testOrgID, testEnvID, testBridgeID); err != nil {
			t.Fatalf("DeleteBridgeInstance() error = %v", err)
		}
	})

	t.Run("404 is idempotent", func(t *testing.T) {
		c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
			delPath: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
		})
		if err := c.DeleteBridgeInstance(context.Background(), testOrgID, testEnvID, testBridgeID); err != nil {
			t.Errorf("DeleteBridgeInstance() on 404 should be nil, got %v", err)
		}
	})
}

func TestListBridges_FiltersByGeneratedBy(t *testing.T) {
	listPath := "/apimanager/xapi/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/apis"
	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "GET", listPath)
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"total": 3,
				"instances": []map[string]interface{}{
					{"id": 1, "assetId": "a-bridge", "metadata": map[string]string{"generatedBy": "mcp_bridge"}},
					{"id": 2, "assetId": "plain-server"}, // no metadata -> excluded
					{"id": 3, "assetId": "other", "metadata": map[string]string{"generatedBy": "something_else"}}, // excluded
				},
			})
		},
	})
	bridges, err := c.ListBridges(context.Background(), testOrgID, testEnvID)
	if err != nil {
		t.Fatalf("ListBridges() error = %v", err)
	}
	if len(bridges) != 1 {
		t.Fatalf("ListBridges filtered len = %d, want 1", len(bridges))
	}
	if bridges[0].ID != 1 || bridges[0].AssetID != "a-bridge" {
		t.Errorf("wrong bridge returned: %+v", bridges[0])
	}
}

func TestGetGatewayInfo(t *testing.T) {
	gwPath := "/gatewaymanager/xapi/v1/organizations/" + testOrgID + "/environments/" + testEnvID + "/gateways/gw-1"
	c := newTestBridgeClient(t, map[string]func(w http.ResponseWriter, r *http.Request){
		gwPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.AssertHTTPRequest(t, r, "GET", gwPath)
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id": "gw-1", "name": "ms-large-gw", "runtimeVersion": "1.13.0",
			})
		},
	})
	gw, err := c.GetGatewayInfo(context.Background(), testOrgID, testEnvID, "gw-1")
	if err != nil {
		t.Fatalf("GetGatewayInfo() error = %v", err)
	}
	if gw.Name != "ms-large-gw" || gw.RuntimeVersion != "1.13.0" {
		t.Errorf("gateway = %+v, want name=ms-large-gw version=1.13.0", gw)
	}
}
