package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"
)

// AnypointClient represents the Anypoint API client.
// It supports two authentication flows:
//   - client_credentials: just ClientID + ClientSecret (for "acts on its own behalf" apps)
//   - password: ClientID + ClientSecret + Username + Password (for "acts on behalf of user" apps)
//
// The flow is chosen automatically: if Username is set, password grant is used.
type AnypointClient struct {
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

// Config represents the configuration for the AnypointClient
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	Timeout      int
	// Token and OrgID are populated on the first NewAnypointClient call and
	// reused by all subsequent calls within the same terraform apply, so that
	// N resources sharing this Config do not each make a parallel auth request.
	// mu guards Token and OrgID against concurrent writes.
	mu    sync.Mutex
	Token string
	OrgID string
	// Cache provides per-apply response caching shared by all resources/data sources.
	Cache *ResponseCache
}

// ToUserClientConfig converts a Config into a UserClientConfig, propagating
// any cached token and the shared response cache so the user client can skip
// re-authentication and reuse cached catalog lookups.
func (c *Config) ToUserClientConfig() *UserClientConfig {
	c.mu.Lock()
	token, orgID := c.Token, c.OrgID
	c.mu.Unlock()
	return &UserClientConfig{
		BaseURL:      c.BaseURL,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		Username:     c.Username,
		Password:     c.Password,
		Timeout:      c.Timeout,
		Token:        token,
		OrgID:        orgID,
		Cache:        c.Cache,
	}
}

// NewAnypointClient creates a new Anypoint API client
func NewAnypointClient(config *Config) (*AnypointClient, error) {
	if config == nil {
		return nil, fmt.Errorf("config cannot be nil")
	}
	if config.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is required")
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
	}

	c := &AnypointClient{
		BaseURL:      baseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     config.Username,
		Password:     config.Password,
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
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	// Cache token so subsequent parallel NewAnypointClient calls skip auth.
	config.mu.Lock()
	if config.Token == "" {
		config.Token = c.Token
		config.OrgID = c.OrgID
	}
	config.mu.Unlock()

	return c, nil
}

// authenticate performs authentication and stores the access token.
// Uses password grant if Username is set, otherwise client_credentials.
//
// For user authentication the OAuth2 password grant is only half the story: several
// Anypoint control planes issue an *opaque* token for that grant which cannot read
// /accounts/api/me, so org resolution fails with 401 even though the grant itself
// succeeded. /accounts/login always returns a token carrying user context, so user auth
// falls back to it — the same fallback UserAnypointClient already performs. Without it
// every area built on this client is unusable with auth_type = "user", including the
// team and connected-app operations that require it.
//
// client_credentials has no such fallback and is left exactly as it was.
func (c *AnypointClient) authenticate() error {
	oauthErr := c.authenticateOAuth2()
	if oauthErr == nil {
		if err := c.resolveOrg(); err == nil {
			return nil
		}
		// Token issued but it cannot identify the user. Only user auth can recover.
		if c.Username == "" {
			return fmt.Errorf("failed to get user info: %w", c.resolveOrg())
		}
	} else if c.Username == "" {
		return oauthErr
	}

	if err := c.authenticateLogin(); err != nil {
		if oauthErr != nil {
			return fmt.Errorf("OAuth2 password grant failed (%v) and login fallback also failed: %w", oauthErr, err)
		}
		return err
	}
	return c.resolveOrg()
}

// resolveOrg reads the caller identity for the current token and stores its org.
func (c *AnypointClient) resolveOrg() error {
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

// authenticateLogin obtains a user-context token from /accounts/login.
func (c *AnypointClient) authenticateLogin() error {
	loginURL := fmt.Sprintf("%s/accounts/login", c.BaseURL)

	jsonData, err := json.Marshal(map[string]string{"username": c.Username, "password": c.Password})
	if err != nil {
		return fmt.Errorf("failed to marshal login data: %w", err)
	}

	req, err := http.NewRequest("POST", loginURL, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create login request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send login request: %w", err)
	}
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("login failed with status %d: %s", resp.StatusCode, string(body))
	}

	var loginResp map[string]interface{}
	if decodeErr := json.NewDecoder(resp.Body).Decode(&loginResp); decodeErr != nil {
		return fmt.Errorf("failed to decode login response: %w", decodeErr)
	}
	token, ok := loginResp["access_token"].(string)
	if !ok {
		return fmt.Errorf("no access token found in login response")
	}
	c.Token = token
	return nil
}

// authenticateOAuth2 runs the token endpoint and stores the resulting access token.
func (c *AnypointClient) authenticateOAuth2() error {
	authURL := fmt.Sprintf("%s/accounts/api/v2/oauth2/token", c.BaseURL)

	authData := map[string]string{
		"client_id":     c.ClientID,
		"client_secret": c.ClientSecret,
	}

	if c.Username != "" {
		// "Acts on behalf of user" connected app — password grant
		authData["grant_type"] = "password"
		authData["username"] = c.Username
		authData["password"] = c.Password
	} else {
		// "Acts on its own behalf" connected app — client_credentials grant
		authData["grant_type"] = "client_credentials"
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

	return nil
}

func (c *AnypointClient) extractOrgID(me map[string]interface{}) (string, error) {
	return ExtractOrgID(me)
}

func (c *AnypointClient) getMe() (map[string]interface{}, error) {
	return GetMe(c.HTTPClient, c.BaseURL, c.Token)
}
