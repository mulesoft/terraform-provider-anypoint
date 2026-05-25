package accessmanagement

import (
	"context"
	"net/http"
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestOrganizationClient_DeleteOrganization(t *testing.T) {
	tests := []struct {
		name        string
		orgID       string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful delete with 200",
			orgID: "org-to-delete",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE, got %s", r.Method)
				}
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:  "successful delete with 204",
			orgID: "org-to-delete",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:  "successful delete with 202",
			orgID: "org-to-delete",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusAccepted)
			},
			wantErr: false,
		},
		{
			name:  "not found returns NotFoundError",
			orgID: "missing-org",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "organization not found")
			},
			wantErr:     true,
			errContains: "organization not found",
		},
		{
			name:  "server error",
			orgID: "org-to-delete",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "internal server error")
			},
			wantErr:     true,
			errContains: "failed to delete organization with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				"/accounts/api/organizations/" + tt.orgID: tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &OrganizationClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					HTTPClient: &http.Client{},
				},
			}

			err := c.DeleteOrganization(context.Background(), tt.orgID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteOrganization() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteOrganization() error = %v, want containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteOrganization() unexpected error = %v", err)
				}
			}
		})
	}
}
