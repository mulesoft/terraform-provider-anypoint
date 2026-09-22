package accessmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/datasource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewConnectedAppScopesDataSource(t *testing.T) {
	dataSource := NewConnectedAppScopesDataSource()

	if dataSource == nil {
		t.Error("NewConnectedAppScopesDataSource() returned nil")
	}

	// Verify it implements the expected interfaces
	if _, ok := dataSource.(datasource.DataSourceWithConfigure); !ok {
		t.Error("ConnectedAppScopesDataSource does not implement DataSourceWithConfigure")
	}
}

func TestConnectedAppScopesDataSource_Metadata(t *testing.T) {
	dataSource := NewConnectedAppScopesDataSource()

	ctx := context.Background()
	req := datasource.MetadataRequest{
		ProviderTypeName: "test",
	}
	resp := &datasource.MetadataResponse{}

	dataSource.Metadata(ctx, req, resp)

	if resp.TypeName != "test_connected_app_scopes" {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, "test_connected_app_scopes")
	}
}

func TestConnectedAppScopesDataSource_Schema(t *testing.T) {
	dataSource := NewConnectedAppScopesDataSource()

	ctx := context.Background()
	req := datasource.SchemaRequest{}
	resp := &datasource.SchemaResponse{}

	dataSource.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check required attributes
	requiredAttrs := []string{"connected_app_id"}
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	// Check computed attributes
	computedAttrs := []string{"id", "scopes"}
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}
}

func TestConnectedAppScopesDataSource_Configure(t *testing.T) {
	dataSource := NewConnectedAppScopesDataSource().(*ConnectedAppScopesDataSource)

	// Test with valid provider data
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-pass",
	}

	ctx := context.Background()
	req := datasource.ConfigureRequest{
		ProviderData: providerData,
	}
	resp := &datasource.ConfigureResponse{}

	dataSource.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}

	// Verify client is configured
	if dataSource.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestConnectedAppScopesDataSourceModel_Validation(t *testing.T) {
	// Test that all model fields exist and are properly typed
	model := ConnectedAppScopesDataSourceModel{}

	// Verify all expected fields exist
	_ = model.ID
	_ = model.ConnectedAppID
	_ = model.Scopes
}

func TestConnectedAppScopesDataSource_ReadClientTests(t *testing.T) {
	tests := []struct {
		name           string
		connectedAppID string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
		expectedScopes int
	}{
		{
			name:           "successful read with scopes",
			connectedAppID: "test-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/connectedApplications/test-app-id/scopes")
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"scope": "admin:cloudhub",
							"context_params": map[string]interface{}{
								"org": "test-org-id",
							},
						},
						{
							"scope":          "read:applications",
							"context_params": map[string]interface{}{},
						},
					},
					"total": 2,
				})
			},
			wantErr:        false,
			expectedScopes: 2,
		},
		{
			name:           "successful read with no scopes",
			connectedAppID: "test-app-id-no-scopes",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/connectedApplications/test-app-id-no-scopes/scopes")
				testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
					"data":  []map[string]interface{}{},
					"total": 0,
				})
			},
			wantErr:        false,
			expectedScopes: 0,
		},
		{
			name:           "connected app not found",
			connectedAppID: "nonexistent-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Connected app not found")
			},
			wantErr:     true,
			errContains: "Could not read connected app scopes for ID nonexistent-app-id",
		},
		{
			name:           "server error",
			connectedAppID: "test-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "Could not read connected app scopes for ID test-app-id",
		},
		{
			name:           "malformed API response",
			connectedAppID: "test-app-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "Could not read connected app scopes for ID test-app-id",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Create handlers for different connected app IDs
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/connectedApplications/test-app-id/scopes":                tt.mockHandler,
				"/accounts/api/connectedApplications/test-app-id-no-scopes/scopes":      tt.mockHandler,
				"/accounts/api/connectedApplications/nonexistent-app-id/scopes":         tt.mockHandler,
				"/accounts/api/connectedApplications/test-app-id-malformed-json/scopes": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			// Create client with mock server
			scopesClient := &accessmanagement.ConnectedAppScopesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			// Test the underlying client directly
			scopes, err := scopesClient.GetConnectedAppScopes(context.Background(), tt.connectedAppID)

			// Verify results
			if tt.wantErr {
				if err == nil {
					t.Errorf("GetConnectedAppScopes() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					// For read tests, we check if error contains key parts
					hasExpectedError := strings.Contains(err.Error(), "not found") ||
						strings.Contains(err.Error(), "500") ||
						strings.Contains(err.Error(), "invalid character")
					if !hasExpectedError {
						t.Errorf("GetConnectedAppScopes() error = %v, want error containing patterns for %v", err, tt.errContains)
					}
				}
			} else {
				if err != nil {
					t.Errorf("GetConnectedAppScopes() unexpected error = %v", err)
				}
				if scopes == nil {
					t.Errorf("GetConnectedAppScopes() returned nil scopes")
				}
				if scopes != nil && len(scopes.Scopes) != tt.expectedScopes {
					t.Errorf("GetConnectedAppScopes() Scopes count = %v, want %v", len(scopes.Scopes), tt.expectedScopes)
				}
			}
		})
	}
}

func TestConnectedAppScopesDataSource_Read(t *testing.T) {
	basePath := "/accounts/api/connectedApplications/test-app-id/scopes"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"scope": "admin:cloudhub",
						"context_params": map[string]interface{}{
							"org": "test-org-id",
						},
					},
				},
				"total": 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewConnectedAppScopesDataSource().(*ConnectedAppScopesDataSource)
	ds.client = &accessmanagement.ConnectedAppScopesClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	// Build the nested scopes set type
	scopeObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"scope":          tftypes.String,
			"display_name":   tftypes.String,
			"context_params": tftypes.Map{ElementType: tftypes.String},
		},
	}

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, nil),
		"connected_app_id": tftypes.NewValue(tftypes.String, "test-app-id"),
		"scopes":           tftypes.NewValue(tftypes.Set{ElementType: scopeObjectType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got ConnectedAppScopesDataSourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "test-app-id" {
		t.Errorf("Expected ID 'test-app-id', got %s", got.ID.ValueString())
	}
	if got.Scopes.IsNull() {
		t.Error("Expected scopes to be populated, got null")
	}
}

func TestConnectedAppScopesDataSource_Read_Error(t *testing.T) {
	basePath := "/accounts/api/connectedApplications/test-app-id/scopes"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	ds := NewConnectedAppScopesDataSource().(*ConnectedAppScopesDataSource)
	ds.client = &accessmanagement.ConnectedAppScopesClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
		},
	}

	ctx := context.Background()
	schemaResp := &datasource.SchemaResponse{}
	ds.Schema(ctx, datasource.SchemaRequest{}, schemaResp)
	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	// Build the nested scopes set type
	scopeObjectType := tftypes.Object{
		AttributeTypes: map[string]tftypes.Type{
			"scope":          tftypes.String,
			"display_name":   tftypes.String,
			"context_params": tftypes.Map{ElementType: tftypes.String},
		},
	}

	configRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":               tftypes.NewValue(tftypes.String, nil),
		"connected_app_id": tftypes.NewValue(tftypes.String, "test-app-id"),
		"scopes":           tftypes.NewValue(tftypes.Set{ElementType: scopeObjectType}, nil),
	})

	req := datasource.ReadRequest{Config: tfsdk.Config{Schema: schemaResp.Schema, Raw: configRaw}}
	resp := &datasource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: configRaw}}
	ds.Read(ctx, req, resp)

	if !resp.Diagnostics.HasError() {
		t.Error("Read() should have errors on server error")
	}
}

// Benchmarks
func BenchmarkConnectedAppScopesDataSource_Schema(b *testing.B) {
	dataSource := NewConnectedAppScopesDataSource()
	ctx := context.Background()
	req := datasource.SchemaRequest{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &datasource.SchemaResponse{}
		dataSource.Schema(ctx, req, resp)
	}
}
