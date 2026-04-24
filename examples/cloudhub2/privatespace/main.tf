terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# resource "anypoint_private_space" "vpc_1" {
#   name            = "example-private-space-2"
#   region          = "us-east-1"
# }

# Example 1: Private space using default organization (from provider credentials)
resource "anypoint_private_space" "private_space" {
  name            = var.private_space_name
  region          = var.private_space_region
}

# Example 2: Private space in a specific organization (only if custom_organization_id is provided)
resource "anypoint_private_space" "private_space_custom_org" {
  count           = var.custom_organization_id != "" ? 1 : 0
  name            = "${var.private_space_name}-custom-org"
  region          = var.private_space_region
  organization_id = var.custom_organization_id
}

# output "privatespace_id" {
#   value = anypoint_private_space.vpc_1.id
# }

# output "privatespace_status" {
#   value = anypoint_private_space.vpc_1.status
# }

# output "privatespace_organization_id" {
#   value = anypoint_private_space.vpc_1.organization_id
# }

# output "privatespace_enable_iam_role" {
#   value = anypoint_private_space.vpc_1.enable_iam_role
# }

# output "privatespace_enable_egress" {
#   value = anypoint_private_space.vpc_1.enable_egress
# }

# output "privatespace_mule_app_deployment_count" {
#   value = anypoint_private_space.vpc_1.mule_app_deployment_count
# }

# output "privatespace_id_2" {
#   value = anypoint_private_space.private_space.id
# }

# output "privatespace_status_2" {
#   value = anypoint_private_space.private_space.status
# }

# output "privatespace_organization_id_2" {
#   value = anypoint_private_space.private_space.organization_id
# }

# output "privatespace_enable_iam_role_2" {
#   value = anypoint_private_space.private_space.enable_iam_role
# }

# output "privatespace_enable_egress_2" {
#   value = anypoint_private_space.private_space.enable_egress
# }

# output "privatespace_mule_app_deployment_count_2" {
#   value = anypoint_private_space.private_space.mule_app_deployment_count
# } 
