# List all members of a team
data "anypoint_team_members" "team_roster" {
  team_id = var.team_id
}

# Output the team members
output "team_members" {
  value = data.anypoint_team_members.team_roster.members
}

output "total_team_members" {
  value = data.anypoint_team_members.team_roster.total
}

# Filter for maintainers only
output "team_maintainers" {
  value = [
    for member in data.anypoint_team_members.team_roster.members :
    member if member.membership_type == "maintainer"
  ]
}

# Filter for externally assigned members (via SAML/SCIM)
output "external_members" {
  value = [
    for member in data.anypoint_team_members.team_roster.members :
    member if member.is_assigned_via_external_groups
  ]
}
