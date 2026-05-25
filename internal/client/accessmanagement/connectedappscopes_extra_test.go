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

func TestConnectedAppScopesClient_RemoveConnectedAppScopes(t *testing.T) {
	tests := []struct {
		name        string
		appID       string
		scopes      []Scope
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:  "successful remove",
			appID: "test-app-id",
			scopes: []Scope{
				{Scope: "read:scopes", ContextParams: map[string]interface{}{}},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				if r.Method != http.MethodDelete {
					t.Errorf("Expected DELETE request, got %s", r.Method)
				}
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:  "server error",
			appID: "test-app-id",
			scopes: []Scope{
				{Scope: "read:scopes"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "API request failed with status 500",
		},
		{
			name:  "not found",
			appID: "missing-app",
			scopes: []Scope{
				{Scope: "read:scopes"},
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Not found")
			},
			wantErr:     true,
			errContains: "API request failed with status 404",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/organizations/test-org-id/connectedApplications/%s/scopes", tt.appID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &ConnectedAppScopesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					OrgID:      "test-org-id",
					HTTPClient: &http.Client{},
				},
			}

			err := c.RemoveConnectedAppScopes(context.Background(), tt.appID, tt.scopes)

			if tt.wantErr {
				if err == nil {
					t.Errorf("RemoveConnectedAppScopes() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("RemoveConnectedAppScopes() error = %v, want containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("RemoveConnectedAppScopes() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestConnectedAppScopesClient_DeleteConnectedAppScopes(t *testing.T) {
	tests := []struct {
		name           string
		appID          string
		getHandler     func(w http.ResponseWriter, r *http.Request)
		deleteHandler  func(w http.ResponseWriter, r *http.Request)
		wantErr        bool
		errContains    string
	}{
		{
			name:  "successful delete with existing scopes",
			appID: "test-app-id",
			getHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, &ConnectedAppScopes{
					Scopes: []Scope{
						{Scope: "read:scopes"},
					},
				})
			},
			deleteHandler: func(w http.ResponseWriter, r *http.Request) {
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:  "no-op when already empty",
			appID: "test-app-id",
			getHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.JSONResponse(w, http.StatusOK, &ConnectedAppScopes{
					Scopes: []Scope{},
				})
			},
			deleteHandler: nil,
			wantErr:       false,
		},
		{
			name:  "get fails propagates error",
			appID: "test-app-id",
			getHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "server error")
			},
			deleteHandler: nil,
			wantErr:       true,
			errContains:   "API request failed with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/accounts/api/connectedApplications/%s/scopes", tt.appID): func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodGet {
						tt.getHandler(w, r)
					} else {
						t.Errorf("Unexpected method %s on GET path", r.Method)
					}
				},
			}
			if tt.deleteHandler != nil {
				handlers[fmt.Sprintf("/accounts/api/organizations/test-org-id/connectedApplications/%s/scopes", tt.appID)] = func(w http.ResponseWriter, r *http.Request) {
					if r.Method == http.MethodDelete {
						tt.deleteHandler(w, r)
					} else {
						t.Errorf("Unexpected method %s on DELETE path", r.Method)
					}
				}
			}

			server := testutil.MockHTTPServer(t, handlers)

			c := &ConnectedAppScopesClient{
				UserAnypointClient: &client.UserAnypointClient{
					BaseURL:    server.URL,
					Token:      "mock-token",
					OrgID:      "test-org-id",
					HTTPClient: &http.Client{},
				},
			}

			err := c.DeleteConnectedAppScopes(context.Background(), tt.appID)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteConnectedAppScopes() expected error, got nil")
				} else if tt.errContains != "" && !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteConnectedAppScopes() error = %v, want containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteConnectedAppScopes() unexpected error = %v", err)
				}
			}
		})
	}
}
