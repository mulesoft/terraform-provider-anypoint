terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Create a QA team (top-level, under org root — parent_team omitted)
resource "anypoint_team" "qa" {
  name = "QA Team"
}

# Create a development team (top-level) with inline permissions + members.
resource "anypoint_team" "development" {
  name = "Development Team With Child teams"

  # Permissions are referenced by their UI display name (case-insensitive); the
  # provider resolves each to an id at apply time. Omit `permissions` entirely to
  # leave them unmanaged. System (internal) assignments are never modified.
  permissions = [
    {
      name = "Exchange Viewer"
      context_params = {
        org = var.org_id
      }
    },
  ]

  # Members are referenced by username (case-insensitive). membership_type is
  # optional and defaults to "member". Members assigned via external groups
  # (SAML/SCIM) are never modified.
  members = [
    {
      username        = "dev-lead"
      membership_type = "maintainer"
    },
    {
      username = "dev-engineer" # membership_type omitted → "member"
    },
  ]
}

# Read the development team back, including its permissions and members.
data "anypoint_team" "development" {
  id              = anypoint_team.development.id
  organization_id = var.org_id
}

# Create a development team with no child teams (top-level)
resource "anypoint_team" "development_without_child_teams" {
  name = "Development Team With No Child teams changed"
}

# Create a sub-team under development (parent resolved by ID)
resource "anypoint_team" "frontend" {
  name           = "Frontend Team"
  parent_team_id = anypoint_team.development.id
}

# Create a sub-team under development_without_child_teams
resource "anypoint_team" "backend" {
  name           = "Backend Team"
  parent_team_id = anypoint_team.development_without_child_teams.id
}

# Create an operations team (top-level)
resource "anypoint_team" "operations" {
  name = "Operations Team"
}

output "development_team_id" {
  description = "ID of the development team"
  value       = anypoint_team.development.id
}

output "development_without_child_teams_team_id" {
  description = "ID of the development team with no child teams"
  value       = anypoint_team.development_without_child_teams.id
}

output "qa_team_id" {
  description = "ID of the QA team"
  value       = anypoint_team.qa.id
}

output "frontend_team_id" {
  description = "ID of the frontend team"
  value       = anypoint_team.frontend.id
}

output "backend_team_id" {
  description = "ID of the backend team"
  value       = anypoint_team.backend.id
}

output "operations_team_id" {
  description = "ID of the operations team"
  value       = anypoint_team.operations.id
}

output "development_team_permissions" {
  description = "Roles read back from the development team by the data source"
  value       = data.anypoint_team.development.permissions
}

output "development_team_members" {
  description = "Members read back from the development team by the data source"
  value       = data.anypoint_team.development.members
}
