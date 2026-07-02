terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  auth_type     = "user"
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  username      = var.anypoint_username
  password      = var.anypoint_password
  base_url      = var.anypoint_base_url
}

# Create a connected app that acts on its own behalf (client_credentials)
resource "anypoint_connected_app" "service_app" {
  client_name = "My Service Application"
  grant_types = ["client_credentials"]
  audience    = "internal"
  enabled     = true
}

# Create a connected app that acts on behalf of a user (authorization_code + password)
resource "anypoint_connected_app" "user_app" {
  client_name = "My User-Facing Application"
  grant_types = ["authorization_code", "password"]
  audience    = "internal"
  enabled     = true

  redirect_uris = [
    "https://myapp.example.com/callback",
    "https://myapp.example.com/auth/callback"
  ]

  client_uri = "https://myapp.example.com"
}

# Create a connected app with JWT Bearer grant (for service-to-service auth)
resource "anypoint_connected_app" "jwt_app" {
  client_name = "JWT Bearer Service"
  grant_types = ["urn:ietf:params:oauth:grant-type:jwt-bearer"]
  audience    = "internal"

  public_keys = [
    "-----BEGIN PUBLIC KEY-----\n<your_public_key_here>\n-----END PUBLIC KEY-----"
  ]
}

# Create a public app that can be used by anyone
resource "anypoint_connected_app" "public_app" {
  client_name = "Public API Client"
  grant_types = ["client_credentials"]
  audience    = "everyone"
  client_uri  = "https://public-docs.example.com"
}

# Output the client credentials (IMPORTANT: client_secret only shown at creation)
output "service_app_client_id" {
  description = "Client ID for the service application"
  value       = anypoint_connected_app.service_app.id
}

output "service_app_client_secret" {
  description = "Client secret for the service application (only available at creation)"
  value       = anypoint_connected_app.service_app.client_secret
  sensitive   = true
}

output "user_app_client_id" {
  description = "Client ID for the user-facing application"
  value       = anypoint_connected_app.user_app.id
}

output "user_app_redirect_uris" {
  description = "Configured redirect URIs"
  value       = anypoint_connected_app.user_app.redirect_uris
}
