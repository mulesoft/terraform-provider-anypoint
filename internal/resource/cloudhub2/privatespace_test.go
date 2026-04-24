package cloudhub2

import (
	"context"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-framework/types"
	"github.com/hashicorp/terraform-plugin-go/tftypes"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewPrivateSpaceResource(t *testing.T) {
	r := NewPrivateSpaceResource()
	
	if r == nil {
		t.Error("NewPrivateSpaceResource() returned nil")
	}
	
	// Verify it implements the expected interfaces
	var _ resource.Resource = r
	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("resource should implement ResourceWithConfigure")
	}
	if _, ok := r.(resource.ResourceWithImportState); !ok {
		t.Error("resource should implement ResourceWithImportState")
	}
}

func TestPrivateSpaceResource_Metadata(t *testing.T) {
	r := NewPrivateSpaceResource()
	testutil.TestResourceMetadata(t, r, "_private_space")
}

func TestPrivateSpaceResource_Schema(t *testing.T) {
	res := NewPrivateSpaceResource()
	
	requiredAttrs := []string{"name", "region"}
	optionalAttrs := []string{"enable_iam_role", "enable_egress", "organization_id"}
	computedAttrs := []string{"id", "status", "root_organization_id", "mule_app_deployment_count", "days_left_for_relaxed_quota", "vpc_migration_in_progress"}
	
	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestPrivateSpaceResource_Configure(t *testing.T) {
	res := NewPrivateSpaceResource().(*PrivateSpaceResource)
	
	// Test with valid provider data
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.ClientConfig{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
	
	testutil.TestResourceConfigure(t, res, providerData)
	
	// Verify client is configured
	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestPrivateSpaceResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewPrivateSpaceResource().(*PrivateSpaceResource)
	
	ctx := context.Background()
	req := resource.ConfigureRequest{
		ProviderData: "invalid-data", // Wrong type
	}
	resp := &resource.ConfigureResponse{}
	
	res.Configure(ctx, req, resp)
	
	if !resp.Diagnostics.HasError() {
		t.Error("Configure() with invalid provider data should have errors")
	}
	
	if res.client != nil {
		t.Error("Configure() with invalid data should not set client")
	}
}

func TestPrivateSpaceResource_Create(t *testing.T) {
	mockPrivateSpace := &cloudhub2.PrivateSpace{
		ID:                 "test-space-id",
		Name:               "Test Space",
		Region:             "us-east-1",
		Status:             "ACTIVE",
		OrganizationID:     "test-org-id",
		RootOrganizationID: "root-org-id",
		EnableEgress:       true,
		EnableIAMRole:      false,
	}
	
	tests := []struct {
		name        string
		model       PrivateSpaceResourceModel
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name: "successful creation",
			model: PrivateSpaceResourceModel{
				Name:          types.StringValue("Test Space"),
				Region:        types.StringValue("us-east-1"),
				EnableEgress:  types.BoolValue(true),
				EnableIAMRole: types.BoolValue(false),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "POST" {
					t.Errorf("Expected POST request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusCreated, mockPrivateSpace)
			},
			wantErr: false,
		},
		{
			name: "creation failure - server error",
			model: PrivateSpaceResourceModel{
				Name:   types.StringValue("Test Space"),
				Region: types.StringValue("invalid-region"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid region")
			},
			wantErr:     true,
			errContains: "failed to create private space",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/runtimefabric/api/organizations/test-org-id/privatespaces": tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)
			
			psClient := &cloudhub2.PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
					OrgID:      "test-org",
				},
			}
			
			ctx := context.Background()
			
			createReq := &cloudhub2.CreatePrivateSpaceRequest{
				Name:   tt.model.Name.ValueString(),
				Region: tt.model.Region.ValueString(),
			}
			
			ps, err := psClient.CreatePrivateSpace(ctx, "test-org-id", createReq)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if ps == nil {
					t.Error("Expected private space but got nil")
				} else {
					if ps.Name != tt.model.Name.ValueString() {
						t.Errorf("Expected name %s, got %s", tt.model.Name.ValueString(), ps.Name)
					}
				}
			}
		})
	}
}

func TestPrivateSpaceResource_Read(t *testing.T) {
	mockPrivateSpace := &cloudhub2.PrivateSpace{
		ID:                      "test-space-id",
		Name:                    "Test Space",
		Region:                  "us-east-1",
		Status:                  "ACTIVE",
		OrganizationID:          "test-org-id",
		RootOrganizationID:      "root-org-id",
		MuleAppDeploymentCount:  5,
		DaysLeftForRelaxedQuota: 30,
		VPCMigrationInProgress:  false,
	}
	
	tests := []struct {
		name        string
		spaceID     string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful read",
			spaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr: false,
		},
		{
			name:    "private space not found",
			spaceID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr:     true,
			errContains: "private space not found",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/runtimefabric/api/organizations/test-org-id/privatespaces/" + tt.spaceID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)
			
			psClient := &cloudhub2.PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
					OrgID:      "test-org",
				},
			}
			
			ctx := context.Background()
			ps, err := psClient.GetPrivateSpace(ctx, "test-org-id", tt.spaceID)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if ps == nil {
					t.Error("Expected private space but got nil")
				} else {
					if ps.ID != tt.spaceID {
						t.Errorf("Expected ID %s, got %s", tt.spaceID, ps.ID)
					}
					if ps.Name != mockPrivateSpace.Name {
						t.Errorf("Expected name %s, got %s", mockPrivateSpace.Name, ps.Name)
					}
				}
			}
		})
	}
}

func TestPrivateSpaceResource_Update(t *testing.T) {
	mockPrivateSpace := &cloudhub2.PrivateSpace{
		ID:     "test-space-id",
		Name:   "Updated Test Space",
		Region: "us-east-1",
		Status: "ACTIVE",
	}
	
	tests := []struct {
		name        string
		spaceID     string
		updateName  string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:       "successful update",
			spaceID:    "test-space-id",
			updateName: "Updated Test Space",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PUT" {
					t.Errorf("Expected PUT request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockPrivateSpace)
			},
			wantErr: false,
		},
		{
			name:       "update failure - not found",
			spaceID:    "nonexistent-id",
			updateName: "Updated Test Space",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/runtimefabric/api/organizations/test-org-id/privatespaces/" + tt.spaceID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)
			
			psClient := &cloudhub2.PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
					OrgID:      "test-org",
				},
			}
			
			ctx := context.Background()
			updateName := tt.updateName
			updateReq := &cloudhub2.UpdatePrivateSpaceRequest{
				Name: &updateName,
			}
			
			ps, err := psClient.UpdatePrivateSpace(ctx, "test-org-id", tt.spaceID, updateReq)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
				if ps == nil {
					t.Error("Expected private space but got nil")
				} else if ps.Name != tt.updateName {
					t.Errorf("Expected name %s, got %s", tt.updateName, ps.Name)
				}
			}
		})
	}
}

func TestPrivateSpaceResource_Delete(t *testing.T) {
	tests := []struct {
		name        string
		spaceID     string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:    "successful deletion",
			spaceID: "test-space-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "DELETE" {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:    "deletion failure - not found",
			spaceID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Private space not found")
			},
			wantErr: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/runtimefabric/api/organizations/test-org-id/privatespaces/" + tt.spaceID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)
			
			psClient := &cloudhub2.PrivateSpacesClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
					OrgID:      "test-org",
				},
			}
			
			ctx := context.Background()
			err := psClient.DeletePrivateSpace(ctx, "test-org-id", tt.spaceID)
			
			if tt.wantErr {
				if err == nil {
					t.Error("Expected error but got none")
				}
			} else {
				if err != nil {
					t.Errorf("Unexpected error: %v", err)
				}
			}
		})
	}
}

func TestPrivateSpaceResource_ImportState(t *testing.T) {
	res := NewPrivateSpaceResource()

	ctx := context.Background()

	schemaReq := resource.SchemaRequest{}
	schemaResp := &resource.SchemaResponse{}
	res.Schema(ctx, schemaReq, schemaResp)

	stateType := schemaResp.Schema.Type().TerraformType(ctx)

	req := resource.ImportStateRequest{
		ID: "test-space-id",
	}
	resp := &resource.ImportStateResponse{
		State: tfsdk.State{
			Schema: schemaResp.Schema,
			Raw:    tftypes.NewValue(stateType, nil),
		},
	}

	if importableResource, ok := res.(resource.ResourceWithImportState); ok {
		importableResource.ImportState(ctx, req, resp)

		if resp.Diagnostics.HasError() {
			t.Errorf("ImportState() has errors: %v", resp.Diagnostics.Errors())
		}
	} else {
		t.Error("Resource does not implement ResourceWithImportState")
	}
}

func TestPrivateSpaceResourceModel_Validation(t *testing.T) {
	// Test that all model fields exist and are properly typed
	model := PrivateSpaceResourceModel{}
	
	// Verify all expected fields exist
	_ = model.ID
	_ = model.Name
	_ = model.Region
	_ = model.EnableIAMRole
	_ = model.EnableEgress
	_ = model.Status
	_ = model.OrganizationID
	_ = model.RootOrganizationID
	_ = model.MuleAppDeploymentCount
	_ = model.DaysLeftForRelaxedQuota
	_ = model.VPCMigrationInProgress
	_ = model.ManagedFirewallRules
	_ = model.FirewallRules
	_ = model.GlobalSpaceStatus
}

// Benchmarks

func BenchmarkPrivateSpaceResource_Schema(b *testing.B) {
	res := NewPrivateSpaceResource()
	ctx := context.Background()
	req := resource.SchemaRequest{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.SchemaResponse{}
		res.Schema(ctx, req, resp)
	}
}

func BenchmarkPrivateSpaceResource_Metadata(b *testing.B) {
	res := NewPrivateSpaceResource()
	ctx := context.Background()
	req := resource.MetadataRequest{}
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resp := &resource.MetadataResponse{}
		res.Metadata(ctx, req, resp)
	}
}