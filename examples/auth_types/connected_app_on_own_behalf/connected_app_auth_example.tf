terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

# Connected App authentication (default behavior)
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
  # auth_type defaults to "connected_app"
}

# Create private space using connected app authentication
# This will use the organization from the connected app's scope
resource "anypoint_private_space" "connected_app_space" {
  name   = "connected-app-space"
  region = var.region
  # organization_id will be auto-detected from connected app
}

# Create private space in a specific organization using connected app
resource "anypoint_private_space" "specific_org_space" {
  name            = "specific-org-space"
  region          = var.region
  organization_id = var.target_organization_id  # Explicit org ID
}