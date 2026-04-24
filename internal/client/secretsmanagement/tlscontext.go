package secretsmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

type TLSContextClient struct {
	*client.AnypointClient
}

func NewTLSContextClient(config *client.ClientConfig) (*TLSContextClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TLSContextClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type TLSContextPathRef struct {
	Path string `json:"path"`
}

type TLSInboundSettings struct {
	EnableClientCertValidation bool `json:"enableClientCertValidation"`
}

type TLSOutboundSettings struct {
	SkipServerCertValidation bool `json:"skipServerCertValidation"`
}

type TLSContext struct {
	Name             string               `json:"name"`
	Target           string               `json:"target"`
	Keystore         *TLSContextPathRef   `json:"keystore,omitempty"`
	Truststore       *TLSContextPathRef   `json:"truststore,omitempty"`
	MinTLSVersion    string               `json:"minTlsVersion,omitempty"`
	MaxTLSVersion    string               `json:"maxTlsVersion,omitempty"`
	AlpnProtocols    []string             `json:"alpnProtocols,omitempty"`
	InboundSettings  *TLSInboundSettings  `json:"inboundSettings,omitempty"`
	OutboundSettings *TLSOutboundSettings `json:"outboundSettings,omitempty"`
	CipherSuites     []string             `json:"cipherSuites,omitempty"`
}

type TLSContextResponse struct {
	Name             string               `json:"name"`
	Target           string               `json:"target"`
	Meta             SecretGroupMeta      `json:"meta"`
	Keystore         *TLSContextPathRef   `json:"keystore,omitempty"`
	Truststore       *TLSContextPathRef   `json:"truststore,omitempty"`
	MinTLSVersion    string               `json:"minTlsVersion,omitempty"`
	MaxTLSVersion    string               `json:"maxTlsVersion,omitempty"`
	AlpnProtocols    []string             `json:"alpnProtocols,omitempty"`
	InboundSettings  *TLSInboundSettings  `json:"inboundSettings,omitempty"`
	OutboundSettings *TLSOutboundSettings `json:"outboundSettings,omitempty"`
	CipherSuites     []string             `json:"cipherSuites,omitempty"`
	ExpirationDate   string               `json:"expirationDate,omitempty"`
}

// --- CRUD Operations ---

func (c *TLSContextClient) basePath(orgID, envID, sgID string) string {
	return fmt.Sprintf("%s/secrets-manager/api/v1/organizations/%s/environments/%s/secretGroups/%s/tlsContexts",
		c.BaseURL, orgID, envID, sgID)
}

func (c *TLSContextClient) CreateTLSContext(ctx context.Context, orgID, envID, sgID string, request *TLSContext) (*TLSContextResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TLS context request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create TLS context with status %d: %s", resp.StatusCode, string(body))
	}

	var createResp CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return c.GetTLSContext(ctx, orgID, envID, sgID, createResp.ID)
}

func (c *TLSContextClient) GetTLSContext(ctx context.Context, orgID, envID, sgID, tlsID string) (*TLSContextResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), tlsID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("TLS context")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get TLS context with status %d: %s", resp.StatusCode, string(body))
	}

	var tls TLSContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&tls); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &tls, nil
}

func (c *TLSContextClient) UpdateTLSContext(ctx context.Context, orgID, envID, sgID, tlsID string, request *TLSContext) (*TLSContextResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), tlsID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TLS context update request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update TLS context with status %d: %s", resp.StatusCode, string(body))
	}

	return c.GetTLSContext(ctx, orgID, envID, sgID, tlsID)
}

func (c *TLSContextClient) DeleteTLSContext(ctx context.Context, orgID, envID, sgID, tlsID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), tlsID)

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete TLS context with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListTLSContexts returns all TLS contexts in the given secret group.
func (c *TLSContextClient) ListTLSContexts(ctx context.Context, orgID, envID, sgID string) ([]TLSContextResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Accept", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list TLS contexts with status %d: %s", resp.StatusCode, string(body))
	}

	var items []TLSContextResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return items, nil
}
