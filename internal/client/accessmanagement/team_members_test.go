package accessmanagement

import (
	"context"
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
		config      *client.Config
		wantErr     bool
		errContains string
	}{
		{
			name: "valid config",
			config: &client.Config{
				ClientID:     "test-client-id",
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
				Username:     "test-user",
				Password:     "test-password",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())

			if tt.config != nil {
				tt.config.BaseURL = server.URL
			}

			client, err := NewTeamMembersClient(tt.config)

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
				if client == nil {
					t.Errorf("NewTeamMembersClient() returned nil client")
				}
			}
		})
	}
}

func TestTeamMembersClient_AddTeamMember(t *testing.T) {
	tests := []struct {
		name           string
		orgID          string
		teamID         string
		userID         string
		membershipType string
		mockHandler    func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:           "successful add member",
			orgID:          "test-org-id",
			teamID:         "test-team-id",
			userID:         "test-user-id",
			membershipType: "member",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/accounts/api/organizations/test-org-id/teams/test-team-id/members/test-user-id")
				body := testutil.AssertJSONBody(t, r, "membership_type")
				if body["membership_type"] != "member" {
					t.Errorf("Expected membership_type 'member', got %v", body["membership_type"])
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:           "add maintainer",
			orgID:          "test-org-id",
			teamID:         "test-team-id",
			userID:         "test-user-id",
			membershipType: "maintainer",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/accounts/api/organizations/test-org-id/teams/test-team-id/members/test-user-id")
				body := testutil.AssertJSONBody(t, r, "membership_type")
				if body["membership_type"] != "maintainer" {
					t.Errorf("Expected membership_type 'maintainer', got %v", body["membership_type"])
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:           "team not found",
			orgID:          "test-org-id",
			teamID:         "nonexistent-team-id",
			userID:         "test-user-id",
			membershipType: "member",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
			},
			wantErr:     true,
			errContains: "team or user",
		},
		{
			name:           "server error",
			orgID:          "test-org-id",
			teamID:         "test-team-id",
			userID:         "test-user-id",
			membershipType: "member",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to add member to team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/members/%s", tt.orgID, tt.teamID, tt.userID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamMembersClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.AddTeamMember(context.Background(), tt.orgID, tt.teamID, tt.userID, tt.membershipType)

			if tt.wantErr {
				if err == nil {
					t.Errorf("AddTeamMember() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("AddTeamMember() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("AddTeamMember() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTeamMembersClient_RemoveTeamMember(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		teamID      string
		userID      string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful removal",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/teams/test-team-id/members/test-user-id")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:   "already removed - not found",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			userID: "nonexistent-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Member not found")
			},
			wantErr: false,
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			userID: "test-user-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to remove member from team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/members/%s", tt.orgID, tt.teamID, tt.userID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamMembersClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.RemoveTeamMember(context.Background(), tt.orgID, tt.teamID, tt.userID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveTeamMember() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("RemoveTeamMember() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("RemoveTeamMember() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTeamMembersClient_ListTeamMembers(t *testing.T) {
	mockMembers := []TeamMember{
		{
			ID:                          "user-1",
			IdentityType:                "user",
			Name:                        "User One",
			MembershipType:              "member",
			IsAssignedViaExternalGroups: false,
			CreatedAt:                   "2023-01-01T00:00:00Z",
			UpdatedAt:                   "2023-01-01T00:00:00Z",
		},
		{
			ID:                          "user-2",
			IdentityType:                "user",
			Name:                        "User Two",
			MembershipType:              "maintainer",
			IsAssignedViaExternalGroups: false,
			CreatedAt:                   "2023-01-02T00:00:00Z",
			UpdatedAt:                   "2023-01-02T00:00:00Z",
		},
	}

	tests := []struct {
		name          string
		orgID         string
		teamID        string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errContains   string
		expectedCount int
	}{
		{
			name:   "successful list",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/teams/test-team-id/members")
				testutil.JSONResponse(w, http.StatusOK, ListTeamMembersResponse{
					Data:  mockMembers,
					Total: 2,
				})
			},
			wantErr:       false,
			expectedCount: 2,
		},
		{
			name:   "team not found",
			orgID:  "test-org-id",
			teamID: "nonexistent-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
			},
			wantErr:     true,
			errContains: "team",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to list team members with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/members", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamMembersClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			members, err := client.ListTeamMembers(context.Background(), tt.orgID, tt.teamID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("ListTeamMembers() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("ListTeamMembers() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("ListTeamMembers() unexpected error = %v", err)
				}
				if len(members) != tt.expectedCount {
					t.Errorf("ListTeamMembers() returned %d members, want %d", len(members), tt.expectedCount)
				}
			}
		})
	}
}
