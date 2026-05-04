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

type CertificateClient struct {
	*client.AnypointClient
}

func NewCertificateClient(config *client.Config) (*CertificateClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &CertificateClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type CertificateResponse struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Meta           SecretGroupMeta `json:"meta"`
	ExpirationDate string          `json:"expirationDate,omitempty"`
	Details        json.RawMessage `json:"details,omitempty"`
	Algorithm      string          `json:"algorithm,omitempty"`
}

// --- Request Models ---

type CreateCertificateRequest struct {
	Name     string
	Type     string // PEM, JKS, PKCS12, JCEKS
	CertFile []byte
}

// --- CRUD Operations ---

func (c *CertificateClient) basePath(orgID, envID, sgID string) string {
	return fmt.Sprintf("%s/secrets-manager/api/v1/organizations/%s/environments/%s/secretGroups/%s/certificates",
		c.BaseURL, orgID, envID, sgID)
}

func (c *CertificateClient) CreateCertificate(ctx context.Context, orgID, envID, sgID string, request *CreateCertificateRequest) (*CertificateResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	body, contentType, err := buildCertificateMultipart(request)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusCreated && resp.StatusCode != http.StatusOK {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to create certificate with status %d: %s", resp.StatusCode, string(respBody))
	}

	var createResp CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return c.GetCertificate(ctx, orgID, envID, sgID, createResp.ID)
}

func (c *CertificateClient) GetCertificate(ctx context.Context, orgID, envID, sgID, certID string) (*CertificateResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), certID)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode == http.StatusNotFound {
		return nil, client.NewNotFoundError("certificate")
	}
	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get certificate with status %d: %s", resp.StatusCode, string(body))
	}

	var cert CertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&cert); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &cert, nil
}

func (c *CertificateClient) UpdateCertificate(ctx context.Context, orgID, envID, sgID, certID string, request *CreateCertificateRequest) (*CertificateResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), certID)

	body, contentType, err := buildCertificateMultipart(request)
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusCreated {
		respBody, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update certificate with status %d: %s", resp.StatusCode, string(respBody))
	}

	return c.GetCertificate(ctx, orgID, envID, sgID, certID)
}

func (c *CertificateClient) DeleteCertificate(ctx context.Context, orgID, envID, sgID, certID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), certID)

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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusAccepted {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete certificate with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListCertificates returns all certificates in the given secret group.
func (c *CertificateClient) ListCertificates(ctx context.Context, orgID, envID, sgID string) ([]CertificateResponse, error) {
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
	defer func() { _ = resp.Body.Close() }()

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list certificates with status %d: %s", resp.StatusCode, string(body))
	}

	var items []CertificateResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return items, nil
}

// --- Multipart Builder ---

func buildCertificateMultipart(request *CreateCertificateRequest) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("name", request.Name); err != nil {
		return nil, "", err
	}
	if err := writer.WriteField("type", request.Type); err != nil {
		return nil, "", err
	}

	if len(request.CertFile) > 0 {
		part, err := writer.CreateFormFile("certStore", "cert.pem")
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(request.CertFile); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}
	return &body, writer.FormDataContentType(), nil
}
