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

# A custom role group with inline permissions and members.
#
# `permissions` and `members` are managed directly on the role group — there are
# no separate resources for them. Each is an authoritative set: entries not
# listed here are removed on apply. Omit an attribute entirely to leave it
# unmanaged.
resource "anypoint_role" "api_managers" {
  name        = "API Managers"
  description = "Custom role group for managing APIs"

  # Permissions are referenced by their UI display name (case-insensitive); the
  # provider resolves each to a role ID at apply time. Use the
  # anypoint_available_permissions data source to discover valid names.
  permissions = [
    {
      # Organization-scoped permission.
      name = "Exchange Viewer"
      context_params = {
        org = var.org_id
      }
    },
    {
      # Environment-scoped permission (adds envId). Some permissions can only be
      # applied to Sandbox/Production environments.
      name = "Read Applications"
      context_params = {
        org   = var.org_id
        envId = var.env_id
      }
    },
  ]

  # Members are referenced by username (case-insensitive). Use the anypoint_users
  # data source to discover usernames.
  members = [
    "jdoe",
    "asmith",
  ]
}

# A role group that leaves permissions and members unmanaged (attributes omitted).
resource "anypoint_role" "data_analysts" {
  name        = "Data Analysts"
  description = "Role group for the data analytics team"
}

# A role group created in an explicit organization.
resource "anypoint_role" "devops_team" {
  name            = "DevOps Engineers"
  description     = "Role group for DevOps team members"
  organization_id = var.org_id
}

output "api_managers_role_id" {
  description = "ID of the API Managers role group"
  value       = anypoint_role.api_managers.id
}

output "api_managers_permissions" {
  description = "Permissions managed on the API Managers role group"
  value       = anypoint_role.api_managers.permissions
}

output "api_managers_members" {
  description = "Members managed on the API Managers role group"
  value       = anypoint_role.api_managers.members
}

output "data_analysts_role_id" {
  description = "ID of the Data Analysts role group"
  value       = anypoint_role.data_analysts.id
}

output "devops_team_role_id" {
  description = "ID of the DevOps Engineers role group"
  value       = anypoint_role.devops_team.id
}
