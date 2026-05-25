package cloudhub2

import (
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func validCH2Config(t *testing.T) *client.Config {
	t.Helper()
	server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
	return &client.Config{
		BaseURL:      server.URL,
		ClientID:     "test-client-id",
		ClientSecret: "test-client-secret",
	}
}

func TestNewTLSContextClient_CloudHub2(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		c, err := NewTLSContextClient(validCH2Config(t))
		if err != nil {
			t.Fatalf("NewTLSContextClient() unexpected error: %v", err)
		}
		if c == nil {
			t.Error("NewTLSContextClient() returned nil")
		}
	})
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewTLSContextClient(&client.Config{ClientSecret: "s"})
		if err == nil || !strings.Contains(err.Error(), "client_id") {
			t.Errorf("expected client_id error, got %v", err)
		}
	})
}

// TestNewVPNConnectionClient_errors tests error cases not covered by vpnconnection_test.go.
func TestNewVPNConnectionClient_errors(t *testing.T) {
	t.Run("missing client_id", func(t *testing.T) {
		_, err := NewVPNConnectionClient(&client.Config{ClientSecret: "s"})
		if err == nil || !strings.Contains(err.Error(), "client_id") {
			t.Errorf("expected client_id error, got %v", err)
		}
	})
	t.Run("missing client_secret", func(t *testing.T) {
		_, err := NewVPNConnectionClient(&client.Config{ClientID: "id"})
		if err == nil || !strings.Contains(err.Error(), "client_secret") {
			t.Errorf("expected client_secret error, got %v", err)
		}
	})
}
