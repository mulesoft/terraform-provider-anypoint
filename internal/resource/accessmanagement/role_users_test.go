package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/tfsdk"
	"github.com/hashicorp/terraform-plugin-go/tftypes"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewRoleUsersResource(t *testing.T) {
	r := NewRoleUsersResource()

	if r == nil {
		t.Error("NewRoleUsersResource() returned nil")
	}

	if _, ok := r.(resource.ResourceWithConfigure); !ok {
		t.Error("RoleUsersResource should implement ResourceWithConfigure")
	}
}

func TestRoleUsersResource_Metadata(t *testing.T) {
	r := NewRoleUsersResource()
	testutil.TestResourceMetadata(t, r, "_role_users")
}

func TestRoleUsersResource_Schema(t *testing.T) {
	res := NewRoleUsersResource()

	requiredAttrs := []string{"role_group_id", "user_id"}
	optionalAttrs := []string{"organization_id"}
	computedAttrs := []string{"id", "username", "first_name", "last_name", "email"}

	testutil.TestResourceSchema(t, res, requiredAttrs, optionalAttrs, computedAttrs)
}

func TestRoleUsersResource_Configure(t *testing.T) {
	res := NewRoleUsersResource().(*RoleUsersResource)

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

func TestRoleUsersResource_Configure_InvalidProviderData(t *testing.T) {
	res := NewRoleUsersResource().(*RoleUsersResource)

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

func TestRoleUsersResource_Create(t *testing.T) {
	addPath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/users/user-abc"
	listPath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		addPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "POST" {
				t.Errorf("Expected POST to add path, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("null"))
		},
		listPath: func(w http.ResponseWriter, r *http.Request) {
			// Read-back after create
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(accessmanagement.ListRoleGroupUsersResponse{
				Data: []accessmanagement.RoleGroupUser{
					{
						ID:        "user-abc",
						Username:  "madhav",
						FirstName: "Madhav",
						LastName:  "Aggarwal",
						Email:     "madhav@example.com",
					},
				},
				Total: 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRoleUsersResource().(*RoleUsersResource)
	res.client = &accessmanagement.RoleUsersClient{
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
		"user_id":         tftypes.NewValue(tftypes.String, "user-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"username":        tftypes.NewValue(tftypes.String, nil),
		"first_name":      tftypes.NewValue(tftypes.String, nil),
		"last_name":       tftypes.NewValue(tftypes.String, nil),
		"email":           tftypes.NewValue(tftypes.String, nil),
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

	var got RoleUsersResourceModel
	if diags := createResp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.ID.ValueString() != "test-rg-id:user-abc" {
		t.Errorf("Expected ID 'test-rg-id:user-abc', got %s", got.ID.ValueString())
	}
	if got.Username.ValueString() != "madhav" {
		t.Errorf("Expected username 'madhav', got %s", got.Username.ValueString())
	}
	if got.Email.ValueString() != "madhav@example.com" {
		t.Errorf("Expected email 'madhav@example.com', got %s", got.Email.ValueString())
	}
}

func TestRoleUsersResource_Read(t *testing.T) {
	listPath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(accessmanagement.ListRoleGroupUsersResponse{
				Data: []accessmanagement.RoleGroupUser{
					{
						ID:        "user-abc",
						Username:  "madhav",
						FirstName: "Madhav",
						LastName:  "Aggarwal",
						Email:     "madhav@example.com",
					},
				},
				Total: 1,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRoleUsersResource().(*RoleUsersResource)
	res.client = &accessmanagement.RoleUsersClient{
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
		"id":              tftypes.NewValue(tftypes.String, "test-rg-id:user-abc"),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"user_id":         tftypes.NewValue(tftypes.String, "user-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"username":        tftypes.NewValue(tftypes.String, "madhav"),
		"first_name":      tftypes.NewValue(tftypes.String, "Madhav"),
		"last_name":       tftypes.NewValue(tftypes.String, "Aggarwal"),
		"email":           tftypes.NewValue(tftypes.String, "madhav@example.com"),
	})

	req := resource.ReadRequest{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	resp := &resource.ReadResponse{State: tfsdk.State{Schema: schemaResp.Schema, Raw: priorStateRaw}}
	res.Read(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Fatalf("Read() reported errors: %v", resp.Diagnostics.Errors())
	}

	var got RoleUsersResourceModel
	if diags := resp.State.Get(ctx, &got); diags.HasError() {
		t.Fatalf("State.Get errors: %v", diags.Errors())
	}
	if got.Username.ValueString() != "madhav" {
		t.Errorf("Expected username 'madhav', got %s", got.Username.ValueString())
	}
}

func TestRoleUsersResource_Read_NotFound(t *testing.T) {
	listPath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/users"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		listPath: func(w http.ResponseWriter, r *http.Request) {
			// Return empty list — user not in role group
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(accessmanagement.ListRoleGroupUsersResponse{
				Data:  []accessmanagement.RoleGroupUser{},
				Total: 0,
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRoleUsersResource().(*RoleUsersResource)
	res.client = &accessmanagement.RoleUsersClient{
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
		"id":              tftypes.NewValue(tftypes.String, "test-rg-id:user-abc"),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"user_id":         tftypes.NewValue(tftypes.String, "user-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"username":        tftypes.NewValue(tftypes.String, "madhav"),
		"first_name":      tftypes.NewValue(tftypes.String, "Madhav"),
		"last_name":       tftypes.NewValue(tftypes.String, "Aggarwal"),
		"email":           tftypes.NewValue(tftypes.String, "madhav@example.com"),
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
		t.Error("Read() should remove resource from state when user not found in role group")
	}
}

func TestRoleUsersResource_Delete(t *testing.T) {
	deletePath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/users/user-abc"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		deletePath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != "DELETE" {
				t.Errorf("Expected DELETE, got %s", r.Method)
			}
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("null"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRoleUsersResource().(*RoleUsersResource)
	res.client = &accessmanagement.RoleUsersClient{
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
		"id":              tftypes.NewValue(tftypes.String, "test-rg-id:user-abc"),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"user_id":         tftypes.NewValue(tftypes.String, "user-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"username":        tftypes.NewValue(tftypes.String, "madhav"),
		"first_name":      tftypes.NewValue(tftypes.String, "Madhav"),
		"last_name":       tftypes.NewValue(tftypes.String, "Aggarwal"),
		"email":           tftypes.NewValue(tftypes.String, "madhav@example.com"),
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

func TestRoleUsersResource_Delete_AlreadyRemoved(t *testing.T) {
	// 409 = user not in group — should be idempotent success
	deletePath := "/accounts/api/organizations/test-org-id/rolegroups/test-rg-id/users/user-abc"

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		deletePath: func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusConflict)
			w.Write([]byte("Could not unassign role group"))
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	res := NewRoleUsersResource().(*RoleUsersResource)
	res.client = &accessmanagement.RoleUsersClient{
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
		"id":              tftypes.NewValue(tftypes.String, "test-rg-id:user-abc"),
		"role_group_id":   tftypes.NewValue(tftypes.String, "test-rg-id"),
		"user_id":         tftypes.NewValue(tftypes.String, "user-abc"),
		"organization_id": tftypes.NewValue(tftypes.String, "test-org-id"),
		"username":        tftypes.NewValue(tftypes.String, "madhav"),
		"first_name":      tftypes.NewValue(tftypes.String, "Madhav"),
		"last_name":       tftypes.NewValue(tftypes.String, "Aggarwal"),
		"email":           tftypes.NewValue(tftypes.String, "madhav@example.com"),
	})

	deleteReq := resource.DeleteRequest{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	deleteResp := &resource.DeleteResponse{
		State: tfsdk.State{Schema: schemaResp.Schema, Raw: stateRaw},
	}
	res.Delete(ctx, deleteReq, deleteResp)

	if deleteResp.Diagnostics.HasError() {
		t.Fatalf("Delete() should not error on 409 (idempotent), got: %v", deleteResp.Diagnostics.Errors())
	}
}
