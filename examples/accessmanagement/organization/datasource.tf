# terraform {
#   required_providers {
#     anypoint = {
#       source  = "mulesoft/anypoint"
#       version = "0.1.0"
#     }
#   }
# }

# # Note: Organization management requires user authentication
# # Set the following environment variables:
# # export TF_VAR_anypoint_admin_username="your-username"
# # export TF_VAR_anypoint_admin_password="your-password"

# provider "anypoint" {
#   alias = "admin"
#   client_id     = var.anypoint_admin_client_id
#   client_secret = var.anypoint_admin_client_secret
#   username      = var.anypoint_admin_username
#   password      = var.anypoint_admin_password
#   base_url      = var.anypoint_base_url
#   auth_type     = "user"
# }

# Example: Fetch information about an existing organization
data "anypoint_organization" "example" {
  provider = anypoint.admin
  id = "<org_id>"
}

# Output the organization information
output "existing_organization_name" {
  value = data.anypoint_organization.example.name
}

output "existing_organization_type" {
  value = data.anypoint_organization.example.org_type
}

output "existing_organization_client_id" {
  value = data.anypoint_organization.example.client_id
}

output "existing_organization_is_federated" {
  value = data.anypoint_organization.example.is_federated
}

output "existing_organization_created_at" {
  value = data.anypoint_organization.example.created_at
}

output "existing_organization_domain" {
  value = data.anypoint_organization.example.domain
}

output "existing_organization_parent_ids" {
  value = data.anypoint_organization.example.parent_organization_ids
}

output "existing_organization_sub_ids" {
  value = data.anypoint_organization.example.sub_organization_ids
}

output "existing_organization_environments" {
  value = data.anypoint_organization.example.environments
}

output "existing_organization_owner" {
  value = data.anypoint_organization.example.owner
}

output "existing_organization_session_timeout" {
  value = data.anypoint_organization.example.session_timeout
}

locals {
  entitlements = jsondecode(data.anypoint_organization.example.entitlements)
  flex_gateway_enabled = local.entitlements.flexGateway.enabled
}

output "flex_gateway_enabled" {
  value = local.flex_gateway_enabled
}