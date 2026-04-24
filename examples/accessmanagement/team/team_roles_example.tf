# # Example configuration for anypoint_team_roles resource
# # This demonstrates how to assign roles to a team

# # First create or reference a team
# resource "anypoint_team" "example" {
#   team_name = "Example Team for Roles"
#   parent_team_id = var.parent_team_id
#   team_type = "internal"
# }

# # Then assign roles to the team
# resource "anypoint_team_roles" "example" {
#   team_id = anypoint_team.example.id

#   roles = [
#         {
#       role_id = "98f87b9d-3e41-49cc-a171-f2580a742049"  # Example admin role
#       context_params = {
#         org = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"    # Organization context
#       }
#     },
#   ]
# }

# # Output the team roles configuration
# output "team_roles_config" {
#   value = {
#     team_id    = anypoint_team_roles.example.team_id
#     roles_count = length(anypoint_team_roles.example.roles)
#   }
# } 