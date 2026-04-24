package client

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
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

// UserClientConfig represents the configuration for the UserAnypointClient
type UserClientConfig struct {
	BaseURL      string
	ClientID     string
	ClientSecret string
	Username     string
	Password     string
	Timeout      int
}

// NewUserAnypointClient creates a new User-based Anypoint API client using password grant
func NewUserAnypointClient(config *UserClientConfig) (*UserAnypointClient, error) {
	if config.ClientID == "" {
		return nil, fmt.Errorf("client_id is required")
	}
	if config.ClientSecret == "" {
		return nil, fmt.Errorf("client_secret is required")
	}
	// Allow username/password to be empty if they can be filled from environment variables
	username := config.Username
	if username == "" {
		username = os.Getenv("ANYPOINT_ADMIN_USERNAME")
	}
	if username == "" {
		return nil, fmt.Errorf("username is required (set via config or ANYPOINT_USERNAME environment variable)")
	}

	password := config.Password
	if password == "" {
		password = os.Getenv("ANYPOINT_ADMIN_PASSWORD")
	}
	if password == "" {
		return nil, fmt.Errorf("password is required (set via config or ANYPOINT_PASSWORD environment variable)")
	}

	baseURL := config.BaseURL
	if baseURL == "" {
		baseURL = "https://anypoint.mulesoft.com"
	}
	timeout := 10 * time.Minute
	if config.Timeout > 0 {
		timeout = time.Duration(config.Timeout) * time.Second
	}

	client := &UserAnypointClient{
		BaseURL:      baseURL,
		ClientID:     config.ClientID,
		ClientSecret: config.ClientSecret,
		Username:     username,
		Password:     password,
		HTTPClient: &http.Client{
			Timeout: timeout,
		},
	}

	// Authenticate and get token
	err := client.authenticate()
	if err != nil {
		return nil, fmt.Errorf("failed to authenticate user: %w", err)
	}

	return client, nil
}

// authenticate performs user authentication using password grant and stores the access token
func (c *UserAnypointClient) authenticate() error {
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
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("user authentication failed with status %d: %s", resp.StatusCode, string(body))
	}

	// Extract token from response
	var authResp map[string]interface{}
	if err := json.NewDecoder(resp.Body).Decode(&authResp); err != nil {
		return fmt.Errorf("failed to decode auth response: %w", err)
	}

	if token, ok := authResp["access_token"].(string); ok {
		c.Token = token
	} else {
		return fmt.Errorf("no access token found in response")
	}

	// Extract OrgID from token - use active organization if available
	if me, err := c.getMe(); err == nil {
		orgID, err := c.extractOrgID(me)
		if err != nil {
			return fmt.Errorf("failed to extract organization ID: %w", err)
		}
		c.OrgID = orgID
	} else {
		return fmt.Errorf("failed to get user info: %w", err)
	}

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
