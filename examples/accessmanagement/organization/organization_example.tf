terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

# Note: Organization management requires user authentication
# Set the following environment variables:
# export TF_VAR_anypoint_admin_username="your-username"
# export TF_VAR_anypoint_admin_password="your-password"

provider "anypoint" {
  alias = "admin"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  base_url      = var.anypoint_base_url
  auth_type     = "user"
}

# Create an organization
resource "anypoint_organization" "example_org" {
  provider = anypoint.admin

  name                   = var.sub_org_name
  parent_organization_id = var.parent_organization_id
  owner_id               = var.owner_user_id

  entitlements = {
    create_sub_orgs     = false
    create_environments = true
    global_deployment   = false

    vcores_production = {
      assigned = 0
    }

    vcores_sandbox = {
      assigned = 0
    }

    vcores_design = {
      assigned = 0
    }

    vpcs = {
      assigned = 0
    }

    network_connections = {
      assigned = 0
    }

    managed_gateway_small = {
      assigned = 0
    }

    managed_gateway_large = {
      assigned = 0
    }
  }

}

# Output the organization information
output "organization_id" {
  description = "The ID of the created organization"
  value       = anypoint_organization.example_org.id
}

output "organization_name" {
  description = "The name of the created organization"
  value       = anypoint_organization.example_org.name
}

output "organization_client_id" {
  description = "The client ID of the created organization"
  value       = anypoint_organization.example_org.client_id
}

output "organization_created_at" {
  description = "The creation timestamp of the organization"
  value       = anypoint_organization.example_org.created_at
}

output "parent_organization_id" {
  description = "The parent organization ID"
  value       = anypoint_organization.example_org.parent_organization_id
}

output "owner_id" {
  description = "The owner user ID"
  value       = anypoint_organization.example_org.owner_id
}