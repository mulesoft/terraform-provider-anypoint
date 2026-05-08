# terraform {
#   required_providers {
#     anypoint = {
#       source = "mulesoft/anypoint"
#       version = "0.1.0"
#     }
#   }
# }

# provider "anypoint" {
#   alias         = "admin"
#   client_id     = var.anypoint_admin_client_id
#   client_secret = var.anypoint_admin_client_secret
#   username      = var.anypoint_admin_username
#   password      = var.anypoint_admin_password
#   base_url      = var.anypoint_base_url
#   auth_type     = "user"
# }

# Example: Fetch information about an existing environment
data "anypoint_environment" "existing_environment" {
  provider = anypoint.admin
  id = "a4d171b4-9ad4-41da-9d77-18a3ade0a93d" # Replace with a valid environment ID
}

# Output the environment information
output "existing_environment_name" {
  value = data.anypoint_environment.existing_environment.name
}

output "existing_environment_type" {
  value = data.anypoint_environment.existing_environment.type
}

output "existing_environment_is_production" {
  value = data.anypoint_environment.existing_environment.is_production
}

output "existing_environment_client_id" {
  value = data.anypoint_environment.existing_environment.client_id
}

output "existing_environment_arc_namespace" {
  value = data.anypoint_environment.existing_environment.arc_namespace
} 