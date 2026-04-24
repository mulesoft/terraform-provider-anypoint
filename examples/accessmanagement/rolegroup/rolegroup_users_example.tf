# Example of managing user assignments for a role group

terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# First create a role group
resource "anypoint_rolegroup" "example" {
  organization_id = "080f1918-0096-4cac-85b5-b1cd9cdf9260"
  name        = "Example User Group Users test 123456"
  description = "Example role group for demonstrating user assignments"
}

# Then assign users to the role group
resource "anypoint_rolegroup_users" "example" {
  organization_id = "080f1918-0096-4cac-85b5-b1cd9cdf9260"
  rolegroup_id = anypoint_rolegroup.example.id
  
  user_ids = [
    "e0102052-4e55-4e61-985b-c284c97f3688",  # Example user 1
    
  ]
}

# Output the role group details
output "rolegroup_details" {
  description = "Details of the created role group"
  value = {
    id          = anypoint_rolegroup.example.id
    name        = anypoint_rolegroup.example.name
    description = anypoint_rolegroup.example.description
  }
}

# Output the user assignments
output "user_assignments" {
  description = "User assignments for the role group"
  value = {
    rolegroup_id = anypoint_rolegroup_users.example.rolegroup_id
    user_ids     = anypoint_rolegroup_users.example.user_ids
    users        = anypoint_rolegroup_users.example.users
  }
}

# Output user details (computed from API)
output "assigned_users" {
  description = "Details of users assigned to the role group"
  value = [
    for user in anypoint_rolegroup_users.example.users : {
      id       = user.id
      username = user.username
      email    = user.email
      enabled  = user.enabled
    }
  ]
} 