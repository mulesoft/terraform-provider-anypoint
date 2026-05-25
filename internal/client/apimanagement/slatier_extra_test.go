package apimanagement

import (
	"context"
	"encoding/json"
	"net/http"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestSLATierClient_GetSLATierByName(t *testing.T) {
	basePath := "/apimanager/api/v1/organizations/org-123/environments/env-456/apis/100/tiers"

	tiers := []SLATier{
		{ID: 1, Name: "Bronze", Description: "Bronze tier"},
		{ID: 2, Name: "Silver", Description: "Silver tier"},
		{ID: 3, Name: "Gold", Description: "Gold tier"},
	}

	tests := []struct {
		name        string
		tierName    string
		handler     func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		wantID      int
		isNotFound  bool
	}{
		{
			name:     "found existing tier",
			tierName: "Silver",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(struct {
					Tiers []SLATier `json:"tiers"`
				}{Tiers: tiers})
			},
			wantErr: false,
			wantID:  2,
		},
		{
			name:     "found first tier",
			tierName: "Bronze",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(struct {
					Tiers []SLATier `json:"tiers"`
				}{Tiers: tiers})
			},
			wantErr: false,
			wantID:  1,
		},
		{
			name:       "tier name not found",
			tierName:   "Platinum",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(struct {
					Tiers []SLATier `json:"tiers"`
				}{Tiers: tiers})
			},
			wantErr:    true,
			isNotFound: true,
		},
		{
			name:       "list error propagates",
			tierName:   "Gold",
			handler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "server error")
			},
			wantErr:    true,
			isNotFound: false,
		},
		{
			name:       "empty tier list",
			tierName:   "Bronze",
			handler: func(w http.ResponseWriter, r *http.Request) {
				w.Header().Set("Content-Type", "application/json")
				_ = json.NewEncoder(w).Encode(struct {
					Tiers []SLATier `json:"tiers"`
				}{Tiers: []SLATier{}})
			},
			wantErr:    true,
			isNotFound: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				basePath: tt.handler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &SLATierClient{
				AnypointClient: &client.AnypointClient{
					BaseURL:    server.URL,
					Token:      "test-token",
					HTTPClient: &http.Client{},
				},
			}

			tier, err := c.GetSLATierByName(context.Background(), "org-123", "env-456", 100, tt.tierName)

			if tt.wantErr {
				if err == nil {
					t.Fatal("Expected error, got nil")
				}
				if tt.isNotFound && !client.IsNotFound(err) {
					t.Errorf("Expected IsNotFound, got %v", err)
				}
			} else {
				if err != nil {
					t.Fatalf("Unexpected error: %v", err)
				}
				if tier == nil {
					t.Fatal("Expected non-nil tier")
				}
				if tier.ID != tt.wantID {
					t.Errorf("Tier ID = %d, want %d", tier.ID, tt.wantID)
				}
				if tier.Name != tt.tierName {
					t.Errorf("Tier Name = %s, want %s", tier.Name, tt.tierName)
				}
			}
		})
	}
}
