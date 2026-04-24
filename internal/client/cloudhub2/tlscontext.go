package cloudhub2

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
)

// TLSContextClient wraps the AnypointClient for TLS context operations
type TLSContextClient struct {
	*client.AnypointClient
}

// NewTLSContextClient creates a new TLSContextClient
func NewTLSContextClient(config *client.ClientConfig) (*TLSContextClient, error) {
	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		return nil, err
	}
	return &TLSContextClient{AnypointClient: anypointClient}, nil
}

// TLSContext represents a TLS context response
type TLSContext struct {
	ID         string          `json:"id"`
	Name       string          `json:"name"`
	TrustStore *TrustStoreInfo `json:"trustStore,omitempty"`
	KeyStore   *KeyStoreInfo   `json:"keyStore,omitempty"`
	Ciphers    CiphersConfig   `json:"ciphers"`
	Type       string          `json:"type"`
}

// TrustStoreInfo represents trust store information in the response
type TrustStoreInfo struct {
	FileName       string   `json:"fileName"`
	ExpirationDate string   `json:"expirationDate"`
	DNList         []DNInfo `json:"dnList"`
	Type           string   `json:"type"`
}

// DNInfo represents distinguished name information
type DNInfo struct {
	Issuer             CertificateSubject `json:"issuer"`
	Subject            CertificateSubject `json:"subject"`
	Version            string             `json:"version"`
	SerialNumber       string             `json:"serialNumber"`
	SignatureAlgorithm string             `json:"signatureAlgorithm"`
	PublicKeyAlgorithm string             `json:"publicKeyAlgorithm"`
	BasicConstraints   BasicConstraints   `json:"basicConstraints"`
	Validity           Validity           `json:"validity"`
	KeyUsage           []string           `json:"keyUsage"`
	CertificateType    string             `json:"certificateType"`
}

// CertificateSubject represents certificate subject or issuer information
type CertificateSubject struct {
	CommonName       string `json:"commonName"`
	CountryName      string `json:"countryName"`
	LocalityName     string `json:"localityName"`
	OrganizationName string `json:"organizationName"`
	OrganizationUnit string `json:"organizationUnit"`
	State            string `json:"state"`
}

// BasicConstraints represents basic constraints of a certificate
type BasicConstraints struct {
	CertificateAuthority bool `json:"certificateAuthority"`
}

// Validity represents certificate validity period
type Validity struct {
	NotBefore string `json:"notBefore"`
	NotAfter  string `json:"notAfter"`
}

// KeyStoreInfo represents key store information in the response
type KeyStoreInfo struct {
	FileName       string   `json:"fileName"`
	Type           string   `json:"type"`
	CN             string   `json:"cn"`
	SAN            []string `json:"san"`
	ExpirationDate string   `json:"expirationDate"`
}

// CiphersConfig represents cipher configuration
type CiphersConfig struct {
	AES128GcmSha256            bool `json:"aes128GcmSha256"`
	AES128Sha256               bool `json:"aes128Sha256"`
	AES256GcmSha384            bool `json:"aes256GcmSha384"`
	AES256Sha256               bool `json:"aes256Sha256"`
	DHERsaAES128Sha256         bool `json:"dheRsaAes128Sha256"`
	DHERsaAES256GcmSha384      bool `json:"dheRsaAes256GcmSha384"`
	DHERsaAES256Sha256         bool `json:"dheRsaAes256Sha256"`
	ECDHEECDSAAES128GcmSha256  bool `json:"ecdheEcdsaAes128GcmSha256"`
	ECDHEECDSAAES256GcmSha384  bool `json:"ecdheEcdsaAes256GcmSha384"`
	ECDHERsaAES128GcmSha256    bool `json:"ecdheRsaAes128GcmSha256"`
	ECDHERsaAES256GcmSha384    bool `json:"ecdheRsaAes256GcmSha384"`
	ECDHEECDSAChacha20Poly1305 bool `json:"ecdheEcdsaChacha20Poly1305"`
	ECDHERsaChacha20Poly1305   bool `json:"ecdheRsaChacha20Poly1305"`
	DHERsaChacha20Poly1305     bool `json:"dheRsaChacha20Poly1305"`
	TLSAES256GcmSha384         bool `json:"tlsAes256GcmSha384"`
	TLSChacha20Poly1305Sha256  bool `json:"tlsChacha20Poly1305Sha256"`
	TLSAES128GcmSha256         bool `json:"tlsAes128GcmSha256"`
}

// CreateTLSContextRequest represents the base request to create a TLS context
type CreateTLSContextRequest struct {
	Name      string           `json:"name"`
	TLSConfig TLSConfigRequest `json:"tlsConfig"`
	Ciphers   CiphersConfig    `json:"ciphers"`
}

// TLSConfigRequest represents TLS configuration in the request
type TLSConfigRequest struct {
	KeyStore KeyStoreRequest `json:"keyStore"`
}

// KeyStoreRequest represents key store configuration in the request
type KeyStoreRequest struct {
	Source string `json:"source"` // "PEM" or "JKS"
	// PEM-specific fields
	Certificate         *string `json:"certificate,omitempty"`
	Key                 *string `json:"key,omitempty"`
	KeyFileName         *string `json:"keyFileName,omitempty"`
	CertificateFileName *string `json:"certificateFileName,omitempty"`
	// JKS-specific fields
	KeystoreBase64   *string `json:"keystoreBase64,omitempty"`
	StorePassphrase  *string `json:"storePassphrase,omitempty"`
	Alias            *string `json:"alias,omitempty"`
	KeystoreFileName *string `json:"keystoreFileName,omitempty"`
	// Common fields
	KeyPassphrase *string `json:"keyPassphrase,omitempty"`
}

// UpdateTLSContextRequest represents the request to update a TLS context
type UpdateTLSContextRequest struct {
	Name      *string           `json:"name,omitempty"`
	TLSConfig *TLSConfigRequest `json:"tlsConfig,omitempty"`
	Ciphers   *CiphersConfig    `json:"ciphers,omitempty"`
}

// CreateTLSContext creates a new TLS context
// Note: The API returns 201 with no response body, so we need to indicate that a follow-up read is required
func (c *TLSContextClient) CreateTLSContext(ctx context.Context, orgID, privateSpaceID string, request *CreateTLSContextRequest) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts", c.BaseURL, orgID, privateSpaceID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return fmt.Errorf("failed to marshal TLS context request: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewBuffer(jsonData))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Authorization", "Bearer "+c.Token)

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to send request: %w", err)
	}
	defer resp.Body.Close()

	// API returns 201 with no body on successful creation
	if resp.StatusCode == http.StatusCreated {
		return nil // Success - creation completed
	}

	// Handle other status codes as errors
	body, _ := io.ReadAll(resp.Body)
	return fmt.Errorf("failed to create TLS context with status %d: %s", resp.StatusCode, string(body))
}

// ListTLSContexts retrieves all TLS contexts for a private space
func (c *TLSContextClient) ListTLSContexts(ctx context.Context, orgID, privateSpaceID string) ([]TLSContext, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts", c.BaseURL, orgID, privateSpaceID)

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

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to list TLS contexts with status %d: %s", resp.StatusCode, string(body))
	}

	var tlsContexts []TLSContext
	if err := json.NewDecoder(resp.Body).Decode(&tlsContexts); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return tlsContexts, nil
}

// GetTLSContext retrieves a TLS context by ID
func (c *TLSContextClient) GetTLSContext(ctx context.Context, orgID, privateSpaceID, tlsContextID string) (*TLSContext, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts/%s", c.BaseURL, orgID, privateSpaceID, tlsContextID)

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
		return nil, client.NewNotFoundError("TLS context")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to get TLS context with status %d: %s", resp.StatusCode, string(body))
	}

	var tlsContext TLSContext
	if err := json.NewDecoder(resp.Body).Decode(&tlsContext); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tlsContext, nil
}

// UpdateTLSContext updates an existing TLS context
func (c *TLSContextClient) UpdateTLSContext(ctx context.Context, orgID, privateSpaceID, tlsContextID string, request *UpdateTLSContextRequest) (*TLSContext, error) {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts/%s", c.BaseURL, orgID, privateSpaceID, tlsContextID)

	jsonData, err := json.Marshal(request)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal TLS context request: %w", err)
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
		return nil, client.NewNotFoundError("TLS context")
	}

	if resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("failed to update TLS context with status %d: %s", resp.StatusCode, string(body))
	}

	var tlsContext TLSContext
	if err := json.NewDecoder(resp.Body).Decode(&tlsContext); err != nil {
		return nil, fmt.Errorf("failed to decode response: %w", err)
	}

	return &tlsContext, nil
}

// DeleteTLSContext deletes a TLS context by ID
func (c *TLSContextClient) DeleteTLSContext(ctx context.Context, orgID, privateSpaceID, tlsContextID string) error {
	url := fmt.Sprintf("%s/runtimefabric/api/organizations/%s/privatespaces/%s/tlsContexts/%s", c.BaseURL, orgID, privateSpaceID, tlsContextID)

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

	if resp.StatusCode == http.StatusNotFound {
		return nil // Already deleted
	}

	if resp.StatusCode != http.StatusOK && resp.StatusCode != http.StatusNoContent {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("failed to delete TLS context with status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}
