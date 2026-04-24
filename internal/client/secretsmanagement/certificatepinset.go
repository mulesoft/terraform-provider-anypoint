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

type CertificatePinsetClient struct {
	*client.AnypointClient
}

func NewCertificatePinsetClient(config *client.Config) (*CertificatePinsetClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &CertificatePinsetClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type CertificatePinsetResponse struct {
	Name           string          `json:"name"`
	Meta           SecretGroupMeta `json:"meta"`
	ExpirationDate string          `json:"expirationDate,omitempty"`
	Details        json.RawMessage `json:"details,omitempty"`
	Algorithm      string          `json:"algorithm,omitempty"`
}

// --- Request Models ---

type CreateCertificatePinsetRequest struct {
	Name    string
	PinFile []byte
}

// --- CRUD Operations ---

func (c *CertificatePinsetClient) basePath(orgID, envID, sgID string) string {
	return fmt.Sprintf("%s/secrets-manager/api/v1/organizations/%s/environments/%s/secretGroups/%s/certificatePinsets",
		c.BaseURL, orgID, envID, sgID)
}

func (c *CertificatePinsetClient) CreateCertificatePinset(ctx context.Context, orgID, envID, sgID string, request *CreateCertificatePinsetRequest) (*CertificatePinsetResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	body, contentType, err := buildPinsetMultipart(request)
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
		return nil, fmt.Errorf("failed to create certificate pinset with status %d: %s", resp.StatusCode, string(respBody))
	}

	var createResp CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return c.GetCertificatePinset(ctx, orgID, envID, sgID, createResp.ID)
}

func (c *CertificatePinsetClient) GetCertificatePinset(ctx context.Context, orgID, envID, sgID, pinID string) (*CertificatePinsetResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), pinID)

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
		return nil, client.NewNotFoundError("certificate pinset")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get certificate pinset with status %d: %s", resp.StatusCode, string(body))
	}

	var pin CertificatePinsetResponse
	if err := json.NewDecoder(resp.Body).Decode(&pin); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &pin, nil
}

func (c *CertificatePinsetClient) UpdateCertificatePinset(ctx context.Context, orgID, envID, sgID, pinID string, request *CreateCertificatePinsetRequest) (*CertificatePinsetResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), pinID)

	body, contentType, err := buildPinsetMultipart(request)
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
		return nil, fmt.Errorf("failed to update certificate pinset with status %d: %s", resp.StatusCode, string(respBody))
	}

	return c.GetCertificatePinset(ctx, orgID, envID, sgID, pinID)
}

func (c *CertificatePinsetClient) DeleteCertificatePinset(ctx context.Context, orgID, envID, sgID, pinID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), pinID)

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
		return fmt.Errorf("failed to delete certificate pinset with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListCertificatePinsets returns all certificate pinsets in the given secret group.
func (c *CertificatePinsetClient) ListCertificatePinsets(ctx context.Context, orgID, envID, sgID string) ([]CertificatePinsetResponse, error) {
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
		return nil, fmt.Errorf("failed to list certificate pinsets with status %d: %s", resp.StatusCode, string(body))
	}

	var items []CertificatePinsetResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return items, nil
}

// --- Multipart Builder ---

func buildPinsetMultipart(request *CreateCertificatePinsetRequest) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("name", request.Name); err != nil {
		return nil, "", err
	}

	if len(request.PinFile) > 0 {
		part, err := writer.CreateFormFile("certificatePinset", "cert.pem")
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(request.PinFile); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &body, writer.FormDataContentType(), nil
}
