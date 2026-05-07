terraform {
  required_providers {
    anypoint = {
      source  = "sfprod.com/mulesoft/anypoint"
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
#
# The `entitlements` block is optional. Every sub-attribute defaults to the
# zero value (false for booleans, 0 for quotas), so you only need to declare
# the entitlements you actually want to override. Omitting the block entirely
# is equivalent to `entitlements = {}` — the Anypoint API will assign defaults
# and the provider will surface them on refresh.
# resource "anypoint_organization" "example_org" {
#   provider = anypoint.admin

#   name                   = var.sub_org_name
#   parent_organization_id = var.parent_organization_id
#   owner_id               = var.owner_user_id

#   entitlements = {
#     create_environments = true
#     managed_gateway_small = {
#       assigned = 0
#     }
#     # Example: request 2 production vCores. All other quotas default to 0.
#     # vcores_production = {
#     #   assigned = 2
#     # }
#   }
# }

# Output the organization information
# output "organization_id" {
#   description = "The ID of the created organization"
#   value       = anypoint_organization.example_org.id
# }

# output "organization_name" {
#   description = "The name of the created organization"
#   value       = anypoint_organization.example_org.name
# }

# output "organization_client_id" {
#   description = "The client ID of the created organization"
#   value       = anypoint_organization.example_org.client_id
# }

# output "organization_created_at" {
#   description = "The creation timestamp of the organization"
#   value       = anypoint_organization.example_org.created_at
# }

# output "parent_organization_id" {
#   description = "The parent organization ID"
#   value       = anypoint_organization.example_org.parent_organization_id
# }

# output "owner_id" {
#   description = "The owner user ID"
#   value       = anypoint_organization.example_org.owner_id
# }