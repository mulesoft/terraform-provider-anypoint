package exchange

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

func TestNewAssetClient(t *testing.T) {
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

			client, err := NewAssetClient(tt.config)

			if tt.wantErr {
				if err == nil {
					t.Errorf("NewAssetClient() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("NewAssetClient() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("NewAssetClient() unexpected error = %v", err)
				}
				if client == nil {
					t.Errorf("NewAssetClient() returned nil client")
				}
			}
		})
	}
}

func TestAssetClient_GetAsset(t *testing.T) {
	mockAsset := &Asset{
		GroupID:      "test-group",
		AssetID:      "test-asset",
		Version:      "1.0.0",
		Name:         "Test Asset",
		Description:  "Test description",
		Type:         "custom",
		Status:       "published",
		IsPublic:     false,
		IsSnapshot:   false,
		MinorVersion: "1.0",
		VersionGroup: "1.0.0",
		CreatedDate:  "2024-01-01T00:00:00Z",
		UpdatedDate:  "2024-01-01T00:00:00Z",
	}

	tests := []struct {
		name          string
		groupID       string
		assetID       string
		version       string
		mockHandler   func(w http.ResponseWriter, r *http.Request)
		wantErr       bool
		errContains   string
		expectedAsset *Asset
	}{
		{
			name:    "successful get",
			groupID: "test-group",
			assetID: "test-asset",
			version: "1.0.0",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "GET", "/exchange/api/v2/assets/test-group/test-asset/1.0.0")
				testutil.JSONResponse(w, http.StatusOK, mockAsset)
			},
			wantErr:       false,
			expectedAsset: mockAsset,
		},
		{
			name:    "asset not found",
			groupID: "test-group",
			assetID: "nonexistent-asset",
			version: "1.0.0",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Asset not found")
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:    "server error",
			groupID: "test-group",
			assetID: "test-asset",
			version: "1.0.0",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to get asset with status 500",
		},
		{
			name:    "malformed response",
			groupID: "test-group",
			assetID: "test-asset",
			version: "1.0.0",
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
				fmt.Sprintf("/exchange/api/v2/assets/%s/%s/%s", tt.groupID, tt.assetID, tt.version): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &AssetClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
			}

			asset, err := client.GetAsset(context.Background(), tt.groupID, tt.assetID, tt.version)

			if tt.wantErr {
				if err == nil {
					t.Errorf("GetAsset() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("GetAsset() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("GetAsset() unexpected error = %v", err)
				}
				if asset == nil {
					t.Errorf("GetAsset() returned nil asset")
				}

				// Validate returned asset
				if asset != nil && tt.expectedAsset != nil {
					if asset.GroupID != tt.expectedAsset.GroupID {
						t.Errorf("GetAsset() GroupID = %v, want %v", asset.GroupID, tt.expectedAsset.GroupID)
					}
					if asset.AssetID != tt.expectedAsset.AssetID {
						t.Errorf("GetAsset() AssetID = %v, want %v", asset.AssetID, tt.expectedAsset.AssetID)
					}
					if asset.Version != tt.expectedAsset.Version {
						t.Errorf("GetAsset() Version = %v, want %v", asset.Version, tt.expectedAsset.Version)
					}
					if asset.Name != tt.expectedAsset.Name {
						t.Errorf("GetAsset() Name = %v, want %v", asset.Name, tt.expectedAsset.Name)
					}
				}
			}
		})
	}
}

func TestAssetClient_GetAsset_NotFound(t *testing.T) {
	groupID := "test-group"
	assetID := "nonexistent-asset"
	version := "1.0.0"
	path := fmt.Sprintf("/exchange/api/v2/assets/%s/%s/%s", groupID, assetID, version)

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		path: func(w http.ResponseWriter, r *http.Request) {
			testutil.ErrorResponse(w, http.StatusNotFound, "Asset not found")
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	client := &AssetClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
	}

	asset, err := client.GetAsset(context.Background(), groupID, assetID, version)

	if err == nil {
		t.Errorf("GetAsset() expected error, got nil")
	}

	if asset != nil {
		t.Errorf("GetAsset() expected nil asset on 404, got %v", asset)
	}

	// Verify the error is recognized as NotFound
	if !strings.Contains(err.Error(), "not found") && !strings.Contains(err.Error(), "404") {
		t.Errorf("GetAsset() error should indicate NotFound, got %v", err)
	}
}

func TestAssetClient_UpdateAsset(t *testing.T) {
	tests := []struct {
		name        string
		groupID     string
		assetID     string
		version     string
		request     *UpdateAssetRequest
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful update",
			groupID: "test-group",
			assetID: "test-asset",
			version: "1.0.0",
			request: &UpdateAssetRequest{
				Name:        testutil.StringPtr("Updated Asset"),
				Description: testutil.StringPtr("Updated description"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PATCH", "/exchange/api/v2/assets/test-group/test-asset")

				body := testutil.AssertJSONBody(t, r, "name", "description")
				if body["name"] != "Updated Asset" {
					t.Errorf("Expected name 'Updated Asset', got %v", body["name"])
				}

				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:    "asset not found",
			groupID: "test-group",
			assetID: "nonexistent-asset",
			version: "1.0.0",
			request: &UpdateAssetRequest{
				Name: testutil.StringPtr("Updated Asset"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Asset not found")
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:    "server error",
			groupID: "test-group",
			assetID: "test-asset",
			version: "1.0.0",
			request: &UpdateAssetRequest{
				Name: testutil.StringPtr("Updated Asset"),
			},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to update asset with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/exchange/api/v2/assets/%s/%s", tt.groupID, tt.assetID): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &AssetClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
			}

			err := client.UpdateAsset(context.Background(), tt.groupID, tt.assetID, tt.request)

			if tt.wantErr {
				if err == nil {
					t.Errorf("UpdateAsset() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("UpdateAsset() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("UpdateAsset() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAssetClient_DeleteAssetVersion(t *testing.T) {
	tests := []struct {
		name        string
		groupID     string
		assetID     string
		version     string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:    "successful deletion",
			groupID: "test-group",
			assetID: "test-asset",
			version: "1.0.0",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/exchange/api/v2/assets/test-group/test-asset/1.0.0")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:    "asset not found - returns nil",
			groupID: "test-group",
			assetID: "nonexistent-asset",
			version: "1.0.0",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "Asset not found")
			},
			wantErr: false, // 404 should return nil, not error
		},
		{
			name:    "server error",
			groupID: "test-group",
			assetID: "test-asset",
			version: "1.0.0",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusInternalServerError, "Internal server error")
			},
			wantErr:     true,
			errContains: "failed to delete asset version with status 500",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/exchange/api/v2/assets/%s/%s/%s", tt.groupID, tt.assetID, tt.version): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			client := &AssetClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
			}

			err := client.DeleteAssetVersion(context.Background(), tt.groupID, tt.assetID, tt.version, false)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteAssetVersion() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("DeleteAssetVersion() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("DeleteAssetVersion() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAsset_JSONSerialization(t *testing.T) {
	asset := &Asset{
		GroupID:      "test-group",
		AssetID:      "test-asset",
		Version:      "1.0.0",
		Name:         "Test Asset",
		Description:  "Test description",
		Type:         "custom",
		Status:       "published",
		IsPublic:     false,
		IsSnapshot:   false,
		MinorVersion: "1.0",
		VersionGroup: "1.0.0",
		CreatedDate:  "2024-01-01T00:00:00Z",
		UpdatedDate:  "2024-01-01T00:00:00Z",
	}

	// Test marshaling
	data, err := json.Marshal(asset)
	if err != nil {
		t.Fatalf("Failed to marshal asset: %v", err)
	}

	// Test unmarshaling
	var decoded Asset
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal asset: %v", err)
	}

	// Validate key fields
	if decoded.GroupID != asset.GroupID {
		t.Errorf("Unmarshaled GroupID = %v, want %v", decoded.GroupID, asset.GroupID)
	}
	if decoded.AssetID != asset.AssetID {
		t.Errorf("Unmarshaled AssetID = %v, want %v", decoded.AssetID, asset.AssetID)
	}
	if decoded.Version != asset.Version {
		t.Errorf("Unmarshaled Version = %v, want %v", decoded.Version, asset.Version)
	}
	if decoded.Name != asset.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", decoded.Name, asset.Name)
	}
}

func TestUpdateAssetRequest_JSONSerialization(t *testing.T) {
	req := &UpdateAssetRequest{
		Name:        testutil.StringPtr("Updated Asset"),
		Description: testutil.StringPtr("Updated description"),
	}

	// Test marshaling
	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal update request: %v", err)
	}

	// Test unmarshaling
	var decoded UpdateAssetRequest
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("Failed to unmarshal update request: %v", err)
	}

	// Validate fields
	if *decoded.Name != *req.Name {
		t.Errorf("Unmarshaled Name = %v, want %v", *decoded.Name, *req.Name)
	}
	if *decoded.Description != *req.Description {
		t.Errorf("Unmarshaled Description = %v, want %v", *decoded.Description, *req.Description)
	}
}

func TestAssetClient_SetCategory(t *testing.T) {
	tests := []struct {
		name        string
		categoryKey string
		values      []string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:        "successful set",
			categoryKey: "department",
			values:      []string{"Finance", "HR"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/categories/department")
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:        "not found",
			categoryKey: "nonexistent",
			values:      []string{"val"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:        "bad request - category not defined in org",
			categoryKey: "undefined-cat",
			values:      []string{"val"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Category 'undefined-cat' not found in organization")
			},
			wantErr:     true,
			errContains: "failed to set category",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/categories/%s", tt.categoryKey): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &AssetClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
			}

			err := c.SetCategory(context.Background(), "test-group", "test-asset", "1.0.0", tt.categoryKey, tt.values)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SetCategory() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("SetCategory() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("SetCategory() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAssetClient_DeleteCategory(t *testing.T) {
	tests := []struct {
		name        string
		categoryKey string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:        "successful delete",
			categoryKey: "department",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/categories/department")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:        "not found returns nil",
			categoryKey: "nonexistent",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
			wantErr: false, // 404 = already removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/categories/%s", tt.categoryKey): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &AssetClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
			}

			err := c.DeleteCategory(context.Background(), "test-group", "test-asset", "1.0.0", tt.categoryKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteCategory() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeleteCategory() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAssetClient_SetCustomField(t *testing.T) {
	tests := []struct {
		name        string
		fieldKey    string
		values      []string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
		errContains string
	}{
		{
			name:     "successful set",
			fieldKey: "subType",
			values:   []string{"Experience API"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "PUT", "/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/fields/subType")
				w.WriteHeader(http.StatusOK)
			},
			wantErr: false,
		},
		{
			name:     "not found",
			fieldKey: "nonexistent",
			values:   []string{"val"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
			wantErr:     true,
			errContains: "not found",
		},
		{
			name:     "bad request - field not defined in org",
			fieldKey: "undefined-field",
			values:   []string{"val"},
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusBadRequest, "Field 'undefined-field' not found in organization")
			},
			wantErr:     true,
			errContains: "failed to set custom field",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/fields/%s", tt.fieldKey): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &AssetClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
			}

			err := c.SetCustomField(context.Background(), "test-group", "test-asset", "1.0.0", tt.fieldKey, tt.values)

			if tt.wantErr {
				if err == nil {
					t.Errorf("SetCustomField() expected error, got nil")
				} else if !strings.Contains(err.Error(), tt.errContains) {
					t.Errorf("SetCustomField() error = %v, want error containing %v", err, tt.errContains)
				}
			} else {
				if err != nil {
					t.Errorf("SetCustomField() unexpected error = %v", err)
				}
			}
		})
	}
}

func TestAssetClient_DeleteCustomField(t *testing.T) {
	tests := []struct {
		name        string
		fieldKey    string
		mockHandler func(w http.ResponseWriter, r *http.Request)
		wantErr     bool
	}{
		{
			name:     "successful delete",
			fieldKey: "subType",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.AssertHTTPRequest(t, r, "DELETE", "/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/fields/subType")
				w.WriteHeader(http.StatusNoContent)
			},
			wantErr: false,
		},
		{
			name:     "not found returns nil",
			fieldKey: "nonexistent",
			mockHandler: func(w http.ResponseWriter, r *http.Request) {
				testutil.ErrorResponse(w, http.StatusNotFound, "not found")
			},
			wantErr: false, // 404 = already removed
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			handlers := map[string]func(w http.ResponseWriter, r *http.Request){
				fmt.Sprintf("/exchange/api/v2/assets/test-group/test-asset/1.0.0/tags/fields/%s", tt.fieldKey): tt.mockHandler,
			}
			server := testutil.MockHTTPServer(t, handlers)

			c := &AssetClient{
				BaseURL:    server.URL,
				Token:      "mock-token",
				HTTPClient: &http.Client{},
			}

			err := c.DeleteCustomField(context.Background(), "test-group", "test-asset", "1.0.0", tt.fieldKey)

			if tt.wantErr {
				if err == nil {
					t.Errorf("DeleteCustomField() expected error, got nil")
				}
			} else {
				if err != nil {
					t.Errorf("DeleteCustomField() unexpected error = %v", err)
				}
			}
		})
	}
}
