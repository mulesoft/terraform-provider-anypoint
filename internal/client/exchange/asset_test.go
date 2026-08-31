package exchange

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strconv"
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

// TestAssetClient_DeleteType_Header is a regression guard for the delete-type
// header bug: the client previously sent "DeleteType: hard-delete", which
// MuleSoft Exchange ignores, silently downgrading every hard delete to a
// SOFT delete. Soft delete leaves a tombstone that blocks recreating the same
// groupId/assetId/version (HTTP 409 ASSET_PRE_CONDITIONS_FAILED).
//
// The platform header is "x-delete-type" with value "hard-delete" or
// "soft-delete". This test asserts the exact header name AND value for both
// version-level and asset-level deletes, and for both hard and soft modes.
func TestAssetClient_DeleteType_Header(t *testing.T) {
	cases := []struct {
		name       string
		hardDelete bool
		wantValue  string
	}{
		{name: "hard delete sends x-delete-type: hard-delete", hardDelete: true, wantValue: "hard-delete"},
		{name: "soft delete sends x-delete-type: soft-delete", hardDelete: false, wantValue: "soft-delete"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			// Version-level delete
			t.Run("DeleteAssetVersion", func(t *testing.T) {
				var gotValue string
				var sawWrongHeader bool
				handlers := map[string]func(w http.ResponseWriter, r *http.Request){
					"/exchange/api/v2/assets/g/a/1.0.0": func(w http.ResponseWriter, r *http.Request) {
						gotValue = r.Header.Get("x-delete-type")
						// The old, WRONG header name must not be present.
						if r.Header.Get("DeleteType") != "" {
							sawWrongHeader = true
						}
						w.WriteHeader(http.StatusNoContent)
					},
				}
				server := testutil.MockHTTPServer(t, handlers)
				client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

				if err := client.DeleteAssetVersion(context.Background(), "g", "a", "1.0.0", tc.hardDelete); err != nil {
					t.Fatalf("DeleteAssetVersion() unexpected error = %v", err)
				}
				if sawWrongHeader {
					t.Errorf("client sent legacy 'DeleteType' header; must use 'x-delete-type'")
				}
				if gotValue != tc.wantValue {
					t.Errorf("x-delete-type = %q, want %q", gotValue, tc.wantValue)
				}
			})

			// Asset-level delete (all versions)
			t.Run("DeleteAsset", func(t *testing.T) {
				var gotValue string
				var sawWrongHeader bool
				handlers := map[string]func(w http.ResponseWriter, r *http.Request){
					"/exchange/api/v2/assets/g/a": func(w http.ResponseWriter, r *http.Request) {
						gotValue = r.Header.Get("x-delete-type")
						if r.Header.Get("DeleteType") != "" {
							sawWrongHeader = true
						}
						w.WriteHeader(http.StatusNoContent)
					},
				}
				server := testutil.MockHTTPServer(t, handlers)
				client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

				if err := client.DeleteAsset(context.Background(), "g", "a", tc.hardDelete); err != nil {
					t.Fatalf("DeleteAsset() unexpected error = %v", err)
				}
				if sawWrongHeader {
					t.Errorf("client sent legacy 'DeleteType' header; must use 'x-delete-type'")
				}
				if gotValue != tc.wantValue {
					t.Errorf("x-delete-type = %q, want %q", gotValue, tc.wantValue)
				}
			})
		})
	}
}

// pagedAssetsHandler returns an http handler that simulates the LIVE-VERIFIED
// Exchange assets contract: GET /exchange/api/v2/assets paged by limit+offset,
// responding with a BARE JSON array (no envelope, no total header). It serves a
// synthetic corpus of `total` assets, honoring the requested offset/limit window,
// and records every (offset,limit) pair it was asked for so tests can assert the
// client's paging behavior. It also enforces the server's real max-limit cap (250):
// a request for limit>250 is answered 400, exactly like production.
func pagedAssetsHandler(total int, sawOffsets, sawLimits *[]int, sawSearch, sawTypes *[]string) func(w http.ResponseWriter, r *http.Request) {
	return func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		offset, _ := strconv.Atoi(q.Get("offset"))
		limit, _ := strconv.Atoi(q.Get("limit"))
		*sawOffsets = append(*sawOffsets, offset)
		*sawLimits = append(*sawLimits, limit)
		if sawSearch != nil {
			*sawSearch = append(*sawSearch, q.Get("search"))
		}
		if sawTypes != nil {
			*sawTypes = append(*sawTypes, q.Get("types"))
		}

		// Live cap: the real endpoint rejects limit>250 with HTTP 400.
		if limit > 250 {
			testutil.ErrorResponse(w, http.StatusBadRequest, "request/query/limit must be <= 250")
			return
		}

		page := make([]Asset, 0)
		for i := offset; i < offset+limit && i < total; i++ {
			page = append(page, Asset{
				GroupID: "g",
				AssetID: fmt.Sprintf("asset-%d", i),
				Version: "1.0.0",
				Name:    fmt.Sprintf("Asset %d", i),
				Type:    "custom",
			})
		}
		// Bare JSON array — NO envelope, NO total-count header (matches production).
		testutil.JSONResponse(w, http.StatusOK, page)
	}
}

// TestAssetClient_ListAllAssets_Paginates is the regression guard for the single-page
// truncation bug in the plural exchange_assets data source (same bug class as the
// ListTeamRoles / ListRoleAssignments / ListSelfManagedGateways pagination fixes).
//
// The Exchange assets endpoint pages via limit+offset and returns a bare JSON array
// with NO total-count, so the ONLY reliable end-of-list signal is a short page. A
// naive single-request implementation silently truncates any org with more matching
// assets than one page — and because the response has no total, the truncation is
// invisible. This corpus (600 assets) spans THREE pages at the max page size (250),
// so it proves the client walks 250 -> 250 -> 100(short=stop) and concatenates all.
func TestAssetClient_ListAllAssets_Paginates(t *testing.T) {
	const total = 600 // 250 + 250 + 100 -> three pages, last is short
	var sawOffsets, sawLimits []int
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/exchange/api/v2/assets": pagedAssetsHandler(total, &sawOffsets, &sawLimits, nil, nil),
	}
	server := testutil.MockHTTPServer(t, handlers)
	client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

	assets, err := client.ListAllAssets(context.Background(), &ListAssetsRequest{
		OrganizationID: "org-1",
		// Limit unset (0) => fetch ALL.
	})
	if err != nil {
		t.Fatalf("ListAllAssets() unexpected error = %v", err)
	}

	if len(assets) != total {
		t.Fatalf("len(assets) = %d, want %d (all pages concatenated)", len(assets), total)
	}
	// Spot-check that items from the LAST page are present (the ones a truncating impl drops).
	if assets[total-1].AssetID != fmt.Sprintf("asset-%d", total-1) {
		t.Errorf("last asset AssetID = %q, want %q", assets[total-1].AssetID, fmt.Sprintf("asset-%d", total-1))
	}
	// Exactly three requests: offsets 0, 250, 500 (stops on the short third page).
	wantOffsets := []int{0, 250, 500}
	if len(sawOffsets) != len(wantOffsets) {
		t.Fatalf("made %d requests (offsets %v), want %d", len(sawOffsets), sawOffsets, len(wantOffsets))
	}
	for i, want := range wantOffsets {
		if sawOffsets[i] != want {
			t.Errorf("request %d offset = %d, want %d", i, sawOffsets[i], want)
		}
	}
	// Every request must use the max supported page size (250) to minimize round-trips.
	for i, l := range sawLimits {
		if l != 250 {
			t.Errorf("request %d limit = %d, want 250 (max page size)", i, l)
		}
	}
}

// TestAssetClient_ListAllAssets_ExactMultipleStops verifies the empty-page boundary:
// when the corpus is an EXACT multiple of the page size, the final full page is NOT a
// short page, so the client must make one more request that returns an empty array and
// stop there — without that trailing empty page it could not know the list had ended.
func TestAssetClient_ListAllAssets_ExactMultipleStops(t *testing.T) {
	const total = 500 // 250 + 250 -> both full; needs a 3rd (empty) request to terminate
	var sawOffsets, sawLimits []int
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/exchange/api/v2/assets": pagedAssetsHandler(total, &sawOffsets, &sawLimits, nil, nil),
	}
	server := testutil.MockHTTPServer(t, handlers)
	client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

	assets, err := client.ListAllAssets(context.Background(), &ListAssetsRequest{OrganizationID: "org-1"})
	if err != nil {
		t.Fatalf("ListAllAssets() unexpected error = %v", err)
	}
	if len(assets) != total {
		t.Fatalf("len(assets) = %d, want %d", len(assets), total)
	}
	// offsets 0, 250, 500 — the third returns empty and terminates the loop.
	wantOffsets := []int{0, 250, 500}
	if len(sawOffsets) != len(wantOffsets) {
		t.Fatalf("made %d requests (offsets %v), want %d (incl. trailing empty page)", len(sawOffsets), sawOffsets, len(wantOffsets))
	}
	for i, want := range wantOffsets {
		if sawOffsets[i] != want {
			t.Errorf("request %d offset = %d, want %d", i, sawOffsets[i], want)
		}
	}
}

// TestAssetClient_ListAllAssets_RespectsCap verifies the OPTIONAL caller cap: when
// req.Limit is a positive value the client returns at most that many assets, and it
// must NOT over-fetch. A cap smaller than one page requests exactly the cap; a cap
// spanning pages walks pages until the cap is met, shrinking the final page request to
// the remaining budget (never fetching a whole extra page it would then discard).
func TestAssetClient_ListAllAssets_RespectsCap(t *testing.T) {
	const total = 600

	t.Run("cap smaller than one page", func(t *testing.T) {
		var sawOffsets, sawLimits []int
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/exchange/api/v2/assets": pagedAssetsHandler(total, &sawOffsets, &sawLimits, nil, nil),
		}
		server := testutil.MockHTTPServer(t, handlers)
		client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

		assets, err := client.ListAllAssets(context.Background(), &ListAssetsRequest{OrganizationID: "org-1", Limit: 10})
		if err != nil {
			t.Fatalf("ListAllAssets() unexpected error = %v", err)
		}
		if len(assets) != 10 {
			t.Fatalf("len(assets) = %d, want 10 (cap honored)", len(assets))
		}
		// A single request sized to the cap (10), not the max page size.
		if len(sawOffsets) != 1 {
			t.Fatalf("made %d requests, want 1 (cap fits in one page)", len(sawOffsets))
		}
		if sawLimits[0] != 10 {
			t.Errorf("request 0 limit = %d, want 10 (page shrunk to cap)", sawLimits[0])
		}
	})

	t.Run("cap spanning multiple pages", func(t *testing.T) {
		var sawOffsets, sawLimits []int
		handlers := map[string]func(w http.ResponseWriter, r *http.Request){
			"/exchange/api/v2/assets": pagedAssetsHandler(total, &sawOffsets, &sawLimits, nil, nil),
		}
		server := testutil.MockHTTPServer(t, handlers)
		client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

		// Cap = 300 -> page0 (250) + page1 (remaining 50). Never over-fetches.
		assets, err := client.ListAllAssets(context.Background(), &ListAssetsRequest{OrganizationID: "org-1", Limit: 300})
		if err != nil {
			t.Fatalf("ListAllAssets() unexpected error = %v", err)
		}
		if len(assets) != 300 {
			t.Fatalf("len(assets) = %d, want 300 (cap honored)", len(assets))
		}
		wantOffsets := []int{0, 250}
		if len(sawOffsets) != len(wantOffsets) {
			t.Fatalf("made %d requests (offsets %v), want %d", len(sawOffsets), sawOffsets, len(wantOffsets))
		}
		if sawLimits[0] != 250 {
			t.Errorf("request 0 limit = %d, want 250 (full page)", sawLimits[0])
		}
		if sawLimits[1] != 50 {
			t.Errorf("request 1 limit = %d, want 50 (remaining budget, no over-fetch)", sawLimits[1])
		}
	})
}

// TestAssetClient_ListAllAssets_EmptyCorpus verifies an org with zero matching assets
// makes exactly one request and returns an empty slice (no infinite loop on offset 0).
func TestAssetClient_ListAllAssets_EmptyCorpus(t *testing.T) {
	var sawOffsets, sawLimits []int
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/exchange/api/v2/assets": pagedAssetsHandler(0, &sawOffsets, &sawLimits, nil, nil),
	}
	server := testutil.MockHTTPServer(t, handlers)
	client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

	assets, err := client.ListAllAssets(context.Background(), &ListAssetsRequest{OrganizationID: "org-1"})
	if err != nil {
		t.Fatalf("ListAllAssets() unexpected error = %v", err)
	}
	if len(assets) != 0 {
		t.Errorf("len(assets) = %d, want 0", len(assets))
	}
	if len(sawOffsets) != 1 {
		t.Errorf("made %d requests, want exactly 1 (single empty page, no loop)", len(sawOffsets))
	}
}

// TestAssetClient_ListAllAssets_ForwardsFilters verifies the search and type filters are
// forwarded IDENTICALLY on every page request — a paginating client that dropped the
// filter on page 2+ would return unfiltered results mixed in.
func TestAssetClient_ListAllAssets_ForwardsFilters(t *testing.T) {
	const total = 600
	var sawOffsets, sawLimits []int
	var sawSearch, sawTypes []string
	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		"/exchange/api/v2/assets": pagedAssetsHandler(total, &sawOffsets, &sawLimits, &sawSearch, &sawTypes),
	}
	server := testutil.MockHTTPServer(t, handlers)
	client := &AssetClient{BaseURL: server.URL, Token: "mock-token", HTTPClient: &http.Client{}}

	_, err := client.ListAllAssets(context.Background(), &ListAssetsRequest{
		OrganizationID: "org-1",
		Search:         "payments",
		Type:           "rest-api",
	})
	if err != nil {
		t.Fatalf("ListAllAssets() unexpected error = %v", err)
	}
	if len(sawSearch) < 2 {
		t.Fatalf("expected multiple page requests, got %d", len(sawSearch))
	}
	for i := range sawSearch {
		if sawSearch[i] != "payments" {
			t.Errorf("request %d search = %q, want payments (filter must be forwarded on every page)", i, sawSearch[i])
		}
		if sawTypes[i] != "rest-api" {
			t.Errorf("request %d types = %q, want rest-api (filter must be forwarded on every page)", i, sawTypes[i])
		}
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

// TestUpdateAssetRequest_OmitsManager pins #65 at the wire layer: the metadata PATCH body
// must NEVER carry a `manager` key. The Exchange endpoint rejects any manager value (HTTP 403
// for a username, HTTP 400 for a uuid — LIVE-VERIFIED 2026-07-22), so `manager` was removed
// from UpdateAssetRequest entirely. Marshaling every OTHER mutable field and asserting the
// serialized keys contain no "manager" guards against it being reintroduced. (If the field is
// ever added back to the struct, this decode into a map still catches the resulting key.)
func TestUpdateAssetRequest_OmitsManager(t *testing.T) {
	req := &UpdateAssetRequest{
		Name:         testutil.StringPtr("Some Asset"),
		Description:  testutil.StringPtr("Some description"),
		ContactName:  testutil.StringPtr("Ada Lovelace"),
		ContactEmail: testutil.StringPtr("ada@example.com"),
	}

	data, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("Failed to marshal update request: %v", err)
	}

	var m map[string]json.RawMessage
	if err := json.Unmarshal(data, &m); err != nil {
		t.Fatalf("Failed to unmarshal update request into a map: %v", err)
	}
	if _, present := m["manager"]; present {
		t.Errorf("UpdateAssetRequest JSON must NOT contain a \"manager\" key (the PATCH endpoint 403/400s on it); got %s", string(data))
	}

	// Sanity: the mutable fields we DO support are present, so the test is actually
	// exercising a populated body (not passing vacuously on an all-omitempty struct).
	for _, want := range []string{"name", "description", "contactName", "contactEmail"} {
		if _, present := m[want]; !present {
			t.Errorf("expected mutable field %q in the PATCH body; got %s", want, string(data))
		}
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

// filePart is a decoded file part from a multipart body: the form field name
// (e.g. "files.schema.json") and the file's contents.
type filePart struct {
	fieldName string
	fileName  string
	content   string
}

// parseMultipartFileParts decodes body against contentType and returns every
// part that carries a filename (i.e. an actual file upload, not a plain field).
// Used to assert the multi-file publish contract without hitting the network.
func parseMultipartFileParts(t *testing.T, body *bytes.Buffer, contentType string) []filePart {
	t.Helper()

	_, params, err := mime.ParseMediaType(contentType)
	if err != nil {
		t.Fatalf("failed to parse media type %q: %v", contentType, err)
	}
	boundary, ok := params["boundary"]
	if !ok {
		t.Fatalf("multipart content type %q has no boundary", contentType)
	}

	reader := multipart.NewReader(bytes.NewReader(body.Bytes()), boundary)
	var parts []filePart
	for {
		p, err := reader.NextPart()
		if err == io.EOF {
			break
		}
		if err != nil {
			t.Fatalf("failed to read multipart part: %v", err)
		}
		if p.FileName() == "" {
			// A plain form field (name/type/status/etc.), not a file.
			continue
		}
		data, err := io.ReadAll(p)
		if err != nil {
			t.Fatalf("failed to read part %q: %v", p.FormName(), err)
		}
		parts = append(parts, filePart{
			fieldName: p.FormName(),
			fileName:  p.FileName(),
			content:   string(data),
		})
	}
	return parts
}

// writeTempFile writes content to a uniquely-named file under t.TempDir() and
// returns its absolute path. name controls the base filename (and therefore the
// extension the multipart field-name logic keys off of).
func writeTempFile(t *testing.T, name, content string) string {
	t.Helper()
	path := filepath.Join(t.TempDir(), name)
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatalf("failed to write temp file %s: %v", path, err)
	}
	return path
}

// TestBuildAssetMultipart_PrimaryOnly asserts the pre-existing single-file
// behaviour is unchanged: exactly one file part, correct field name, no
// ExtraFiles bleed-through.
func TestBuildAssetMultipart_PrimaryOnly(t *testing.T) {
	primary := writeTempFile(t, "orders-api.raml", "#%RAML 1.0\ntitle: Orders\n")

	req := &CreateAssetRequest{
		Name:       "RT RAML REST API",
		Type:       "rest-api",
		Status:     "published",
		FilePath:   primary,
		Classifier: "raml",
	}

	body, contentType, err := buildAssetMultipart(req)
	if err != nil {
		t.Fatalf("buildAssetMultipart() unexpected error = %v", err)
	}

	parts := parseMultipartFileParts(t, body, contentType)
	if len(parts) != 1 {
		t.Fatalf("expected exactly 1 file part, got %d: %+v", len(parts), parts)
	}
	if got, want := parts[0].fieldName, "files.raml.raml"; got != want {
		t.Errorf("field name = %q, want %q", got, want)
	}
	if got, want := parts[0].fileName, "orders-api.raml"; got != want {
		t.Errorf("file name = %q, want %q", got, want)
	}
}

// TestBuildAssetMultipart_PolicyTwoFiles is the core regression for task #103:
// a policy asset (schema.json + metadata.yaml) must produce TWO file parts in
// ONE multipart body, each with the correct files.{classifier}.{ext} field name
// and its own file contents. This is the exact shape the Exchange API requires
// (else MISSING_FILES_ERROR).
func TestBuildAssetMultipart_PolicyTwoFiles(t *testing.T) {
	schema := writeTempFile(t, "schema.json", `{"$schema":"http://json-schema.org/draft-07/schema#"}`)
	metadata := writeTempFile(t, "metadata.yaml", "#%Policy Definition 0.1\nname: E2E policy\n")

	req := &CreateAssetRequest{
		Name:       "E2E policy",
		Type:       "policy",
		Status:     "published",
		FilePath:   schema,
		Classifier: "schema",
		ExtraFiles: []AssetFileUpload{
			{FilePath: metadata, Classifier: "metadata"},
		},
	}

	body, contentType, err := buildAssetMultipart(req)
	if err != nil {
		t.Fatalf("buildAssetMultipart() unexpected error = %v", err)
	}

	parts := parseMultipartFileParts(t, body, contentType)
	if len(parts) != 2 {
		t.Fatalf("expected exactly 2 file parts, got %d: %+v", len(parts), parts)
	}

	// Index by field name so ordering isn't asserted (only presence + content).
	byField := make(map[string]filePart, len(parts))
	for _, p := range parts {
		byField[p.fieldName] = p
	}

	schemaPart, ok := byField["files.schema.json"]
	if !ok {
		t.Fatalf("missing files.schema.json part; got fields %v", fieldNames(parts))
	}
	if !strings.Contains(schemaPart.content, "json-schema.org") {
		t.Errorf("schema part content not preserved: %q", schemaPart.content)
	}
	if got, want := schemaPart.fileName, "schema.json"; got != want {
		t.Errorf("schema file name = %q, want %q", got, want)
	}

	metaPart, ok := byField["files.metadata.yaml"]
	if !ok {
		t.Fatalf("missing files.metadata.yaml part; got fields %v", fieldNames(parts))
	}
	if !strings.Contains(metaPart.content, "#%Policy Definition 0.1") {
		t.Errorf("metadata part content not preserved: %q", metaPart.content)
	}
	if got, want := metaPart.fileName, "metadata.yaml"; got != want {
		t.Errorf("metadata file name = %q, want %q", got, want)
	}
}

// TestBuildAssetMultipart_ExtraFilesSkipEmpty asserts an ExtraFiles entry with
// an empty FilePath is silently skipped (defensive: a partially-built upload
// list must not emit a bogus empty part or fail).
func TestBuildAssetMultipart_ExtraFilesSkipEmpty(t *testing.T) {
	primary := writeTempFile(t, "schema.json", "{}")
	real := writeTempFile(t, "metadata.yaml", "#%Policy Definition 0.1\n")

	req := &CreateAssetRequest{
		Name:       "E2E policy",
		Type:       "policy",
		FilePath:   primary,
		Classifier: "schema",
		ExtraFiles: []AssetFileUpload{
			{FilePath: "", Classifier: "ignored"}, // skipped
			{FilePath: real, Classifier: "metadata"},
		},
	}

	body, contentType, err := buildAssetMultipart(req)
	if err != nil {
		t.Fatalf("buildAssetMultipart() unexpected error = %v", err)
	}

	parts := parseMultipartFileParts(t, body, contentType)
	if len(parts) != 2 {
		t.Fatalf("expected 2 file parts (empty entry skipped), got %d: %v", len(parts), fieldNames(parts))
	}
	for _, p := range parts {
		if strings.Contains(p.fieldName, "ignored") {
			t.Errorf("empty-FilePath ExtraFiles entry should be skipped, but saw %q", p.fieldName)
		}
	}
}

// TestBuildAssetMultipart_MultiFileFileless asserts ExtraFiles can attach files
// even when the primary FilePath is empty — the loop is independent of the
// primary file's presence.
func TestBuildAssetMultipart_ExtraFilesWithoutPrimary(t *testing.T) {
	meta := writeTempFile(t, "metadata.yaml", "#%Policy Definition 0.1\n")

	req := &CreateAssetRequest{
		Name: "no-primary",
		Type: "policy",
		// FilePath intentionally empty
		ExtraFiles: []AssetFileUpload{
			{FilePath: meta, Classifier: "metadata"},
		},
	}

	body, contentType, err := buildAssetMultipart(req)
	if err != nil {
		t.Fatalf("buildAssetMultipart() unexpected error = %v", err)
	}

	parts := parseMultipartFileParts(t, body, contentType)
	if len(parts) != 1 {
		t.Fatalf("expected exactly 1 file part, got %d: %v", len(parts), fieldNames(parts))
	}
	if got, want := parts[0].fieldName, "files.metadata.yaml"; got != want {
		t.Errorf("field name = %q, want %q", got, want)
	}
}

// TestBuildAssetMultipart_ExtraFileReadError asserts a non-existent ExtraFiles
// path surfaces as an error (not a silent skip) — a missing second file must
// fail the publish rather than send an incomplete multipart body.
func TestBuildAssetMultipart_ExtraFileReadError(t *testing.T) {
	primary := writeTempFile(t, "schema.json", "{}")

	req := &CreateAssetRequest{
		Name:       "E2E policy",
		Type:       "policy",
		FilePath:   primary,
		Classifier: "schema",
		ExtraFiles: []AssetFileUpload{
			{FilePath: filepath.Join(t.TempDir(), "does-not-exist.yaml"), Classifier: "metadata"},
		},
	}

	_, _, err := buildAssetMultipart(req)
	if err == nil {
		t.Fatal("expected error for missing ExtraFiles path, got nil")
	}
	if !strings.Contains(err.Error(), "failed to read file") {
		t.Errorf("error = %v, want it to mention the file read failure", err)
	}
}

// fieldNames extracts the field names from parts for readable failure messages.
func fieldNames(parts []filePart) []string {
	names := make([]string, len(parts))
	for i, p := range parts {
		names[i] = p.fieldName
	}
	return names
}

// TestAssetClient_CreateAsset_PolicyMultiFile is the end-to-end wire test for
// task #103: it drives the full CreateAsset path (POST multipart → GET readback)
// and captures the ACTUAL request body the server received, asserting BOTH policy
// file parts (schema.json + metadata.yaml) crossed the network in one request.
// buildAssetMultipart unit tests prove the body is built correctly; this proves
// CreateAsset actually transmits it.
func TestAssetClient_CreateAsset_PolicyMultiFile(t *testing.T) {
	orgID := "test-org"
	groupID := "test-group"
	assetID := "rt-policy"
	version := "0.1.0"

	schema := writeTempFile(t, "schema.json", `{"$schema":"http://json-schema.org/draft-07/schema#"}`)
	metadata := writeTempFile(t, "metadata.yaml", "#%Policy Definition 0.1\nname: E2E policy\n")

	var capturedParts []filePart
	postPath := fmt.Sprintf("/exchange/api/v2/organizations/%s/assets/%s/%s/%s", orgID, groupID, assetID, version)
	getPath := fmt.Sprintf("/exchange/api/v2/assets/%s/%s/%s", groupID, assetID, version)

	handlers := map[string]func(w http.ResponseWriter, r *http.Request){
		postPath: func(w http.ResponseWriter, r *http.Request) {
			if r.Method != http.MethodPost {
				t.Errorf("expected POST, got %s", r.Method)
			}
			if got := r.Header.Get("x-sync-publication"); got != "true" {
				t.Errorf("x-sync-publication = %q, want \"true\"", got)
			}
			raw, err := io.ReadAll(r.Body)
			if err != nil {
				t.Fatalf("failed to read request body: %v", err)
			}
			capturedParts = parseMultipartFileParts(t, bytes.NewBuffer(raw), r.Header.Get("Content-Type"))
			w.WriteHeader(http.StatusCreated)
		},
		getPath: func(w http.ResponseWriter, r *http.Request) {
			testutil.JSONResponse(w, http.StatusOK, map[string]interface{}{
				"groupId": groupID,
				"assetId": assetID,
				"version": version,
				"name":    "E2E policy",
				"type":    "policy",
				"status":  "published",
			})
		},
	}
	server := testutil.MockHTTPServer(t, handlers)

	c := &AssetClient{
		BaseURL:    server.URL,
		Token:      "mock-token",
		HTTPClient: &http.Client{},
	}

	asset, err := c.CreateAsset(context.Background(), &CreateAssetRequest{
		OrganizationID: orgID,
		GroupID:        groupID,
		AssetID:        assetID,
		Version:        version,
		Name:           "E2E policy",
		Type:           "policy",
		Status:         "published",
		FilePath:       schema,
		Classifier:     "schema",
		ExtraFiles: []AssetFileUpload{
			{FilePath: metadata, Classifier: "metadata"},
		},
	})
	if err != nil {
		t.Fatalf("CreateAsset() unexpected error = %v", err)
	}
	if asset == nil || asset.Name != "E2E policy" {
		t.Fatalf("CreateAsset() returned unexpected asset: %+v", asset)
	}

	if len(capturedParts) != 2 {
		t.Fatalf("server received %d file parts, want 2: %v", len(capturedParts), fieldNames(capturedParts))
	}
	byField := make(map[string]filePart, len(capturedParts))
	for _, p := range capturedParts {
		byField[p.fieldName] = p
	}
	if _, ok := byField["files.schema.json"]; !ok {
		t.Errorf("POST body missing files.schema.json part; got %v", fieldNames(capturedParts))
	}
	if _, ok := byField["files.metadata.yaml"]; !ok {
		t.Errorf("POST body missing files.metadata.yaml part; got %v", fieldNames(capturedParts))
	}
}
