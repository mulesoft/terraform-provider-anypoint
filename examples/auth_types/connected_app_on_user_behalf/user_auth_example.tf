terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

# User authentication using password grant
provider "anypoint" {
  auth_type     = "user"
  client_id     = var.anypoint_admin_client_id     # Connected app client ID that supports password grant
  client_secret = var.anypoint_admin_client_secret # Connected app client secret
  username      = var.anypoint_admin_username      # User's username
  password      = var.anypoint_admin_password      # User's password
  base_url      = var.anypoint_base_url
}

# Create private space using user authentication
# This will use the user's active organization by default
resource "anypoint_private_space" "user_active_org_space" {
  name   = "user-active-org-space"
  region = var.region
  # organization_id will be auto-detected from user's active organization
}

# Create private space in a specific organization the user has access to
resource "anypoint_private_space" "user_specific_org_space" {
  name            = "user-specific-org-space"
  region          = var.region
  organization_id = var.target_organization_id  # Must be an org the user has access to
}

# Example of data source usage
# Note: You would typically use existing data sources like anypoint_organization
# once the user auth client functionality is fully implemented

# Example comment showing how multi-org access would work:
# When the user auth client is fully implemented with organization switching,
# you could create resources in different organizations by specifying organization_id