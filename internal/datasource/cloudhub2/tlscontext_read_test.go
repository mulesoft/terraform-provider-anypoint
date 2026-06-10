package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	anypointclient "github.com/mulesoft/terraform-provider-anypoint/internal/client"
	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestTLSContextDataSource_Read(t *testing.T) {
	// GetTLSContext uses the list endpoint — mock /tlsContexts (no ID suffix)
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/tlsContexts"

	mockTLS := ch2client.TLSContext{
		ID:   "tls-ctx-1",
		Name: "my-tls-context",
		Type: "PEM",
		Ciphers: ch2client.CiphersConfig{
			AES128GcmSha256: true,
			AES256GcmSha384: true,
		},
	}

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, []ch2client.TLSContext{mockTLS})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewTLSContextDataSource().(*TLSContextDataSource)
	ds.client = &ch2client.TLSContextClient{
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
		"id":              tftypes.NewValue(tftypes.String, "tls-ctx-1"),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"type":            tftypes.NewValue(tftypes.String, nil),
		"ciphers":         tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["ciphers"], nil),
		"trust_store":     tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["trust_store"], nil),
		"key_store":       tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["key_store"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got TLSContextDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "tls-ctx-1" {
		t.Errorf("ID = %s, want tls-ctx-1", got.ID.ValueString())
	}
	if got.Name.ValueString() != "my-tls-context" {
		t.Errorf("Name = %s, want my-tls-context", got.Name.ValueString())
	}
}

func TestTLSContextDataSource_Read_Error(t *testing.T) {
	// GetTLSContext uses the list endpoint — return server error on list call
	basePath := "/runtimefabric/api/organizations/test-org-id/privatespaces/test-ps-id/tlsContexts"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewTLSContextDataSource().(*TLSContextDataSource)
	ds.client = &ch2client.TLSContextClient{
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
		"id":              tftypes.NewValue(tftypes.String, "tls-ctx-1"),
		"private_space_id": tftypes.NewValue(tftypes.String, "test-ps-id"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"name":            tftypes.NewValue(tftypes.String, nil),
		"type":            tftypes.NewValue(tftypes.String, nil),
		"ciphers":         tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["ciphers"], nil),
		"trust_store":     tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["trust_store"], nil),
		"key_store":       tftypes.NewValue(stateType.(tftypes.Object).AttributeTypes["key_store"], nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should report errors on API failure")
	}
}
