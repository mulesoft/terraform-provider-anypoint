package secretsmanagement

import (
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

// validConfig creates a client.Config that points at a mock server.
func validSecretsConfig(t *testing.T) *client.Config {
	t.Helper()
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	return &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
}

func missingIDConfig() *client.Config {
	return &client.Config{ClientSecret: "test-secret"}
}

func missingSecretConfig() *client.Config {
	return &client.Config{ClientID: "test-id"}
}

// --- NewCertificateClient ---

func TestNewCertificateClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewCertificateClient(validSecretsConfig(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil client")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewCertificateClient(missingIDConfig())
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "client_id") {
			t.Errorf("error = %v, want client_id mention", err)
		}
	})
	t.Run("missing client_secret", func(t *testing.T) {
		_, err := NewCertificateClient(missingSecretConfig())
		if err == nil {
			t.Fatal("expected error")
		}
		if !strings.Contains(err.Error(), "client_secret") {
			t.Errorf("error = %v, want client_secret mention", err)
		}
	})
}

// --- NewCertificatePinsetClient ---

func TestNewCertificatePinsetClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewCertificatePinsetClient(validSecretsConfig(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil client")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewCertificatePinsetClient(missingIDConfig())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- NewKeystoreClient ---

func TestNewKeystoreClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewKeystoreClient(validSecretsConfig(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil client")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewKeystoreClient(missingIDConfig())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- NewSecretGroupClient ---

func TestNewSecretGroupClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewSecretGroupClient(validSecretsConfig(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil client")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewSecretGroupClient(missingIDConfig())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- NewSharedSecretClient ---

func TestNewSharedSecretClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewSharedSecretClient(validSecretsConfig(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil client")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewSharedSecretClient(missingIDConfig())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- NewTLSContextClient (secretsmanagement) ---

func TestNewTLSContextClient_SecretsManagement(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewTLSContextClient(validSecretsConfig(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil client")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewTLSContextClient(missingIDConfig())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}

// --- NewTruststoreClient ---

func TestNewTruststoreClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewTruststoreClient(validSecretsConfig(t))
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		if c == nil {
			t.Error("expected non-nil client")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewTruststoreClient(missingIDConfig())
		if err == nil {
			t.Fatal("expected error")
		}
	})
}
