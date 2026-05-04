package acctest

import (
	"os"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/providerserver"
	"github.com/hashicorp/terraform-plugin-go/tfprotov6"

	"github.com/mulesoft/terraform-provider-anypoint/internal/client"
	"github.com/mulesoft/terraform-provider-anypoint/internal/provider"
)

var TestAccProtoV6ProviderFactories = map[string]func() (tfprotov6.ProviderServer, error){
	"anypoint": providerserver.NewProtocol6WithError(provider.New("test")()),
}

func TestAccPreCheck(t *testing.T) {
	t.Helper()

	requiredEnvVars := []string{
		"ANYPOINT_CLIENT_ID",
		"ANYPOINT_CLIENT_SECRET",
		"ANYPOINT_BASE_URL",
	}

	for _, envVar := range requiredEnvVars {
		if v := os.Getenv(envVar); v == "" {
			t.Fatalf("%s must be set for acceptance tests", envVar)
		}
	}
}

// CreateTestClient creates a client for testing purposes using environment variables.
func CreateTestClient(t *testing.T) *client.AnypointClient {
	if t != nil {
		t.Helper()
	}

	config := &client.Config{
		ClientID:     os.Getenv("ANYPOINT_CLIENT_ID"),
		ClientSecret: os.Getenv("ANYPOINT_CLIENT_SECRET"),
		BaseURL:      os.Getenv("ANYPOINT_BASE_URL"),
	}

	if config.ClientID == "" || config.ClientSecret == "" || config.BaseURL == "" {
		if t != nil {
			t.Skip("Skipping test due to missing environment variables")
		}
		return nil
	}

	anypointClient, err := client.NewAnypointClient(config)
	if err != nil {
		if t != nil {
			t.Fatalf("Failed to create test client: %v", err)
		}
		return nil
	}

	return anypointClient
}

// CreateUserTestClient creates a user-based client for testing purposes using environment variables.
func CreateUserTestClient(t *testing.T) *client.UserAnypointClient {
	t.Helper()

	config := &client.UserClientConfig{
		Username: os.Getenv("ANYPOINT_USERNAME"),
		Password: os.Getenv("ANYPOINT_PASSWORD"),
		BaseURL:  os.Getenv("ANYPOINT_BASE_URL"),
	}

	if config.Username == "" || config.Password == "" || config.BaseURL == "" {
		t.Skip("Skipping test due to missing environment variables")
	}

	userClient, err := client.NewUserAnypointClient(config)
	if err != nil {
		t.Fatalf("Failed to create user test client: %v", err)
	}

	return userClient
}
