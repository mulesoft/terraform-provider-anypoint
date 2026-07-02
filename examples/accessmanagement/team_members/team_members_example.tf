terraform {
  required_providers {
    anypoint = {
      source  = "mulesoft/anypoint"
      version = "~> 1.0.0"
    }
  }
}

provider "anypoint" {
  auth_type     = "user"
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  username      = var.anypoint_username
  password      = var.anypoint_password
  base_url      = var.anypoint_base_url
}

# Add a user to a team as a regular member
resource "anypoint_team_members" "team_member" {
  team_id         = var.team_id
  user_id         = var.user_id
  membership_type = "member"
}

# Add a user as a team maintainer (can manage team membership)
resource "anypoint_team_members" "team_maintainer" {
  team_id         = var.team_id
  user_id         = var.maintainer_user_id
  membership_type = "maintainer"
}

# Add a user with default membership type (member)
resource "anypoint_team_members" "default_member" {
  team_id = var.team_id
  user_id = var.user_id_2
}

output "team_member_id" {
  description = "Composite ID for the team membership"
  value       = anypoint_team_members.team_member.id
}

output "team_maintainer_id" {
  description = "Composite ID for the team maintainer"
  value       = anypoint_team_members.team_maintainer.id
}

output "membership_type" {
  description = "Membership type of the first member"
  value       = anypoint_team_members.team_member.membership_type
}
