package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"sync"
	"time"
)

// UserAnypointClient represents the Anypoint API client using user credentials (password grant)
type UserAnypointClient struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	HTTPClient   *http.Client
	Token        string
	OrgID        string
}

// UserClientConfig represents the configuration for the UserAnypointClient.
// Token and OrgID are populated on the first NewUserAnypointClient call and
// reused by all subsequent calls within the same terraform apply.
// mu guards Token and OrgID against concurrent writes.
type UserClientConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	Timeout      int
	mu           sync.Mutex
	Token        string
	OrgID        string
}

// NewUserAnypointClient creates a new Anypoint API client for resources that
// historically only supported password grant (Environment, Organization).
//
// It supports two modes, selected by whether username/password resolve to a
// non-empty value:
//   - password grant (auth_type = "user"): client_id + client_secret + username + password
//   - client_credentials grant (auth_type = "connected_app", the default): client_id +
//     client_secret only, matching how every other resource in this provider authenticates.
//
// Previously this constructor unconditionally required username/password, which meant
// Environment/Organization resources errored under the documented default
// auth_type = "connected_app" ("username is required...") even though the provider-level
// schema only requires username/password when auth_type = "user". See
// https://github.com/mulesoft/terraform-provider-anypoint/issues (environment/organization
// resources fail under connected_app auth).
func NewUserAnypointClient(config *UserClientConfig) (*UserAnypointClient, error) {
	if config.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}
	// Username/password are only required for password grant. The provider-level
	// Configure folds in ANYPOINT_USERNAME / ANYPOINT_PASSWORD before we get here, so
	// config.Username / config.Password may be non-empty via either the provider block
	// or those env vars. As a final safety net we also honor ANYPOINT_ADMIN_USERNAME /
	// ANYPOINT_ADMIN_PASSWORD for call sites that bypass the provider-level resolution.
	// If neither resolves, we fall back to client_credentials grant instead of erroring —
	// this is the expected path for auth_type = "connected_app" (the documented default).
	username := config.Username
	if username == "" {
		username = os.Getenv("ANYPOINT_ADMIN_USERNAME")
	}

	password := config.Password
	if password == "" {
		password = os.Getenv("ANYPOINT_ADMIN_PASSWORD")
	}

	if (username == "") != (password == "") {
		return nil, fmt.Errorf("username and password must be set together for password grant auth " +
			"(set both on the provider configuration, or via ANYPOINT_USERNAME/ANYPOINT_PASSWORD or " +
			"ANYPOINT_ADMIN_USERNAME/ANYPOINT_ADMIN_PASSWORD) — or leave both unset to use client_credentials grant")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://anypoint.mulesoft.com"
	}
	timeout := 10 * time.Minute
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	c := &UserAnypointClient{
		BaseURL:      baseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     username,
		Password:     password,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}

	config.mu.Lock()
	if config.Token != "" {
		c.Token = config.Token
		c.OrgID = config.OrgID
		config.mu.Unlock()
		return c, nil
	}
	config.mu.Unlock()

	if err := c.authenticate(); err != nil {
		return nil, fmt.Errorf("failed to authenticate user: %w", err)
	}

	config.mu.Lock()
	if config.Token == "" {
		config.Token = c.Token
		config.OrgID = c.OrgID
	}
	config.mu.Unlock()

	return c, nil
}

// authenticate performs authentication and stores the access token. Uses password grant
// when Username/Password are set, otherwise falls back to client_credentials grant — see
// NewUserAnypointClient for why both are supported here.
func (c *UserAnypointClient) authenticate() error {
	authURL := fmt.Sprintf("%s/accounts/api/v2/oauth2/token", c.BaseURL)

	var authData map[string]string
	if c.Username != "" && c.Password != "" {
		authData = map[string]string{
			"grant_type":    "password",
			"client_id":     c.ClientID,
			"client_secret": c.ClientSecret,
			"username":      c.Username,
			"password":      c.Password,
		}
	} else {
		authData = map[string]string{
			"grant_type":    "client_credentials",
			"client_id":     c.ClientID,
			"client_secret": c.ClientSecret,
		}
	}

	jsonData, err := json.Marshal(authData)
	if err != nil {
		return fmt.Errorf("failed to marshal auth data: %w", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send auth request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Extract token from response
	var authResp map[string]interface{}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&authResp); decodeErr != nil {
		return fmt.Errorf("failed to decode auth response: %w", decodeErr)
	}

	if token, ok := authResp["access_token"].(string); ok {
		c.Token = token
	} else {
		return fmt.Errorf("no access token found in response")
	}

	// Extract OrgID from token - use active organization if available
	me, err := c.getMe()
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}
	orgID, err := c.extractOrgID(me)
	if err != nil {
		return fmt.Errorf("failed to extract organization ID: %w", err)
	}
	c.OrgID = orgID

	return nil
}

func (c *UserAnypointClient) extractOrgID(me map[string]interface{}) (string, error) {
	return ExtractOrgID(me)
}

func (c *UserAnypointClient) getMe() (map[string]interface{}, error) {
	return GetMe(c.HTTPClient, c.BaseURL, c.Token)
}

// SwitchOrganization allows switching to a different organization context
func (c *UserAnypointClient) SwitchOrganization(orgID string) error {
	// Verify the user has access to this organization
	me, err := c.getMe()
	if err != nil {
		return fmt.Errorf("failed to get user info: %w", err)
	}

	hasAccess := false
	if user, ok := me["user"].(map[string]interface{}); ok {
		if memberOrgs, ok := user["memberOfOrganizations"].([]interface{}); ok {
			for _, orgInterface := range memberOrgs {
				if org, ok := orgInterface.(map[string]interface{}); ok {
					if id, ok := org["id"].(string); ok && id == orgID {
						hasAccess = true
						break
					}
				}
			}
		}
	}

	if !hasAccess {
		return fmt.Errorf("user does not have access to organization %s", orgID)
	}

	c.OrgID = orgID
	return nil
}

// GetAccessibleOrganizations returns a list of organizations the user has access to
func (c *UserAnypointClient) GetAccessibleOrganizations() ([]map[string]string, error) {
	me, err := c.getMe()
	if err != nil {
		return nil, fmt.Errorf("failed to get user info: %w", err)
	}

	var organizations []map[string]string

	if user, ok := me["user"].(map[string]interface{}); ok {
		if memberOrgs, ok := user["memberOfOrganizations"].([]interface{}); ok {
			for _, orgInterface := range memberOrgs {
				if org, ok := orgInterface.(map[string]interface{}); ok {
					orgInfo := map[string]string{}
					if id, ok := org["id"].(string); ok {
						orgInfo["id"] = id
					}
					if name, ok := org["name"].(string); ok {
						orgInfo["name"] = name
					}
					if len(orgInfo) > 0 {
						organizations = append(organizations, orgInfo)
					}
				}
			}
		}
	}

	return organizations, nil
}
