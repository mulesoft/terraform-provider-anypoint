# Example: Fetch information about an existing role group
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

output "existing_role_created_at" {
  value = data.anypoint_role.existing_role.created_at
}
