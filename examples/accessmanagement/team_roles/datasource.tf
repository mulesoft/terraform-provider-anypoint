# List all roles assigned to a team
data "anypoint_team_roles" "team_permissions" {
  team_id = var.team_id
}

# Output the team roles
output "assigned_roles" {
  value = data.anypoint_team_roles.team_permissions.roles
}

output "total_roles_assigned" {
  value = data.anypoint_team_roles.team_permissions.total
}

# Filter for organization-scoped roles (no envId)
output "org_scoped_roles" {
  value = [
    for role in data.anypoint_team_roles.team_permissions.roles :
    role if !contains(keys(role.context_params), "envId")
  ]
}

# Filter for environment-scoped roles
output "env_scoped_roles" {
  value = [
    for role in data.anypoint_team_roles.team_permissions.roles :
    role if contains(keys(role.context_params), "envId")
  ]
}
