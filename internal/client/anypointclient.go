package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"time"
)

// AnypointClient represents the Anypoint API client
type AnypointClient struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	HTTPClient   *http.Client
	Token        string
	OrgID        string
}

// Config represents the configuration for the AnypointClient
type Config struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	Timeout      int
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

	client := &AnypointClient{
		BaseURL:      baseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}

	// Authenticate and get token
	err := client.authenticate()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate: %w", err)
	}

	return client, nil
}

// authenticate performs authentication and stores the access token
func (c *AnypointClient) authenticate() error {
	authURL := fmt.Sprintf("%s/accounts/api/v2/oauth2/token", c.BaseURL)

	authData := map[string]string{
		"client_id":     c.ClientID,
		"client_secret": c.ClientSecret,
		"grant_type":    "client_credentials",
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

	// Extract OrgID from token
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

func (c *AnypointClient) extractOrgID(me map[string]interface{}) (string, error) {
	return ExtractOrgID(me)
}

func (c *AnypointClient) getMe() (map[string]interface{}, error) {
	return GetMe(c.HTTPClient, c.BaseURL, c.Token)
}
