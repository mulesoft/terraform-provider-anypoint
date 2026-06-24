package accessmanagement

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRolePermissionResource(t *testing.T) {
	r := NewRolePermissionResource()

	if r == nil {
		t.Error("NewRolePermissionResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("RolePermissionResource should implement ResourceWithConfigure")
	}
}

func TestRolePermissionResource_Metadata(t *testing.T) {
	r := NewRolePermissionResource()
	testutil.TestResourceMetadata(t, r, "_role_permission")
}

func TestRolePermissionResource_Schema(t *testing.T) {
	res := NewRolePermissionResource()

	requiredAttrs := []string{"role_group_id", "role_id", "context_params"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id", "role_name", "created_at"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestRolePermissionResource_Configure(t *testing.T) {
	res := NewRolePermissionResource().(*RolePermissionResource)

	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	providerData := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	testutil.TestResourceConfigure(t, res, providerData)

	if res.client == nil {
		t.Error("Configure() should set client")
	}
}

func TestRolePermissionResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewRolePermissionResource().(*RolePermissionResource)

	ctx := context.Background()
	req := resource.ConfigureRequest{
		ProviderData: "invalid-data",
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

func TestRolePermissionResource_Read(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data": []map[string]interface{}{
					{
						"role_group_assignment_id": "assign-123",
						"role_group_id":            "test-rg-id",
						"role_id":                  "role-abc",
						"org_id":                   "test-org-id",
						"name":                     "Audit Log Viewer",
						"description":              "View audit logs",
						"internal":                 false,
						"context_params":           map[string]string{"org": "test-org-id"},
						"created_at":               "2024-01-01T00:00:00Z",
					},
				},
				"total": 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRolePermissionResource().(*RolePermissionResource)
	res.client = &accessmanagement.RolePermissionClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "assign-123"),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"role_id":         tftypes.NewValue(tftypes.String, "role-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"context_params": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
			"org": tftypes.NewValue(tftypes.String, "test-org-id"),
		}),
		"role_name":  tftypes.NewValue(tftypes.String, "Audit Log Viewer"),
		"created_at": tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}
	var got RolePermissionResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "assign-123" {
		t.Errorf("Expected ID 'assign-123', got %s", got.ID.ValueString())
	}
	if got.RoleName.ValueString() != "Audit Log Viewer" {
		t.Errorf("Expected RoleName 'Audit Log Viewer', got %s", got.RoleName.ValueString())
	}
}

func TestRolePermissionResource_Read_NotFound(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			// Return empty list — the specific assignment doesn't exist
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"data":  []map[string]interface{}{},
				"total": 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRolePermissionResource().(*RolePermissionResource)
	res.client = &accessmanagement.RolePermissionClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	priorStateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "assign-123"),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"role_id":         tftypes.NewValue(tftypes.String, "role-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"context_params": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
			"org": tftypes.NewValue(tftypes.String, "test-org-id"),
		}),
		"role_name":  tftypes.NewValue(tftypes.String, ""),
		"created_at": tftypes.NewValue(tftypes.String, ""),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	// Should not have errors - it should just remove the resource from state
	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() should not have errors on 404, got: %v", resp.Diagnostics.Errors())
	}

	// State should be empty (resource removed)
	if !resp.State.Raw.IsNull() {
		t.Error("Read() should remove resource from state when assignment not found")
	}
}

func TestRolePermissionResource_Create(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/roles"
	callCount := 0

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			callCount++
			if r.Method == "POST" {
				// Verify the request body
				body, _ := io.ReadAll(r.Body)
				var req []accessmanagement.AssignRoleRequest
				json.Unmarshal(body, &req)
				if len(req) != 1 || req[0].RoleID != "role-abc" {
					t.Errorf("Unexpected POST body: %s", string(body))
				}

				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode([]accessmanagement.AssignRoleResponse{{
					RoleGroupID:           "test-rg-id",
					RoleID:                "role-abc",
					RoleGroupAssignmentID: "new-assign-id",
					ContextParams:         map[string]string{"org": "test-org-id"},
				}})
			} else if r.Method == "GET" {
				// Read-back after create
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"data": []map[string]interface{}{
						{
							"role_group_assignment_id": "new-assign-id",
							"role_group_id":            "test-rg-id",
							"role_id":                  "role-abc",
							"org_id":                   "test-org-id",
							"name":                     "Audit Log Viewer",
							"description":              "View audit logs",
							"context_params":           map[string]string{"org": "test-org-id"},
							"created_at":               "2024-06-23T14:00:00Z",
						},
					},
					"total": 1,
				})
			}
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRolePermissionResource().(*RolePermissionResource)
	res.client = &accessmanagement.RolePermissionClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	planRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, nil),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"role_id":         tftypes.NewValue(tftypes.String, "role-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"context_params": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
			"org": tftypes.NewValue(tftypes.String, "test-org-id"),
		}),
		"role_name":  tftypes.NewValue(tftypes.String, nil),
		"created_at": tftypes.NewValue(tftypes.String, nil),
	})

	createReq := resource.CreateRequest{
		Plan: tfsdk.Plan{Schema: schemaResp.Schema, Raw: planRaw},
	}
	createResp := &resource.CreateResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: tftypes.NewValue(stateType, nil)},
	}
	res.Create(ctx, createReq, createResp)

	if createResp.Diagnostics.HasError() {
		t.Fatalf("Create() reported errors: %v", createResp.Diagnostics.Errors())
	}

	var got RolePermissionResourceModel
	if diags := createResp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "new-assign-id" {
		t.Errorf("Expected ID 'new-assign-id', got %s", got.ID.ValueString())
	}
	if got.RoleName.ValueString() != "Audit Log Viewer" {
		t.Errorf("Expected RoleName 'Audit Log Viewer', got %s", got.RoleName.ValueString())
	}
}

func TestRolePermissionResource_Delete(t *testing.T) {
	basePath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/roles"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		basePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("Expected DELETE, got %s", r.Method)
			}

			body, _ := io.ReadAll(r.Body)
			var req []accessmanagement.AssignRoleRequest
			json.Unmarshal(body, &req)
			if len(req) != 1 || req[0].RoleID != "role-abc" {
				t.Errorf("Unexpected DELETE body: %s", string(body))
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode([]int{1})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRolePermissionResource().(*RolePermissionResource)
	res.client = &accessmanagement.RolePermissionClient{
		UserAnypointClient: &client.UserAnypointClient{
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

	stateRaw := tftypes.NewValue(stateType, map[string]tftypes.Value{
		"id":              tftypes.NewValue(tftypes.String, "assign-123"),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"role_id":         tftypes.NewValue(tftypes.String, "role-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"context_params": tftypes.NewValue(tftypes.Map{ElementType: tftypes.String}, map[string]tftypes.Value{
			"org": tftypes.NewValue(tftypes.String, "test-org-id"),
		}),
		"role_name":  tftypes.NewValue(tftypes.String, "Audit Log Viewer"),
		"created_at": tftypes.NewValue(tftypes.String, "2024-01-01T00:00:00Z"),
	})

	deleteReq := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	deleteResp := &resource.DeleteResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	res.Delete(ctx, deleteReq, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete() reported errors: %v", deleteResp.Diagnostics.Errors())
	}
}
