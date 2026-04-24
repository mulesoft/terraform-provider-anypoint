# Example of managing role assignments for a role group

# terraform {
#   required_providers {
#     anypoint = {
#       source  = "sf.com/mulesoft/anypoint"
#       version = "0.1.0"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# # First create a role group
resource "anypoint_rolegroup" "example_custom_org" {
  organization_id = "080f1918-0096-4cac-85b5-b1cd9cdf9260"
  name        = "Example Role Group New 123"
  description = "Example role group for demonstrating role assignments"
}

# Then assign roles to the role group
resource "anypoint_rolegroup_roles" "example_custom_org" {
  organization_id = "080f1918-0096-4cac-85b5-b1cd9cdf9260"
  rolegroup_id = anypoint_rolegroup.example_custom_org.id
  
  roles = [
    {
      role_id = "d74ef94a-4292-4896-b860-b05bd7f90d6d"  # Example admin role
      context_params = {
        org = "080f1918-0096-4cac-85b5-b1cd9cdf9260"    # Organization context
      }
    },
    {
      role_id = "ceeabcd5-eb31-41c9-b387-01a0e9095620"  # Example admin role
      context_params = {
        org = "080f1918-0096-4cac-85b5-b1cd9cdf9260"    # Organization context
      }
    },    
  ]
}

# Output the role group details
output "rolegroup_details_custom_org" {
  description = "Details of the created role group"
  value = {
    id          = anypoint_rolegroup.example_custom_org.id
    name        = anypoint_rolegroup.example_custom_org.name
    description = anypoint_rolegroup.example_custom_org.description
  }
}

# Output the role assignments
output "role_assignments_custom_org" {
  description = "Role assignments for the role group"
  value = {
    rolegroup_id = anypoint_rolegroup_roles.example_custom_org.rolegroup_id
    roles        = anypoint_rolegroup_roles.example_custom_org.roles
  }
} 