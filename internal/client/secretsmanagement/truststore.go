package secretsmanagement

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"mime/multipart"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

type TruststoreClient struct {
	*client.AnypointClient
}

func NewTruststoreClient(config *client.ClientConfig) (*TruststoreClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TruststoreClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type TruststoreResponse struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Meta           SecretGroupMeta `json:"meta"`
	ExpirationDate string          `json:"expirationDate,omitempty"`
	Details        json.RawMessage `json:"details,omitempty"`
	Algorithm      string          `json:"algorithm,omitempty"`
}

// --- Request Models ---

type CreateTruststoreRequest struct {
	Name       string
	Type       string // PEM, JKS, PKCS12, JCEKS
	TrustStore []byte // raw truststore bytes
	Passphrase string // for JKS/PKCS12/JCEKS
}

// --- CRUD Operations ---

func (c *TruststoreClient) basePath(orgID, envID, sgID string) string {
	return fmt.Sprintf("%s/secrets-manager/api/v1/organizations/%s/environments/%s/secretGroups/%s/truststores",
		c.BaseURL, orgID, envID, sgID)
}

func (c *TruststoreClient) CreateTruststore(ctx context.Context, orgID, envID, sgID string, request *CreateTruststoreRequest) (*TruststoreResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	body, contentType, err := buildTruststoreMultipart(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create truststore with status %d: %s", resp.StatusCode, string(respBody))
	}

	var createResp CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return c.GetTruststore(ctx, orgID, envID, sgID, createResp.ID)
}

func (c *TruststoreClient) GetTruststore(ctx context.Context, orgID, envID, sgID, tsID string) (*TruststoreResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), tsID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("truststore")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get truststore with status %d: %s", resp.StatusCode, string(body))
	}

	var ts TruststoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&ts); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &ts, nil
}

func (c *TruststoreClient) UpdateTruststore(ctx context.Context, orgID, envID, sgID, tsID string, request *CreateTruststoreRequest) (*TruststoreResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), tsID)

	body, contentType, err := buildTruststoreMultipart(request)
	if err != nil {
		return nil, fmt.Errorf("failed to build multipart body: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "PUT", url, body)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", contentType)
	req.Header.Set("Authorization", "Bearer "+c.Token)
	req.Header.Set("X-ANYPNT-ORG-ID", orgID)
	req.Header.Set("X-ANYPNT-ENV-ID", envID)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update truststore with status %d: %s", resp.StatusCode, string(respBody))
	}

	return c.GetTruststore(ctx, orgID, envID, sgID, tsID)
}

func (c *TruststoreClient) DeleteTruststore(ctx context.Context, orgID, envID, sgID, tsID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), tsID)

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
		return fmt.Errorf("failed to delete truststore with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListTruststores returns all truststores in the given secret group.
func (c *TruststoreClient) ListTruststores(ctx context.Context, orgID, envID, sgID string) ([]TruststoreResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	req, err := http.NewRequestWithContext(ctx, "GET", url, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to create request: %w", err)
	}

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
		return nil, fmt.Errorf("failed to list truststores with status %d: %s", resp.StatusCode, string(body))
	}

	var items []TruststoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return items, nil
}

// --- Multipart Builder ---

func buildTruststoreMultipart(request *CreateTruststoreRequest) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("name", request.Name); err != nil {
		return nil, "", err
	}
	if err := writer.WriteField("type", request.Type); err != nil {
		return nil, "", err
	}

	if len(request.TrustStore) > 0 {
		ext := "pem"
		if request.Type == "JKS" || request.Type == "JCEKS" {
			ext = "jks"
		} else if request.Type == "PKCS12" {
			ext = "p12"
		}
		part, err := writer.CreateFormFile("trustStore", "truststore."+ext)
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(request.TrustStore); err != nil {
			return nil, "", err
		}
	}

	if request.Passphrase != "" {
		if err := writer.WriteField("storePassphrase", request.Passphrase); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return &body, writer.FormDataContentType(), nil
}
