# List all users assigned to a role group
data "anypoint_role_users" "role_group_members" {
  role_group_id = var.role_group_id
}

# Output the users
output "assigned_users" {
  value = data.anypoint_role_users.role_group_members.users
}

output "total_users_in_role_group" {
  value = data.anypoint_role_users.role_group_members.total
}

# Look up users by username (use this to find user IDs for assignments)
data "anypoint_users" "search_users" {
  organization_id = var.org_id
  search_text     = var.search_username
}

# Output user search results
output "found_users" {
  value = data.anypoint_users.search_users.users
}

# Example: Get the first user ID from search results
output "first_user_id" {
  value = length(data.anypoint_users.search_users.users) > 0 ? data.anypoint_users.search_users.users[0].id : null
}
