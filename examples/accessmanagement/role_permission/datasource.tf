# List all permissions assigned to a role group
data "anypoint_role_permissions" "role_group_permissions" {
  role_group_id = var.role_group_id
}

# Output the permissions
output "assigned_permissions" {
  value = data.anypoint_role_permissions.role_group_permissions.permissions
}

output "permission_count" {
  value = length(data.anypoint_role_permissions.role_group_permissions.permissions)
}

# Discover available roles (use this to find role IDs for assignments)
data "anypoint_available_roles" "all_roles" {
  organization_id = var.org_id
}

# Filter for CloudHub roles
output "cloudhub_roles" {
  value = [
    for role in data.anypoint_available_roles.all_roles.roles :
    role if strcontains(lower(role.name), "cloudhub")
  ]
}

# Filter for environment-scoped roles
output "environment_scoped_roles" {
  value = [
    for role in data.anypoint_available_roles.all_roles.roles :
    role if contains(keys(role.context_params), "envId")
  ]
}
