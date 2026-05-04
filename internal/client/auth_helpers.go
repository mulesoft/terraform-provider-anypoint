package client

import (
	"encoding/json"
	"fmt"
	"net/http"
)

// ExtractOrgID extracts the organization ID from the /accounts/api/me response.
//
// Priority order:
//  1. Connected app's organization (client.org_id) — UI-agnostic
//  2. User's primary organization (user.organization.id)
//  3. Active organization from session (user.properties.cs_auth.activeOrganizationId) — UI-dependent
func ExtractOrgID(me map[string]interface{}) (string, error) {
	if client, ok := me["client"].(map[string]interface{}); ok {
		if orgID, ok := client["org_id"].(string); ok && orgID != "" {
			return orgID, nil
		}
	}

	if user, ok := me["user"].(map[string]interface{}); ok {
		if org, ok := user["organization"].(map[string]interface{}); ok {
			if id, ok := org["id"].(string); ok && id != "" {
				return id, nil
			}
		}
	}

	if user, ok := me["user"].(map[string]interface{}); ok {
		if properties, ok := user["properties"].(map[string]interface{}); ok {
			if csAuth, ok := properties["cs_auth"].(map[string]interface{}); ok {
				if activeOrgID, ok := csAuth["activeOrganizationId"].(string); ok && activeOrgID != "" {
					return activeOrgID, nil
				}
			}
		}
	}

	return "", fmt.Errorf("no organization ID found in user info")
}

// GetMe calls /accounts/api/me and returns the decoded JSON response.
func GetMe(httpClient *http.Client, baseURL, token string) (map[string]interface{}, error) {
	url := fmt.Sprintf("%s/accounts/api/me", baseURL)
	req, err := http.NewRequest("GET", url, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+token)

	resp, err := httpClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("failed to get user info with status %d", resp.StatusCode)
	}

	var me map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&me); err != nil {
		return nil, err
	}
	return me, nil
}
