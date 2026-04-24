# terraform {
#   required_providers {
#     anypoint = {
#       source = "sf.com/mulesoft/anypoint"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# # Example: Fetch information about an existing private space
# data "anypoint_private_space" "example" {
#   id = "your-private-space-id"
# }

# # Output the private space information
# output "private_space_name" {
#   value = data.anypoint_private_space.example.name
# }

# output "private_space_region" {
#   value = data.anypoint_private_space.example.region
# }

# output "private_space_status" {
#   value = data.anypoint_private_space.example.status
# }

# output "private_space_organization_id" {
#   value = data.anypoint_private_space.existing_private_space.organization_id
# }

# output "private_space_enable_iam_role" {
#   value = data.anypoint_private_space.existing_private_space.enable_iam_role
# }

# output "private_space_enable_egress" {
#   value = data.anypoint_private_space.existing_private_space.enable_egress
# }

# output "private_space_app_count" {
#   value = data.anypoint_private_space.existing_private_space.mule_app_deployment_count
# }

# output "private_space_vpc_migration" {
#   value = data.anypoint_private_space.existing_private_space.vpc_migration_in_progress
# }

# output "private_space_network" {
#   value = data.anypoint_private_space.existing_private_space.network
# }
