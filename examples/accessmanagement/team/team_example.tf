terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url = var.anypoint_base_url
}

# Create a QA team (top-level)
resource "anypoint_team" "qa" {
  team_name      = "QA Team"
  parent_team_id = var.parent_team_id
  team_type      = "internal"
}

# # Create a development team (top-level)
resource "anypoint_team" "development" {
  team_name      = "Development Team With Child teams"  
  parent_team_id = var.parent_team_id
  team_type      = "internal"
}

# Create a development team with no child teams(top-level)
resource "anypoint_team" "development_without_child_teams" {
  team_name      = "Development Team With No Child teams changed"
  parent_team_id = var.parent_team_id  
  team_type      = "internal"
}

# # Create a sub-team under development
resource "anypoint_team" "frontend" {
  team_name      = "Frontend Team"
  parent_team_id = anypoint_team.development.id
  team_type      = "internal"
}

# Create a sub-team under development
resource "anypoint_team" "backend" {
  team_name      = "Backend Team"
  parent_team_id = anypoint_team.development_without_child_teams.id
  team_type      = "internal"
}

# Create an operations team (top-level)
resource "anypoint_team" "operations" {
  team_name      = "Operations Team"
  parent_team_id = var.parent_team_id
  team_type      = "internal"
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