terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  alias = "admin"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  base_url      = var.anypoint_base_url
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  auth_type     = "user"
}

# Create a user
resource "anypoint_user" "example_user_1" {
  provider = anypoint.admin

  username     = var.username
  first_name   = var.first_name
  last_name    = var.last_name
  email        = var.email
  phone_number = var.phone_number
  password     = var.password
  mfa_verification_excluded = var.mfa_verification_excluded
}

# Output the user information
output "user_id" {
  description = "The ID of the created user"
  value       = anypoint_user.example_user_1.id
}

output "user_username" {
  description = "The username of the created user"
  value       = anypoint_user.example_user_1.username
}

output "user_email" {
  description = "The email of the created user"
  value       = anypoint_user.example_user_1.email
}

output "user_full_name" {
  description = "The full name of the created user"
  value       = "${anypoint_user.example_user_1.first_name} ${anypoint_user.example_user_1.last_name}"
}

output "user_phone_number" {
  description = "The phone number of the created user"
  value       = "${anypoint_user.example_user_1.phone_number}"
}