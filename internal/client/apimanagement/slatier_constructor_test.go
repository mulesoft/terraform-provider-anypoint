package apimanagement

import (
	"strings"
	"testing"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/testutil"
)

func TestNewSLATierClient(t *testing.T) {
	t.Run("valid config", func(t *testing.T) {
		server := testutil.MockHTTPServer(t, testutil.StandardMockHandlers())
		cfg := &client.Config{
			BaseURL:      server.URL,
			ClientID:     "test-client-id",
			ClientSecret: "test-client-secret",
		}
		c, err := NewSLATierClient(cfg)
		if err != nil {
			t.Fatalf("NewSLATierClient() unexpected error: %v", err)
		}
		if c == nil {
			t.Error("NewSLATierClient() returned nil")
		}
	})

	t.Run("missing client_id returns error", func(t *testing.T) {
		_, err := NewSLATierClient(&client.Config{ClientSecret: "secret"})
		if err == nil {
			t.Fatal("Expected error for missing client_id")
		}
		if !strings.Contains(err.Error(), "client_id") {
			t.Errorf("error = %v, want mention of client_id", err)
		}
	})

	t.Run("missing client_secret returns error", func(t *testing.T) {
		_, err := NewSLATierClient(&client.Config{ClientID: "id"})
		if err == nil {
			t.Fatal("Expected error for missing client_secret")
		}
		if !strings.Contains(err.Error(), "client_secret") {
			t.Errorf("error = %v, want mention of client_secret", err)
		}
	})

	t.Run("nil config returns error", func(t *testing.T) {
		_, err := NewSLATierClient(nil)
		if err == nil {
			t.Fatal("Expected error for nil config")
		}
	})
}
