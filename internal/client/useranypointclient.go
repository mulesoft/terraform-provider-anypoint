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
	Cache        *ResponseCache
}

// UserClientConfig represents the configuration for the UserAnypointClient.
// Token and OrgID are populated on the first NewUserAnypointClient call and
// reused by all subsequent calls within the same terraform apply.
// mu guards Token and OrgID against concurrent writes.
// Cache provides per-apply response caching for expensive lookup calls
// (ListAvailableRoles, ListOrgUsers, ListTeams) so that N resources sharing
// this config do not each fire redundant API calls for the same catalog data.
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
	Cache        *ResponseCache
}

// NewUserAnypointClient creates a new User-based Anypoint API client using password grant
func NewUserAnypointClient(config *UserClientConfig) (*UserAnypointClient, error) {
	if config.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}
	// Allow username/password to be empty if they can be filled from environment variables.
	// The provider-level Configure already folds in ANYPOINT_USERNAME / ANYPOINT_PASSWORD
	// before we get here, so config.Username / config.Password may be non-empty via either
	// the provider block or those env vars. As a final safety net we also honor
	// ANYPOINT_ADMIN_USERNAME / ANYPOINT_ADMIN_PASSWORD for call sites that bypass the
	// provider-level resolution.
	username := config.Username
	if username == "" {
		username = os.Getenv("ANYPOINT_ADMIN_USERNAME")
	}
	if username == "" {
		return nil, fmt.Errorf("username is required (set it on the provider configuration, or via the ANYPOINT_USERNAME or ANYPOINT_ADMIN_USERNAME environment variable)")
	}

	password := config.Password
	if password == "" {
		password = os.Getenv("ANYPOINT_ADMIN_PASSWORD")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required (set it on the provider configuration, or via the ANYPOINT_PASSWORD or ANYPOINT_ADMIN_PASSWORD environment variable)")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://anypoint.mulesoft.com"
	}
	timeout := 10 * time.Minute
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	// Ensure cache is initialized (nil-safe for tests that don't set it)
	cache := config.Cache
	if cache == nil {
		cache = NewResponseCache()
		config.Cache = cache
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
		Cache: cache,
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

// authenticate performs user authentication using password grant and stores the access token.
// It first tries the OAuth2 password grant endpoint. If the resulting token cannot access
// /accounts/api/me (production Anypoint returns opaque tokens that lack user context),
// it falls back to the /accounts/login endpoint which always returns user-context tokens.
func (c *UserAnypointClient) authenticate() error {
	// Try OAuth2 password grant first (works on devx and environments that issue JWTs)
	token, err := c.authenticateOAuth2()
	if err == nil {
		c.Token = token
		me, meErr := c.getMe()
		if meErr == nil {
			orgID, orgErr := c.extractOrgID(me)
			if orgErr == nil {
				c.OrgID = orgID
				return nil
			}
		}
		// OAuth2 token didn't work for /me — fall through to login endpoint
	}

	// Fallback: use /accounts/login (always returns tokens with user context)
	loginToken, loginErr := c.authenticateLogin()
	if loginErr != nil {
		if err != nil {
			return fmt.Errorf("OAuth2 password grant failed (%v) and login fallback also failed: %w", err, loginErr)
		}
		return fmt.Errorf("login authentication failed: %w", loginErr)
	}
	c.Token = loginToken

	me, err := c.getMe()
	if err != nil {
		return fmt.Errorf("failed to get user info after login: %w", err)
	}
	orgID, err := c.extractOrgID(me)
	if err != nil {
		return fmt.Errorf("failed to extract organization ID: %w", err)
	}
	c.OrgID = orgID

	return nil
}

// authenticateOAuth2 performs the OAuth2 password grant flow.
func (c *UserAnypointClient) authenticateOAuth2() (string, error) {
	authURL := fmt.Sprintf("%s/accounts/api/v2/oauth2/token", c.BaseURL)

	authData := map[string]string{
		"grant_type":    "password",
		"client_id":     c.ClientID,
		"client_secret": c.ClientSecret,
		"username":      c.Username,
		"password":      c.Password,
	}

	jsonData, err := json.Marshal(authData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal auth data: %w", err)
	}

	req, err := http.NewRequest("POST", authURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create auth request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send auth request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("user authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	var authResp map[string]interface{}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&authResp); decodeErr != nil {
		return "", fmt.Errorf("failed to decode auth response: %w", decodeErr)
	}

	if token, ok := authResp["access_token"].(string); ok {
		return token, nil
	}
	return "", fmt.Errorf("no access token found in response")
}

// authenticateLogin uses the /accounts/login endpoint as a fallback.
// This always returns tokens with full user context on all Anypoint environments.
func (c *UserAnypointClient) authenticateLogin() (string, error) {
	loginURL := fmt.Sprintf("%s/accounts/login", c.BaseURL)

	loginData := map[string]string{
		"username": c.Username,
		"password": c.Password,
	}

	jsonData, err := json.Marshal(loginData)
	if err != nil {
		return "", fmt.Errorf("failed to marshal login data: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return "", fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return "", fmt.Errorf("failed to send login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return "", fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp map[string]interface{}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&loginResp); decodeErr != nil {
		return "", fmt.Errorf("failed to decode login response: %w", decodeErr)
	}

	if token, ok := loginResp["access_token"].(string); ok {
		return token, nil
	}
	return "", fmt.Errorf("no access token found in login response")
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
