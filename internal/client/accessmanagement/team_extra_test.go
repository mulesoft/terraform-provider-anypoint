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

func TestTeamClient_UpdateTeamParent(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		teamID      string
		request     *UpdateTeamParentRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:   "successful update with 200",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &UpdateTeamParentRequest{
				ParentTeamID: "new-parent-team-id",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodPut {
					t.Errorf("Expected PUT, got %s", r.Method)
				}
				body := testutil.AssertJSONBody(t, r, "parent_team_id")
				if body["parent_team_id"] != "new-parent-team-id" {
					t.Errorf("Expected parent_team_id=new-parent-team-id, got %v", body["parent_team_id"])
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:   "successful update with 201",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &UpdateTeamParentRequest{
				ParentTeamID: "new-parent-team-id",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusCreated)
			},
			wantErr: false,
		},
		{
			name:   "not found returns NotFoundError",
			orgID:  "test-org-id",
			teamID: "missing-team",
			request: &UpdateTeamParentRequest{
				ParentTeamID: "some-parent",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "team not found")
			},
			wantErr:     true,
			errContains: "team not found",
		},
		{
			name:   "server error",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &UpdateTeamParentRequest{
				ParentTeamID: "some-parent",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "internal error")
			},
			wantErr:     true,
			errContains: "failed to update team parent with status 500",
		},
		{
			name:   "bad request",
			orgID:  "test-org-id",
			teamID: "test-team-id",
			request: &UpdateTeamParentRequest{
				ParentTeamID: "invalid",
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "invalid parent")
			},
			wantErr:     true,
			errContains: "failed to update team parent with status 400",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/%s/teams/%s/parent", tt.orgID, tt.teamID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &TeamClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := c.UpdateTeamParent(context.Background(), tt.orgID, tt.teamID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateTeamParent() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateTeamParent() error = %v, want containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateTeamParent() unexpected error = %v", err)
				}
			}
		})
	}
}
