terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  auth_type     = "user"
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  username      = var.anypoint_username
  password      = var.anypoint_password
  base_url      = var.anypoint_base_url
}

# Assign a user to a role group
resource "anypoint_role_users" "user_assignment" {
  role_group_id = var.role_group_id
  user_id       = var.user_id
}

# Assign multiple users to the same role group
resource "anypoint_role_users" "user_assignment_2" {
  role_group_id = var.role_group_id
  user_id       = var.user_id_2
}

output "user_assignment_id" {
  description = "Assignment ID for the first user"
  value       = anypoint_role_users.user_assignment.id
}

output "user_username" {
  description = "Username of the assigned user"
  value       = anypoint_role_users.user_assignment.username
}

output "user_email" {
  description = "Email of the assigned user"
  value       = anypoint_role_users.user_assignment.email
}

output "user_full_name" {
  description = "Full name of the assigned user"
  value       = "${anypoint_role_users.user_assignment.first_name} ${anypoint_role_users.user_assignment.last_name}"
}
