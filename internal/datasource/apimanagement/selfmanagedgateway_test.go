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

const smgDSGatewaysPath = "/standalone/api/v1/organizations/test-org-id/environments/test-env-id/gateways"

func TestNewSelfManagedGatewayDataSource(t *testing.T) {
	ds := NewSelfManagedGatewayDataSource()
	if ds == nil {
		t.Fatal("NewSelfManagedGatewayDataSource() returned nil")
	}
	if _, ok := ds.(datasource.DataSourceWithConfigure); !ok {
		t.Error("SelfManagedGatewayDataSource should implement DataSourceWithConfigure")
	}
}

func TestSelfManagedGatewayDataSource_Metadata(t *testing.T) {
	ds := NewSelfManagedGatewayDataSource()
	ctx := context.Background()
	req := datasource.MetadataRequest{ProviderTypeName: "test"}
	resp := &datasource.MetadataResponse{}
	ds.Metadata(ctx, req, resp)
	if resp.TypeName != "test_self_managed_gateways" {
		t.Errorf("Metadata() TypeName = %q, want %q", resp.TypeName, "test_self_managed_gateways")
	}
}

func TestSelfManagedGatewayDataSource_Schema(t *testing.T) {
	ds := NewSelfManagedGatewayDataSource()
	ctx := context.Background()
	resp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, resp)
	if resp.Diagnostics.HasError() {
		t.Fatalf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}
	for _, attr := range []string{"environment_id"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing required attribute %q", attr)
			continue
		}
		if !a.IsRequired() {
			t.Errorf("Schema() attribute %q should be required", attr)
		}
	}
	for _, attr := range []string{"id", "gateways"} {
		a, ok := resp.Schema.Attributes[attr]
		if !ok {
			t.Errorf("Schema() missing computed attribute %q", attr)
			continue
		}
		if !a.IsComputed() {
			t.Errorf("Schema() attribute %q should be computed", attr)
		}
	}
	// include_deleted is the Optional escape hatch for surfacing soft-delete tombstones.
	if a, ok := resp.Schema.Attributes["include_deleted"]; !ok {
		t.Error("Schema() missing optional attribute include_deleted")
	} else if !a.IsOptional() {
		t.Error("Schema() attribute include_deleted should be optional")
	}
}

func TestSelfManagedGatewayDataSource_Configure(t *testing.T) {
	ds := NewSelfManagedGatewayDataSource().(*SelfManagedGatewayDataSource)
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

func TestSelfManagedGatewayDataSource_Configure_InvalidProviderData(t *testing.T) {
	ds := NewSelfManagedGatewayDataSource().(*SelfManagedGatewayDataSource)
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

func TestSelfManagedGatewayDataSourceModel_Validation(t *testing.T) {
	model := SelfManagedGatewayDataSourceModel{}
	_ = model.ID
	_ = model.OrganizationID
	_ = model.EnvironmentID
	_ = model.IncludeDeleted
	_ = model.Gateways
}

func newSMGDataSource(baseURL string) *SelfManagedGatewayDataSource {
	ds := NewSelfManagedGatewayDataSource().(*SelfManagedGatewayDataSource)
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

// smgDSConfigRaw builds a config value for the data source. includeDeleted is passed as the
// value of the optional include_deleted attribute (nil ⇒ attribute omitted/null in config).
// All attributes of the object type must be present in the raw value (Computed ones set to
// null), or tftypes.NewValue rejects it.
func smgDSConfigRaw(t *testing.T, ds *SelfManagedGatewayDataSource, includeDeleted interface{}) (datasource.SchemaResponse, tftypes.Value) {
	t.Helper()
	ctx := context.Background()
	schemaResp := datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, &schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)
	objType := stateType.(tftypes.Object)
	elemType := objType.AttributeTypes["gateways"].(tftypes.List).ElementType
	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"environment_id":  tftypes.NewValue(tftypes.String, "test-env-id"),
		"include_deleted": tftypes.NewValue(tftypes.Bool, includeDeleted),
		"gateways":        tftypes.NewValue(tftypes.List{ElementType: elemType}, nil),
	})
	return schemaResp, configRaw
}

// liveAndTombstoneListHandler returns a list payload with one live gateway (flex-a, full
// real shape including tags + replicas) and one soft-deleted tombstone (flex-b, DELETED).
func liveAndTombstoneListHandler() map[string]func(w http.ResponseWriter, r *http.Request) {
	return map[string]func(w http.ResponseWriter, r *http.Request){
		smgDSGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			// Shape LIVE-VERIFIED (2026-07-21): id/name/status/organizationId/lastUpdate/
			// tags/replicas; replicas report one entry per connectivity status bucket.
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content": []map[string]interface{}{
					{
						"id":             "gw-1",
						"name":           "flex-a",
						"status":         "CONNECTED",
						"organizationId": "test-org-id",
						"lastUpdate":     "2026-07-21T14:29:07.69Z",
						"tags":           []string{"team-a", "prod"},
						"replicas": []map[string]interface{}{
							{"status": "CONNECTED", "count": 2, "certificateExpirationDates": []string{"2027-01-01T00:00:00Z"}},
							{"status": "DISCONNECTED", "count": 0, "certificateExpirationDates": []string{}},
						},
					},
					{
						"id":         "gw-2",
						"name":       "flex-b",
						"status":     "DELETED",
						"lastUpdate": "2026-07-20T10:00:00Z",
					},
				},
				"totalElements": 2,
			})
		},
	}
}

// By default the data source filters out DELETED tombstones and maps the full real shape
// (status/last_update/tags/replicas) for the live gateway.
func TestSelfManagedGatewayDataSource_Read_FiltersTombstoneByDefault(t *testing.T) {
	server := testutil.MockHTTPServer(t, liveAndTombstoneListHandler())
	ds := newSMGDataSource(server.URL)

	ctx := context.Background()
	schemaResp, configRaw := smgDSConfigRaw(t, ds, nil) // include_deleted omitted

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got SelfManagedGatewayDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	// The DELETED tombstone (flex-b) must be filtered out; only the live gateway remains.
	if len(got.Gateways) != 1 {
		t.Fatalf("expected 1 gateway (tombstone filtered), got %d", len(got.Gateways))
	}
	if got.ID.ValueString() != "test-org-id/test-env-id" {
		t.Errorf("id = %q, want test-org-id/test-env-id", got.ID.ValueString())
	}
	gw := got.Gateways[0]
	if gw.Name.ValueString() != "flex-a" || gw.Status.ValueString() != "CONNECTED" {
		t.Errorf("gateways[0] = name %q status %q, want flex-a/CONNECTED", gw.Name.ValueString(), gw.Status.ValueString())
	}
	if gw.LastUpdate.ValueString() != "2026-07-21T14:29:07.69Z" {
		t.Errorf("gateways[0].last_update = %q, want 2026-07-21T14:29:07.69Z", gw.LastUpdate.ValueString())
	}
	if len(gw.Tags) != 2 || gw.Tags[0].ValueString() != "team-a" {
		t.Errorf("gateways[0].tags = %v, want [team-a prod]", gw.Tags)
	}
	if len(gw.Replicas) != 2 {
		t.Fatalf("gateways[0].replicas len = %d, want 2", len(gw.Replicas))
	}
	if gw.Replicas[0].Status.ValueString() != "CONNECTED" || gw.Replicas[0].Count.ValueInt64() != 2 {
		t.Errorf("gateways[0].replicas[0] = status %q count %d, want CONNECTED/2",
			gw.Replicas[0].Status.ValueString(), gw.Replicas[0].Count.ValueInt64())
	}
	if len(gw.Replicas[0].CertificateExpirationDates) != 1 || gw.Replicas[0].CertificateExpirationDates[0].ValueString() != "2027-01-01T00:00:00Z" {
		t.Errorf("gateways[0].replicas[0].certificate_expiration_dates = %v, want [2027-01-01T00:00:00Z]", gw.Replicas[0].CertificateExpirationDates)
	}
}

// With include_deleted = true the tombstone is surfaced alongside the live gateway (audit path).
func TestSelfManagedGatewayDataSource_Read_IncludeDeletedSurfacesTombstone(t *testing.T) {
	server := testutil.MockHTTPServer(t, liveAndTombstoneListHandler())
	ds := newSMGDataSource(server.URL)

	ctx := context.Background()
	schemaResp, configRaw := smgDSConfigRaw(t, ds, true) // include_deleted = true

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got SelfManagedGatewayDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}

	if len(got.Gateways) != 2 {
		t.Fatalf("expected 2 gateways (tombstone included), got %d", len(got.Gateways))
	}
	// The tombstone must be present with its DELETED status intact.
	var sawTombstone bool
	for _, gw := range got.Gateways {
		if gw.Name.ValueString() == "flex-b" && gw.Status.ValueString() == "DELETED" {
			sawTombstone = true
		}
	}
	if !sawTombstone {
		t.Error("include_deleted=true should surface the DELETED tombstone flex-b")
	}
}

func TestSelfManagedGatewayDataSource_Read_Empty(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgDSGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"content":       []map[string]interface{}{},
				"totalElements": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	ds := newSMGDataSource(server.URL)

	ctx := context.Background()
	schemaResp, configRaw := smgDSConfigRaw(t, ds, nil)

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got SelfManagedGatewayDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if len(got.Gateways) != 0 {
		t.Errorf("expected 0 gateways, got %d", len(got.Gateways))
	}
}

func TestSelfManagedGatewayDataSource_Read_Error(t *testing.T) {
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		smgDSGatewaysPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)
	ds := newSMGDataSource(server.URL)

	ctx := context.Background()
	schemaResp, configRaw := smgDSConfigRaw(t, ds, nil)

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}
