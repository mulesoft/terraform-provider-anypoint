package accessmanagement

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewTeamMembersClient(t *testing.T) {
	tests := []struct {
		name        string
		config      *client.ClientConfig
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &client.ClientConfig{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.ClientConfig{
				ClientSecret: "test-client-secret",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.ClientConfig{
				ClientID: "test-client-id",
			},
			wantErr:     true,
			errContains: "client_secret is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())

			if tt.config != nil {
				tt.config.BaseURL = server.URL
			}

			c, err := NewTeamMembersClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTeamMembersClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewTeamMembersClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewTeamMembersClient() unexpected error = %v", err)
				}
				if c == nil {
					t.Errorf("NewTeamMembersClient() returned nil client")
				}
			}
		})
	}
}

func TestTeamMembersClient_AddMembersToTeam(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		teamID      string
		members     []TeamMember
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful addition",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			members: []TeamMember{
				{ID: "user-1", MembershipType: "member"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "PATCH" {
					t.Errorf("Expected PATCH request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			members: []TeamMember{
				{ID: "user-1", MembershipType: "member"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to add members to team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/members", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TeamMembersClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := c.AddMembersToTeam(context.Background(), tt.orgID, tt.teamID, tt.members)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddMembersToTeam() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AddMembersToTeam() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AddMembersToTeam() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTeamMembersClient_GetTeamMembers(t *testing.T) {
	mockResponse := TeamMemberResponse{
		Data: []TeamMemberDetails{
			{ID: "user-1", Username: "testuser", MembershipType: "member"},
		},
	}

	tests := []struct {
		name        string
		orgID       string
		teamID      string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful get",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != "GET" {
					t.Errorf("Expected GET request, got %s", r.Method)
				}
				testutil.JSONResponse(w, http.StatusOK, mockResponse)
			},
			wantErr: false,
		},
		{
			name:   "not found returns empty",
			orgID:  "test-org-id",
			teamID: "nonexistent-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNotFound)
			},
			wantErr: false,
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get members for team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/members", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TeamMembersClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			result, err := c.GetTeamMembers(context.Background(), tt.orgID, tt.teamID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTeamMembers() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetTeamMembers() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetTeamMembers() unexpected error = %v", err)
				}
				if result == nil {
					t.Errorf("GetTeamMembers() returned nil")
				}
			}
		})
	}
}

func TestTeamMemberDetails_JSONSerialization(t *testing.T) {
	item := &TeamMemberDetails{
		ID:             "test-user-id",
		Username:       "testuser",
		MembershipType: "member",
	}

	data, err := json.Marshal(item)
	if err != nil {
		t.Fatalf("Failed to marshal TeamMemberDetails: %v", err)
	}

	var decoded TeamMemberDetails
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal TeamMemberDetails: %v", err)
	}

	if decoded.ID != item.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, item.ID)
	}
	if decoded.Username != item.Username {
		t.Errorf("Unmarshaled Username = %v, want %v", decoded.Username, item.Username)
	}
	if decoded.MembershipType != item.MembershipType {
		t.Errorf("Unmarshaled MembershipType = %v, want %v", decoded.MembershipType, item.MembershipType)
	}
}
