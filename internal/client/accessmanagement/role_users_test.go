package accessmanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

func TestNewRoleUsersClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/accounts/api/v2/oauth2/token":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"access_token": "mock-token",
				"token_type":   "bearer",
			})
		case "/accounts/api/me":
			w.Header().Set("Content-Type", "application/json")
			_ = json.NewEncoder(w).Encode(map[string]interface{}{
				"user": map[string]interface{}{
					"id":       "user-123",
					"username": "test-user",
					"organization": map[string]interface{}{
						"id":   "org-123",
						"name": "Test Org",
					},
				},
				"client": map[string]interface{}{
					"org_id": "org-123",
				},
			})
		}
	}))
	defer server.Close()

	config := &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
		Username:     "test-user",
		Password:     "test-password",
	}

	c, err := NewRoleUsersClient(config)
	if err != nil {
		t.Fatalf("NewRoleUsersClient() error = %v", err)
	}
	if c == nil {
		t.Fatal("NewRoleUsersClient() returned nil")
	}
}

func TestRoleUsersClient_AddUserToRoleGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/accounts/api/organizations/test-org/rolegroups/test-rg/users/user-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("null"))
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	err := c.AddUserToRoleGroup(context.Background(), "test-org", "test-rg", "user-123")
	if err != nil {
		t.Fatalf("AddUserToRoleGroup() error = %v", err)
	}
}

func TestRoleUsersClient_AddUserToRoleGroup_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusNotFound)
		_, _ = w.Write([]byte("This resource does not exist"))
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	err := c.AddUserToRoleGroup(context.Background(), "test-org", "test-rg", "nonexistent-user")
	if err == nil {
		t.Fatal("AddUserToRoleGroup() should error on 404")
	}
	if !client.IsNotFound(err) {
		t.Errorf("Expected NotFoundError, got: %v", err)
	}
}

func TestRoleUsersClient_RemoveUserFromRoleGroup(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		expectedPath := "/accounts/api/organizations/test-org/rolegroups/test-rg/users/user-123"
		if r.URL.Path != expectedPath {
			t.Errorf("Expected path %s, got %s", expectedPath, r.URL.Path)
		}
		if r.Method != http.MethodDelete {
			t.Errorf("Expected DELETE, got %s", r.Method)
		}
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("null"))
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	err := c.RemoveUserFromRoleGroup(context.Background(), "test-org", "test-rg", "user-123")
	if err != nil {
		t.Fatalf("RemoveUserFromRoleGroup() error = %v", err)
	}
}

func TestRoleUsersClient_RemoveUserFromRoleGroup_Conflict(t *testing.T) {
	// 409 means user not in group — should be treated as success (idempotent)
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusConflict)
		_, _ = w.Write([]byte("Could not unassign role group"))
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	err := c.RemoveUserFromRoleGroup(context.Background(), "test-org", "test-rg", "user-123")
	if err != nil {
		t.Fatalf("RemoveUserFromRoleGroup() should not error on 409 (idempotent), got: %v", err)
	}
}

func TestRoleUsersClient_ListRoleGroupUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListRoleGroupUsersResponse{
			Data: []RoleGroupUser{
				{
					ID:             "user-aaa",
					Username:       "alice",
					FirstName:      "Alice",
					LastName:       "Smith",
					Email:          "alice@example.com",
					OrganizationID: "test-org",
					Enabled:        true,
					Type:           "host",
				},
				{
					ID:             "user-bbb",
					Username:       "bob",
					FirstName:      "Bob",
					LastName:       "Jones",
					Email:          "bob@example.com",
					OrganizationID: "test-org",
					Enabled:        true,
					Type:           "host",
				},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	users, err := c.ListRoleGroupUsers(context.Background(), "test-org", "test-rg")
	if err != nil {
		t.Fatalf("ListRoleGroupUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}
	if users[0].Username != "alice" {
		t.Errorf("Expected first user 'alice', got %s", users[0].Username)
	}
}

func TestRoleUsersClient_GetRoleGroupUser(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListRoleGroupUsersResponse{
			Data: []RoleGroupUser{
				{ID: "user-aaa", Username: "alice", Email: "alice@example.com"},
				{ID: "user-bbb", Username: "bob", Email: "bob@example.com"},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	user, err := c.GetRoleGroupUser(context.Background(), "test-org", "test-rg", "user-bbb")
	if err != nil {
		t.Fatalf("GetRoleGroupUser() error = %v", err)
	}
	if user.Username != "bob" {
		t.Errorf("Expected username 'bob', got %s", user.Username)
	}
}

func TestRoleUsersClient_GetRoleGroupUser_NotFound(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListRoleGroupUsersResponse{
			Data:  []RoleGroupUser{},
			Total: 0,
		})
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	_, err := c.GetRoleGroupUser(context.Background(), "test-org", "test-rg", "nonexistent")
	if err == nil {
		t.Fatal("GetRoleGroupUser() should error when user not found")
	}
	if !client.IsNotFound(err) {
		t.Errorf("Expected NotFoundError, got: %v", err)
	}
}

func TestRoleUsersClient_ListOrgUsers(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(ListOrgUsersResponse{
			Data: []OrgUser{
				{ID: "user-aaa", Username: "alice", FirstName: "Alice", LastName: "Smith", Email: "alice@example.com", Enabled: true},
				{ID: "user-bbb", Username: "bob", FirstName: "Bob", LastName: "Jones", Email: "bob@example.com", Enabled: true},
			},
			Total: 2,
		})
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	users, err := c.ListOrgUsers(context.Background(), "test-org")
	if err != nil {
		t.Fatalf("ListOrgUsers() error = %v", err)
	}
	if len(users) != 2 {
		t.Fatalf("Expected 2 users, got %d", len(users))
	}
}

func TestRoleUsersClient_ListOrgUsers_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
		_, _ = w.Write([]byte("internal error"))
	}))
	defer server.Close()

	c := &RoleUsersClient{
		AnypointClient: &client.AnypointClient{
			BaseURL:    server.URL,
			Token:      "mock-token",
			HTTPClient: &http.Client{},
			OrgID:      "test-org",
		},
	}

	_, err := c.ListOrgUsers(context.Background(), "test-org")
	if err == nil {
		t.Fatal("ListOrgUsers() should error on server error")
	}
}
