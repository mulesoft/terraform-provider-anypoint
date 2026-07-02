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

# Create a custom role group for API Managers
resource "anypoint_role" "api_managers" {
  name        = "API Managers"
  description = "Custom role group for managing APIs"
}

# Create a custom role group for Data Analysts
resource "anypoint_role" "data_analysts" {
  name        = "Data Analysts"
  description = "Role group for data analytics team"
}

# Create a custom role group with explicit organization_id
resource "anypoint_role" "devops_team" {
  name            = "DevOps Engineers"
  description     = "Role group for DevOps team members"
  organization_id = var.org_id
}

output "api_managers_role_id" {
  description = "ID of the API Managers role group"
  value       = anypoint_role.api_managers.id
}

output "data_analysts_role_id" {
  description = "ID of the Data Analysts role group"
  value       = anypoint_role.data_analysts.id
}

output "devops_team_role_id" {
  description = "ID of the DevOps Engineers role group"
  value       = anypoint_role.devops_team.id
}
