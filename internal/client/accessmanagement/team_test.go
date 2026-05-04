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

func TestNewTeamClient(t *testing.T) {
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
			},
			wantErr: false,
		},
		{
			name: "missing client ID",
			config: &client.Config{
				ClientSecret: "test-client-secret",
			},
			wantErr:     true,
			errContains: "client_id is required",
		},
		{
			name: "missing client secret",
			config: &client.Config{
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

			client, err := NewTeamClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewTeamClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewTeamClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewTeamClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewTeamClient() returned nil client")
				}
			}
		})
	}
}

func TestTeamClient_CreateTeam(t *testing.T) {
	mockTeam := &Team{
		ID:        "test-team-id",
		TeamName:  "Test Team",
		OrgID:     "test-org-id",
		TeamType:  "internal",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-01-01T00:00:00Z",
	}

	tests := []struct {
		name         string
		orgID        string
		request      *CreateTeamRequest
		mockHandler  func(w http.ResponseWriter, r *http.Request)
		wantErr      bool
		errContains  string
		expectedTeam *Team
	}{
		{
			name:  "successful creation",
			orgID: "test-org-id",
			request: &CreateTeamRequest{
				TeamName: "Test Team",
				TeamType: "internal",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/teams")

				body := testutil.AssertJSONBody(t, r, "team_name", "team_type")

				if body["team_name"] != "Test Team" {
					t.Errorf("Expected team_name 'Test Team', got %v", body["team_name"])
				}
				if body["team_type"] != "internal" {
					t.Errorf("Expected team_type 'internal', got %v", body["team_type"])
				}

				testutil.JSONResponse(w, http.StatusCreated, mockTeam)
			},
			wantErr:      false,
			expectedTeam: mockTeam,
		},
		{
			name:  "successful creation with parent team",
			orgID: "test-org-id",
			request: &CreateTeamRequest{
				TeamName:     "Child Team",
				ParentTeamID: "parent-team-id",
				TeamType:     "internal",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "POST", "/accounts/api/organizations/test-org-id/teams")

				body := testutil.AssertJSONBody(t, r, "team_name", "team_type", "parent_team_id")

				if body["team_name"] != "Child Team" {
					t.Errorf("Expected team_name 'Child Team', got %v", body["team_name"])
				}
				if body["parent_team_id"] != "parent-team-id" {
					t.Errorf("Expected parent_team_id 'parent-team-id', got %v", body["parent_team_id"])
				}

				testutil.JSONResponse(w, http.StatusCreated, &Team{
					ID:        "child-team-id",
					TeamName:  "Child Team",
					OrgID:     "test-org-id",
					TeamType:  "internal",
					CreatedAt: "2023-01-01T00:00:00Z",
					UpdatedAt: "2023-01-01T00:00:00Z",
				})
			},
			wantErr: false,
		},
		{
			name:  "team already exists",
			orgID: "test-org-id",
			request: &CreateTeamRequest{
				TeamName: "Existing Team",
				TeamType: "internal",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusConflict, "Team already exists")
			},
			wantErr:     true,
			errContains: "failed to create team with status 409",
		},
		{
			name:  "invalid parent team",
			orgID: "test-org-id",
			request: &CreateTeamRequest{
				TeamName:     "Child Team",
				ParentTeamID: "nonexistent-parent",
				TeamType:     "internal",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Invalid parent team")
			},
			wantErr:     true,
			errContains: "failed to create team with status 400",
		},
		{
			name:  "server error",
			orgID: "test-org-id",
			request: &CreateTeamRequest{
				TeamName: "Test Team",
				TeamType: "internal",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to create team with status 500",
		},
		{
			name:  "malformed response",
			orgID: "test-org-id",
			request: &CreateTeamRequest{
				TeamName: "Test Team",
				TeamType: "internal",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusCreated)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams", tt.orgID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			team, err := client.CreateTeam(context.Background(), tt.orgID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("CreateTeam() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("CreateTeam() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("CreateTeam() unexpected error = %v", err)
				}
				if team == nil {
					t.Errorf("CreateTeam() returned nil team")
				}

				// Validate returned team
				if team != nil && tt.expectedTeam != nil {
					if team.ID != tt.expectedTeam.ID {
						t.Errorf("CreateTeam() ID = %v, want %v", team.ID, tt.expectedTeam.ID)
					}
					if team.TeamName != tt.expectedTeam.TeamName {
						t.Errorf("CreateTeam() TeamName = %v, want %v", team.TeamName, tt.expectedTeam.TeamName)
					}
					if team.OrgID != tt.expectedTeam.OrgID {
						t.Errorf("CreateTeam() OrgID = %v, want %v", team.OrgID, tt.expectedTeam.OrgID)
					}
					if team.TeamType != tt.expectedTeam.TeamType {
						t.Errorf("CreateTeam() TeamType = %v, want %v", team.TeamType, tt.expectedTeam.TeamType)
					}
				}
			}
		})
	}
}

func TestTeamClient_GetTeam(t *testing.T) {
	mockTeam := &Team{
		ID:        "test-team-id",
		TeamName:  "Test Team",
		OrgID:     "test-org-id",
		TeamType:  "internal",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-01-01T00:00:00Z",
	}

	tests := []struct {
		name         string
		orgID        string
		teamID       string
		mockHandler  func(w http.ResponseWriter, r *http.Request)
		wantErr      bool
		errContains  string
		expectedTeam *Team
	}{
		{
			name:   "successful get",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/accounts/api/organizations/test-org-id/teams/test-team-id")
				testutil.JSONResponse(w, http.StatusOK, mockTeam)
			},
			wantErr:      false,
			expectedTeam: mockTeam,
		},
		{
			name:   "team not found",
			orgID:  "test-org-id",
			teamID: "nonexistent-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
			},
			wantErr:     true,
			errContains: "team not found",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get team with status 500",
		},
		{
			name:   "malformed response",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				w.WriteHeader(http.StatusOK)
				_, _ = w.Write([]byte(`{"invalid": json}`))
			},
			wantErr:     true,
			errContains: "failed to decode response",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			team, err := client.GetTeam(context.Background(), tt.orgID, tt.teamID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetTeam() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetTeam() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetTeam() unexpected error = %v", err)
				}
				if team == nil {
					t.Errorf("GetTeam() returned nil team")
				}

				// Validate returned team
				if team != nil && tt.expectedTeam != nil {
					if team.ID != tt.expectedTeam.ID {
						t.Errorf("GetTeam() ID = %v, want %v", team.ID, tt.expectedTeam.ID)
					}
					if team.TeamName != tt.expectedTeam.TeamName {
						t.Errorf("GetTeam() TeamName = %v, want %v", team.TeamName, tt.expectedTeam.TeamName)
					}
					if team.OrgID != tt.expectedTeam.OrgID {
						t.Errorf("GetTeam() OrgID = %v, want %v", team.OrgID, tt.expectedTeam.OrgID)
					}
					if team.TeamType != tt.expectedTeam.TeamType {
						t.Errorf("GetTeam() TeamType = %v, want %v", team.TeamType, tt.expectedTeam.TeamType)
					}
					if team.CreatedAt != tt.expectedTeam.CreatedAt {
						t.Errorf("GetTeam() CreatedAt = %v, want %v", team.CreatedAt, tt.expectedTeam.CreatedAt)
					}
					if team.UpdatedAt != tt.expectedTeam.UpdatedAt {
						t.Errorf("GetTeam() UpdatedAt = %v, want %v", team.UpdatedAt, tt.expectedTeam.UpdatedAt)
					}
				}
			}
		})
	}
}

func TestTeamClient_UpdateTeam(t *testing.T) {
	mockTeam := &Team{
		ID:        "test-team-id",
		TeamName:  "Updated Team",
		OrgID:     "test-org-id",
		TeamType:  "internal",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-01-02T00:00:00Z",
	}

	tests := []struct {
		name        string
		orgID       string
		teamID      string
		request     *UpdateTeamRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful update",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &UpdateTeamRequest{
				TeamName: testutil.StringPtr("Updated Team"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PATCH", "/accounts/api/organizations/test-org-id/teams/test-team-id")

				body := testutil.AssertJSONBody(t, r, "team_name")
				if body["team_name"] != "Updated Team" {
					t.Errorf("Expected team_name 'Updated Team', got %v", body["team_name"])
				}

				testutil.JSONResponse(w, http.StatusOK, mockTeam)
			},
			wantErr: false,
		},
		{
			name:   "team not found",
			orgID:  "test-org-id",
			teamID: "nonexistent-team-id",
			request: &UpdateTeamRequest{
				TeamName: testutil.StringPtr("Updated Team"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
			},
			wantErr:     true,
			errContains: "team not found",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &UpdateTeamRequest{
				TeamName: testutil.StringPtr("Updated Team"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to update team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			team, err := client.UpdateTeam(context.Background(), tt.orgID, tt.teamID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateTeam() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateTeam() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateTeam() unexpected error = %v", err)
				}
				if team == nil {
					t.Errorf("UpdateTeam() returned nil team")
				}
			}
		})
	}
}

func TestTeamClient_DeleteTeam(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		teamID      string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful deletion",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/accounts/api/organizations/test-org-id/teams/test-team-id")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:   "team not found",
			orgID:  "test-org-id",
			teamID: "nonexistent-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Team not found")
			},
			wantErr:     true,
			errContains: "Team not found",
		},
		{
			name:   "team has dependencies",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusConflict, "Team has dependencies")
			},
			wantErr:     true,
			errContains: "failed to delete team with status 409",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to delete team with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &TeamClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := client.DeleteTeam(context.Background(), tt.orgID, tt.teamID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteTeam() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteTeam() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteTeam() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestTeam_JSONSerialization(t *testing.T) {
	team := &Team{
		ID:        "test-team-id",
		TeamName:  "Test Team",
		OrgID:     "test-org-id",
		TeamType:  "internal",
		CreatedAt: "2023-01-01T00:00:00Z",
		UpdatedAt: "2023-01-01T00:00:00Z",
	}

	// Test marshaling
	data, err := json.Marshal(team)
	if err != nil {
		t.Fatalf("Failed to marshal team: %v", err)
	}

	// Test unmarshaling
	var decoded Team
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal team: %v", err)
	}

	// Validate key fields
	if decoded.ID != team.ID {
		t.Errorf("Unmarshaled ID = %v, want %v", decoded.ID, team.ID)
	}
	if decoded.TeamName != team.TeamName {
		t.Errorf("Unmarshaled TeamName = %v, want %v", decoded.TeamName, team.TeamName)
	}
	if decoded.OrgID != team.OrgID {
		t.Errorf("Unmarshaled OrgID = %v, want %v", decoded.OrgID, team.OrgID)
	}
	if decoded.TeamType != team.TeamType {
		t.Errorf("Unmarshaled TeamType = %v, want %v", decoded.TeamType, team.TeamType)
	}
}

func TestCreateTeamRequest_JSONSerialization(t *testing.T) {
	req := &CreateTeamRequest{
		TeamName:     "Test Team",
		ParentTeamID: "parent-team-id",
		TeamType:     "internal",
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal create request: %v", err)
	}

	// Test unmarshaling
	var decoded CreateTeamRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal create request: %v", err)
	}

	// Validate fields
	if decoded.TeamName != req.TeamName {
		t.Errorf("Unmarshaled TeamName = %v, want %v", decoded.TeamName, req.TeamName)
	}
	if decoded.ParentTeamID != req.ParentTeamID {
		t.Errorf("Unmarshaled ParentTeamID = %v, want %v", decoded.ParentTeamID, req.ParentTeamID)
	}
	if decoded.TeamType != req.TeamType {
		t.Errorf("Unmarshaled TeamType = %v, want %v", decoded.TeamType, req.TeamType)
	}
}
