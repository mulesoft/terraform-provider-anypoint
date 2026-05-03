terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

# Configure the Anypoint Provider
# The connected_app_scopes resource will automatically use user authentication
# Credentials can be provided via provider config or environment variables
provider "anypoint" {
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  base_url      = var.anypoint_base_url
  # Optional: Explicitly set username/password, or use environment variables:
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  auth_type     = "user"
}

# Manage scopes for a connected application
resource "anypoint_connected_app_scopes" "example" {
  connected_app_id = var.connected_app_id

  scopes = [
    {
      scope = "admin:cloudhub"
      context_params = {
        org = var.target_organization_id
      }
    },
    {
      scope = "manage:runtime_fabrics"
      context_params = {
        org = var.target_organization_id
      }
    },  
    {
      scope = "create:environment"
      context_params = {
        org = var.target_organization_id
      }
    },  
  ]
}

# Output the configured scopes
output "configured_scopes" {
  description = "The scopes configured for the connected app"
  value       = anypoint_connected_app_scopes.example.scopes
}

output "connected_app_id" {
  description = "The ID of the connected app being managed"
  value       = anypoint_connected_app_scopes.example.connected_app_id
}