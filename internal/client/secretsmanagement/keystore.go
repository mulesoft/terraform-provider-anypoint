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

type KeystoreClient struct {
	*client.AnypointClient
}

func NewKeystoreClient(config *client.ClientConfig) (*KeystoreClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &KeystoreClient{AnypointClient: anypointClient}, nil
}

// --- Domain Models ---

type KeystoreResponse struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Meta           SecretGroupMeta `json:"meta"`
	ExpirationDate string          `json:"expirationDate,omitempty"`
	Details        json.RawMessage `json:"details,omitempty"`
	Algorithm      string          `json:"algorithm,omitempty"`
}

// CreateResponse is the simple response returned by POST/PUT operations.
type CreateResponse struct {
	Message string `json:"message"`
	ID      string `json:"id"`
}

// --- Request Models ---

type CreateKeystoreRequest struct {
	Name        string
	Type        string // PEM, JKS, PKCS12, JCEKS
	Certificate []byte // raw cert bytes (PEM text or binary)
	Key         []byte // raw key bytes (PEM text or binary, for PEM type)
	Keystore    []byte // raw keystore bytes (for JKS/PKCS12/JCEKS)
	Passphrase  string // keystore/key passphrase
	Alias       string // entry alias within keystore
	CaPath      []byte // optional CA certificate chain
}

// --- CRUD Operations ---

func (c *KeystoreClient) basePath(orgID, envID, sgID string) string {
	return fmt.Sprintf("%s/secrets-manager/api/v1/organizations/%s/environments/%s/secretGroups/%s/keystores",
		c.BaseURL, orgID, envID, sgID)
}

func (c *KeystoreClient) CreateKeystore(ctx context.Context, orgID, envID, sgID string, request *CreateKeystoreRequest) (*KeystoreResponse, error) {
	url := c.basePath(orgID, envID, sgID)

	body, contentType, err := buildMultipartBody(request)
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
		return nil, fmt.Errorf("failed to create keystore with status %d: %s", resp.StatusCode, string(respBody))
	}

	var createResp CreateResponse
	if err := json.NewDecoder(resp.Body).Decode(&createResp); err != nil {
		return nil, fmt.Errorf("failed to decode create response: %w", err)
	}

	return c.GetKeystore(ctx, orgID, envID, sgID, createResp.ID)
}

func (c *KeystoreClient) GetKeystore(ctx context.Context, orgID, envID, sgID, ksID string) (*KeystoreResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), ksID)

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
		return nil, client.NewNotFoundError("keystore")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get keystore with status %d: %s", resp.StatusCode, string(body))
	}

	var ks KeystoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&ks); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return &ks, nil
}

func (c *KeystoreClient) UpdateKeystore(ctx context.Context, orgID, envID, sgID, ksID string, request *CreateKeystoreRequest) (*KeystoreResponse, error) {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), ksID)

	body, contentType, err := buildMultipartBody(request)
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
		return nil, fmt.Errorf("failed to update keystore with status %d: %s", resp.StatusCode, string(respBody))
	}

	return c.GetKeystore(ctx, orgID, envID, sgID, ksID)
}

func (c *KeystoreClient) DeleteKeystore(ctx context.Context, orgID, envID, sgID, ksID string) error {
	url := fmt.Sprintf("%s/%s", c.basePath(orgID, envID, sgID), ksID)

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
		return fmt.Errorf("failed to delete keystore with status %d: %s", resp.StatusCode, string(body))
	}
	return nil
}

// ListKeystores returns all keystores in the given secret group.
func (c *KeystoreClient) ListKeystores(ctx context.Context, orgID, envID, sgID string) ([]KeystoreResponse, error) {
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
		return nil, fmt.Errorf("failed to list keystores with status %d: %s", resp.StatusCode, string(body))
	}

	var items []KeystoreResponse
	if err := json.NewDecoder(resp.Body).Decode(&items); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}
	return items, nil
}

// --- Multipart Builder ---

func buildMultipartBody(request *CreateKeystoreRequest) (*bytes.Buffer, string, error) {
	var body bytes.Buffer
	writer := multipart.NewWriter(&body)

	if err := writer.WriteField("name", request.Name); err != nil {
		return nil, "", err
	}
	if err := writer.WriteField("type", request.Type); err != nil {
		return nil, "", err
	}

	switch request.Type {
	case "PEM":
		if len(request.Certificate) > 0 {
			part, err := writer.CreateFormFile("certificate", "cert.pem")
			if err != nil {
				return nil, "", err
			}
			if _, err := part.Write(request.Certificate); err != nil {
				return nil, "", err
			}
		}
		if len(request.Key) > 0 {
			part, err := writer.CreateFormFile("key", "key.pem")
			if err != nil {
				return nil, "", err
			}
			if _, err := part.Write(request.Key); err != nil {
				return nil, "", err
			}
		}
		if request.Passphrase != "" {
			if err := writer.WriteField("keyPassphrase", request.Passphrase); err != nil {
				return nil, "", err
			}
		}

	case "JKS", "PKCS12", "JCEKS":
		if len(request.Keystore) > 0 {
			ext := "p12"
			if request.Type == "JKS" || request.Type == "JCEKS" {
				ext = "jks"
			}
			part, err := writer.CreateFormFile("keyStore", "keystore."+ext)
			if err != nil {
				return nil, "", err
			}
			if _, err := part.Write(request.Keystore); err != nil {
				return nil, "", err
			}
		}
		if request.Passphrase != "" {
			if err := writer.WriteField("keyPassphrase", request.Passphrase); err != nil {
				return nil, "", err
			}
		}
		if request.Alias != "" {
			if err := writer.WriteField("alias", request.Alias); err != nil {
				return nil, "", err
			}
		}
	}

	if len(request.CaPath) > 0 {
		part, err := writer.CreateFormFile("caPathCertificate", "ca.pem")
		if err != nil {
			return nil, "", err
		}
		if _, err := part.Write(request.CaPath); err != nil {
			return nil, "", err
		}
	}

	if err := writer.Close(); err != nil {
		return nil, "", err
	}

	return &body, writer.FormDataContentType(), nil
}
