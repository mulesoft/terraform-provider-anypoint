# Example: Fetch information about an existing role group, including its
# permissions (role assignments) and members.
data "anypoint_role" "existing_role" {
  id = "<role_group_id>" # Replace with a valid role group ID
}

# Output the role group information
output "existing_role_name" {
  value = data.anypoint_role.existing_role.name
}

output "existing_role_description" {
  value = data.anypoint_role.existing_role.description
}

output "existing_role_editable" {
  value = data.anypoint_role.existing_role.editable
}

output "existing_role_external_names" {
  value = data.anypoint_role.existing_role.external_names
}

# Permissions (role assignments) granted by the role group. Excludes system and
# platform-injected side-effect grants.
output "existing_role_permissions" {
  value = data.anypoint_role.existing_role.permissions
}

# Usernames of the role group's members.
output "existing_role_members" {
  value = data.anypoint_role.existing_role.members
}

output "existing_role_created_at" {
  value = data.anypoint_role.existing_role.created_at
}
