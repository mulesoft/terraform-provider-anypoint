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

# Assign an organization-scoped role to a team
resource "anypoint_team_roles" "team_cloudhub_admin" {
  team_id = var.team_id
  role_id = "<cloudhub_admin_role_id>" # Use data.anypoint_available_roles to discover role IDs

  context_params = {
    org = var.org_id
  }
}

# Assign an environment-scoped role to a team
resource "anypoint_team_roles" "team_env_developer" {
  team_id = var.team_id
  role_id = "<cloudhub_developer_role_id>"

  context_params = {
    org   = var.org_id
    envId = var.environment_id
  }
}

# Assign Exchange Contributor role to a team
resource "anypoint_team_roles" "team_exchange_contributor" {
  team_id = var.team_id
  role_id = "<exchange_contributor_role_id>"

  context_params = {
    org = var.org_id
  }
}

output "team_cloudhub_admin_id" {
  description = "Assignment ID for CloudHub Admin role"
  value       = anypoint_team_roles.team_cloudhub_admin.id
}

output "team_env_developer_name" {
  description = "Name of the environment developer role"
  value       = anypoint_team_roles.team_env_developer.name
}

output "team_exchange_contributor_description" {
  description = "Description of the Exchange Contributor role"
  value       = anypoint_team_roles.team_exchange_contributor.description
}
