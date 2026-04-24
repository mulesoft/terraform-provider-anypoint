# Example configuration for anypoint_team_members resource
# This demonstrates how to manage team membership

# First create or reference a team
resource "anypoint_team" "example_1" {
  team_name = "Example Team for Members 123"
  parent_team_id = var.parent_team_id
  team_type = "internal"
}

# Then manage team membership
resource "anypoint_team_members" "example" {
  team_id = anypoint_team.example_1.id

  members = [
    {
      id              = "f7f43384-b33e-470c-ad4c-285aa0c01212"  # Example user 1
      membership_type = "member"
    }
  ]
}

# Output the team membership configuration
output "team_membership_config" {
  value = {
    team_id      = anypoint_team_members.example.team_id
    member_count = length(anypoint_team_members.example.members)
    maintainers  = [
      for user in anypoint_team_members.example.users : user
      if user.membership_type == "maintainer"
    ]
  }
}

# Access full user details (computed)
output "team_user_details" {
  value = anypoint_team_members.example.users
} 