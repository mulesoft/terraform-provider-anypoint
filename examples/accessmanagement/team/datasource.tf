# terraform {
#   required_providers {
#     anypoint = {
#       source = "mulesoft/anypoint"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# Example: Fetch information about an existing team
data "anypoint_team" "existing_team" {
  id = "c63f78eb-39c8-4fb2-80df-09f885c480e0" # Replace with a valid team ID
}

# Output the team information
output "existing_team_name" {
  value = data.anypoint_team.existing_team.name
}

output "existing_team_created_at" {
  value = data.anypoint_team.existing_team.created_at
}

output "existing_team_updated_at" {
  value = data.anypoint_team.existing_team.updated_at
}

output "existing_team_parent_team_id" {
  value = data.anypoint_team.existing_team.parent_team_id
}

# ---------------------------------------------------------------------------
# Example: list teams in the organization, optionally filtered by name.
#
# The list returns a summary per team — id, name, team_type, ancestry and
# timestamps. It does NOT include permissions or members; use the singular
# anypoint_team data source above for those.
# ---------------------------------------------------------------------------
data "anypoint_teams" "all" {
  organization_id = var.org_id
  name_filter     = "" # e.g. "Platform" to narrow the list
}

output "team_names" {
  description = "Names of every team matching the filter."
  value       = [for t in data.anypoint_teams.all.teams : t.name]
}

# ---------------------------------------------------------------------------
# Example: list users, to discover the usernames accepted by
# anypoint_team.members[].username.
# ---------------------------------------------------------------------------
data "anypoint_users" "all" {
  organization_id = var.org_id
}

output "usernames" {
  description = "Valid values for a members[].username entry."
  value       = [for u in data.anypoint_users.all.users : u.username]
}
