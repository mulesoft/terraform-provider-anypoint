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

# Example: Fetch information about an existing user
data "anypoint_user" "existing_user" {
  provider = anypoint.admin
  id = "f7f43384-b33e-470c-ad4c-285aa0c01212"
}

# Output the user information
output "existing_user_username" {
  value = data.anypoint_user.existing_user.username
}

output "existing_user_first_name" {
  value = data.anypoint_user.existing_user.first_name
}

output "existing_user_last_name" {
  value = data.anypoint_user.existing_user.last_name
}

output "existing_user_email" {
  value = data.anypoint_user.existing_user.email
}

output "existing_user_phone_number" {
  value = data.anypoint_user.existing_user.phone_number
}

output "existing_user_mfa_excluded" {
  value = data.anypoint_user.existing_user.mfa_verification_excluded
}