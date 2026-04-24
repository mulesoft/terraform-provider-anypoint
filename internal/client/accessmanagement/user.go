package accessmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// UserClient wraps the UserAnypointClient for user operations
type UserClient struct {
	*client.UserAnypointClient
}

// NewUserClient creates a new UserClient using user-based (password grant) authentication
func NewUserClient(config *client.UserClientConfig) (*UserClient, error) {
	userAnypointClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &UserClient{UserAnypointClient: userAnypointClient}, nil
}

// User represents an Anypoint User
type User struct {
	ID                      string           `json:"id,omitempty"`
	Username                string           `json:"username"`
	FirstName               string           `json:"firstName"`
	LastName                string           `json:"lastName"`
	Email                   string           `json:"email"`
	PhoneNumber             string           `json:"phoneNumber,omitempty"`
	Enabled                 bool             `json:"enabled"`
	CreatedAt               string           `json:"createdAt,omitempty"`
	UpdatedAt               string           `json:"updatedAt,omitempty"`
	Organization            UserOrganization `json:"organization"`
	MfaVerificationExcluded bool             `json:"mfaVerificationExcluded,omitempty"`
	// Password is not returned in responses for security reasons
}

// UserOrganization represents the organization details nested within a User
type UserOrganization struct {
	ID          string `json:"id"`
	IsFederated bool   `json:"isFederated"`
}

// CreateUserRequest represents the request to create a user
type CreateUserRequest struct {
	Username                string `json:"username"`
	FirstName               string `json:"firstName"`
	LastName                string `json:"lastName"`
	Email                   string `json:"email"`
	PhoneNumber             string `json:"phoneNumber,omitempty"`
	Password                string `json:"password"`
	MfaVerificationExcluded bool   `json:"mfaVerificationExcluded,omitempty"`
}

// UpdateUserRequest represents the request to update a user
type UpdateUserRequest struct {
	Username                *string `json:"username,omitempty"`
	FirstName               *string `json:"firstName,omitempty"`
	LastName                *string `json:"lastName,omitempty"`
	Email                   *string `json:"email,omitempty"`
	PhoneNumber             *string `json:"phoneNumber,omitempty"`
	MfaVerificationExcluded *bool   `json:"mfaVerificationExcluded,omitempty"`
	// Password updates would typically be handled separately
}

// DeleteUsersRequest represents the request to delete multiple users
type DeleteUsersRequest []string

// CreateUser creates a new user in Anypoint
func (c *UserClient) CreateUser(ctx context.Context, orgID string, user *CreateUserRequest) (*User, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/users", c.BaseURL, orgID)

	jsonData, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// Handle different response status codes
	switch resp.StatusCode {
	case http.StatusCreated:
		// Success case
	case http.StatusNotFound:
		return nil, client.NewNotFoundError("organization")
	case http.StatusConflict:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("user already exists: %s", string(body))
	case http.StatusTooManyRequests:
		return nil, fmt.Errorf("rate limit exceeded, please try again later")
	default:
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create user with status %d: %s", resp.StatusCode, string(body))
	}

	var createdUser User
	if err := json.NewDecoder(resp.Body).Decode(&createdUser); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &createdUser, nil
}

// GetUser retrieves a user by ID
func (c *UserClient) GetUser(ctx context.Context, orgID, userID string) (*User, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/users/%s", c.BaseURL, orgID, userID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("user")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get user with status %d: %s", resp.StatusCode, string(body))
	}

	var user User
	if err := json.NewDecoder(resp.Body).Decode(&user); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &user, nil
}

// UpdateUser updates an existing user
func (c *UserClient) UpdateUser(ctx context.Context, orgID, userID string, user *UpdateUserRequest) (*User, error) {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/users/%s", c.BaseURL, orgID, userID)

	jsonData, err := json.Marshal(user)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal user data: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("user")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update user with status %d: %s", resp.StatusCode, string(body))
	}

	var updatedUser User
	if err := json.NewDecoder(resp.Body).Decode(&updatedUser); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &updatedUser, nil
}

// DeleteUser deletes a user by ID
func (c *UserClient) DeleteUser(ctx context.Context, orgID, userID string) error {
	url := fmt.Sprintf("%s/accounts/api/organizations/%s/users/%s", c.BaseURL, orgID, userID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete user with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
