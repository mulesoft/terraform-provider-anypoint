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

// The singular DS reads ONE gateway via the by-id GET primitive: the plural list path
// with the gateway id appended.
const smgSingleGetPath = "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1"

// The singular DS additionally reads the RICH per-replica detail from the dedicated
// /replicas sub-resource (the data behind the Runtime Manager "Replicas" tab).
const smgSingleReplicasPath = "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/gw-1/replicas"

func TestNewSelfManagedGatewaySingleDataSource(t *testing.T) {
	ds := NewSelfManagedGatewaySingleDataSource()
	if ds == nil {
		t.Fatal("NewSelfManagedGatewaySingleDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("SelfManagedGatewaySingleDataSource should implement DataSourceWithConfigure")
	}
}

func TestSelfManagedGatewaySingleDataSource_Metadata(t *testing.T) {
	ds := NewSelfManagedGatewaySingleDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "anypoint"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)

	// The SINGULAR data source shares the resource's type name — the plural list DS is
	// anypoint_self_managed_gateways (with the trailing 's').
	if resp.TypeName != "anypoint_self_managed_gateway" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "anypoint_self_managed_gateway")
	}
}

func TestSelfManagedGatewaySingleDataSource_Schema(t *testing.T) {
	ds := NewSelfManagedGatewaySingleDataSource()
	ctx := context.Background()
	resp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// id + environment_id are the lookup keys and must be required.
	for _, attr := range []string{"id", "environment_id"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing required attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("Schema() attribute %q should be required", attr)
		}
	}

	// organization_id defaults to the provider org, so it is optional+computed.
	if a, ok := resp.Schema.Attributes["organization_id"]; !ok {
		t.Error("Schema() missing attribute organization_id")
	} else if !a.IsOptional() || !a.IsComputed() {
		t.Error("Schema() attribute organization_id should be optional+computed")
	}

	// The full detail set must be computed — including versions, which the plural DS omits.
	for _, attr := range []string{"name", "status", "last_update", "tags", "versions", "replicas"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Schema() attribute %q should be computed", attr)
		}
	}
}

func TestSelfManagedGatewaySingleDataSource_Configure(t *testing.T) {
	ds := NewSelfManagedGatewaySingleDataSource().(*SelfManagedGatewaySingleDataSource)
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &anypointclient.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: providerData}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)
	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}
	if ds.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestSelfManagedGatewaySingleDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewSelfManagedGatewaySingleDataSource().(*SelfManagedGatewaySingleDataSource)
	ctx := context.Background()
	req := datasource.ConfigureRequest{ProviderData: "invalid"}
	resp := &datasource.ConfigureResponse{}
	ds.Configure(ctx, req, resp)
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should produce errors")
	}
	if ds.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func newSMGSingleDataSource(baseURL string) *SelfManagedGatewaySingleDataSource {
	ds := NewSelfManagedGatewaySingleDataSource().(*SelfManagedGatewaySingleDataSource)
	ds.client = &apimgmtclient.SelfManagedGatewayClient{
		AnypointClient: &anypointclient.AnypointClient{
			BaseURL:    baseURL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org-id",
		},
	}
	return ds
}

// smgSingleConfigRaw builds a config value with the two lookup keys set and every computed
// attribute set to null. All attributes of the object type must be present or tftypes.NewValue
// rejects the value.
func smgSingleConfigRaw(t *testing.T, ds *SelfManagedGatewaySingleDataSource, id string) (datasource.SchemaResponse, tftypes.Value) {
	t.Helper()
	ctx := context.Background()
	schemaResp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, id),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"status":          tftypes.NewValue(tftypes.String, nil),
		"last_update":     tftypes.NewValue(tftypes.String, nil),
		"tags":            tftypes.NewValue(objType.AttributeTypes["tags"], nil),
		"versions":        tftypes.NewValue(objType.AttributeTypes["versions"], nil),
		"replicas":        tftypes.NewValue(objType.AttributeTypes["replicas"], nil),
		"replica_details": tftypes.NewValue(objType.AttributeTypes["replica_details"], nil),
	})
	return schemaResp, configRaw
}

func TestSelfManagedGatewaySingleDataSource_Read(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgSingleGetPath: func(w http.ResponseWriter, r *http.Request) {
			// GET-by-id shape LIVE-VERIFIED (2026-07-21): the same object shape as the list,
			// plus a "versions" array (empty until a replica reports a runtime version).
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"id":             "gw-1",
				"name":           "flex-a",
				"status":         "CONNECTED",
				"organizationId": "test-org-id",
				"lastUpdate":     "2026-07-21T14:29:07.69Z",
				"tags":           []string{"team-a", "prod"},
				"versions":       []string{"1.9.0"},
				"replicas": []map[string]interface{}{
					{"status": "CONNECTED", "count": 2, "certificateExpirationDates": []string{"2027-01-01T00:00:00Z"}},
					{"status": "DISCONNECTED", "count": 0, "certificateExpirationDates": []string{}},
				},
			})
		},
		smgSingleReplicasPath: func(w http.ResponseWriter, r *http.Request) {
			// Rich per-replica detail envelope LIVE-VERIFIED (2026-07-26): a paginated
			// {content,totalElements,pageNumber,pageSize,gateway} wrapper. connectedAt is set and
			// disconnectedAt null for a CONNECTED node; the reverse for a DISCONNECTED node.
			// configurationStatus.message is null while up to date.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"id":                        "rep-connected",
						"targetId":                  "gw-1",
						"gatewayVersion":            "1.13.3",
						"status":                    "CONNECTED",
						"connectedAt":               "2026-07-25T11:24:34.39Z",
						"disconnectedAt":            nil,
						"configurationStatus":       map[string]interface{}{"status": "UP_TO_DATE", "message": nil},
						"name":                      "node-connected.default",
						"cid":                       "arm-mcm2-service-connected",
						"certificateExpirationDate": "2027-10-31T15:31:46.00Z",
						"nodeId":                    "rep-connected",
						"provider":                  "RR",
					},
					{
						"id":                        "rep-disconnected",
						"targetId":                  "gw-1",
						"gatewayVersion":            "1.13.3",
						"status":                    "DISCONNECTED",
						"connectedAt":               "2026-07-25T11:18:21.56Z",
						"disconnectedAt":            "2026-07-25T11:25:21.56Z",
						"configurationStatus":       map[string]interface{}{"status": "APPLYING", "message": "syncing"},
						"name":                      "node-disconnected.default",
						"cid":                       "arm-mcm2-service-disconnected",
						"certificateExpirationDate": "2027-10-31T15:31:46.00Z",
						"nodeId":                    "rep-disconnected",
						"provider":                  "RR",
					},
				},
				"totalElements": 2,
				"pageNumber":    0,
				"pageSize":      100,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	ds := newSMGSingleDataSource(server.URL)

	ctx := context.Background()
	schemaResp, configRaw := smgSingleConfigRaw(t, ds, "gw-1")

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got SelfManagedGatewaySingleDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if got.Name.ValueString() != "flex-a" {
		t.Errorf("Name = %q, want flex-a", got.Name.ValueString())
	}
	if got.Status.ValueString() != "CONNECTED" {
		t.Errorf("Status = %q, want CONNECTED", got.Status.ValueString())
	}
	if got.LastUpdate.ValueString() != "2026-07-21T14:29:07.69Z" {
		t.Errorf("LastUpdate = %q, want 2026-07-21T14:29:07.69Z", got.LastUpdate.ValueString())
	}
	if len(got.Tags) != 2 || got.Tags[0].ValueString() != "team-a" || got.Tags[1].ValueString() != "prod" {
		t.Errorf("Tags = %v, want [team-a prod]", got.Tags)
	}
	// versions is the headline field the plural DS omits.
	if len(got.Versions) != 1 || got.Versions[0].ValueString() != "1.9.0" {
		t.Errorf("Versions = %v, want [1.9.0]", got.Versions)
	}
	if len(got.Replicas) != 2 {
		t.Fatalf("Replicas len = %d, want 2", len(got.Replicas))
	}
	if got.Replicas[0].Status.ValueString() != "CONNECTED" || got.Replicas[0].Count.ValueInt64() != 2 {
		t.Errorf("Replicas[0] = status %q count %d, want CONNECTED/2",
			got.Replicas[0].Status.ValueString(), got.Replicas[0].Count.ValueInt64())
	}
	if len(got.Replicas[0].CertificateExpirationDates) != 1 || got.Replicas[0].CertificateExpirationDates[0].ValueString() != "2027-01-01T00:00:00Z" {
		t.Errorf("Replicas[0].CertificateExpirationDates = %v, want [2027-01-01T00:00:00Z]", got.Replicas[0].CertificateExpirationDates)
	}

	// Rich per-replica detail (Replicas-tab data) is the headline addition of this data source.
	if len(got.ReplicaDetails) != 2 {
		t.Fatalf("ReplicaDetails len = %d, want 2", len(got.ReplicaDetails))
	}
	rc := got.ReplicaDetails[0]
	if rc.ID.ValueString() != "rep-connected" || rc.NodeID.ValueString() != "rep-connected" {
		t.Errorf("ReplicaDetails[0] id/node_id = %q/%q, want rep-connected/rep-connected", rc.ID.ValueString(), rc.NodeID.ValueString())
	}
	if rc.Name.ValueString() != "node-connected.default" {
		t.Errorf("ReplicaDetails[0].Name = %q, want node-connected.default", rc.Name.ValueString())
	}
	if rc.TargetID.ValueString() != "gw-1" || rc.GatewayVersion.ValueString() != "1.13.3" {
		t.Errorf("ReplicaDetails[0] target_id/gateway_version = %q/%q, want gw-1/1.13.3", rc.TargetID.ValueString(), rc.GatewayVersion.ValueString())
	}
	if rc.Status.ValueString() != "CONNECTED" {
		t.Errorf("ReplicaDetails[0].Status = %q, want CONNECTED", rc.Status.ValueString())
	}
	if rc.ConnectedAt.ValueString() != "2026-07-25T11:24:34.39Z" {
		t.Errorf("ReplicaDetails[0].ConnectedAt = %q, want 2026-07-25T11:24:34.39Z", rc.ConnectedAt.ValueString())
	}
	// disconnectedAt is null for a connected node -> must map to "" (never null/unknown).
	if rc.DisconnectedAt.ValueString() != "" {
		t.Errorf("ReplicaDetails[0].DisconnectedAt = %q, want empty (null in payload)", rc.DisconnectedAt.ValueString())
	}
	if rc.ConfigurationStatus.ValueString() != "UP_TO_DATE" {
		t.Errorf("ReplicaDetails[0].ConfigurationStatus = %q, want UP_TO_DATE", rc.ConfigurationStatus.ValueString())
	}
	// configurationStatus.message is null when up to date -> "".
	if rc.ConfigurationMessage.ValueString() != "" {
		t.Errorf("ReplicaDetails[0].ConfigurationMessage = %q, want empty (null in payload)", rc.ConfigurationMessage.ValueString())
	}
	if rc.CertificateExpirationDate.ValueString() != "2027-10-31T15:31:46.00Z" {
		t.Errorf("ReplicaDetails[0].CertificateExpirationDate = %q, want 2027-10-31T15:31:46.00Z", rc.CertificateExpirationDate.ValueString())
	}
	if rc.Cid.ValueString() != "arm-mcm2-service-connected" || rc.Provider.ValueString() != "RR" {
		t.Errorf("ReplicaDetails[0] cid/provider = %q/%q, want arm-mcm2-service-connected/RR", rc.Cid.ValueString(), rc.Provider.ValueString())
	}

	// The DISCONNECTED node exercises the inverse null-mapping: disconnectedAt set, and a
	// non-null configuration message that must be surfaced verbatim.
	rd := got.ReplicaDetails[1]
	if rd.Status.ValueString() != "DISCONNECTED" {
		t.Errorf("ReplicaDetails[1].Status = %q, want DISCONNECTED", rd.Status.ValueString())
	}
	if rd.DisconnectedAt.ValueString() != "2026-07-25T11:25:21.56Z" {
		t.Errorf("ReplicaDetails[1].DisconnectedAt = %q, want 2026-07-25T11:25:21.56Z", rd.DisconnectedAt.ValueString())
	}
	if rd.ConfigurationStatus.ValueString() != "APPLYING" || rd.ConfigurationMessage.ValueString() != "syncing" {
		t.Errorf("ReplicaDetails[1] config status/message = %q/%q, want APPLYING/syncing", rd.ConfigurationStatus.ValueString(), rd.ConfigurationMessage.ValueString())
	}
}

// A 404 from the by-id GET must surface as an error diagnostic (a singular data source
// referencing a non-existent id is a configuration error, not empty state).
func TestSelfManagedGatewaySingleDataSource_Read_NotFound(t *testing.T) {
	missingPath := "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways/missing"
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		missingPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "Gateway not found by id")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	ds := newSMGSingleDataSource(server.URL)

	ctx := context.Background()
	schemaResp, configRaw := smgSingleConfigRaw(t, ds, "missing")

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Fatal("Read() on a missing self-managed gateway should report an error, got none")
	}
}
