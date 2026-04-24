# terraform {
#   required_providers {
#     anypoint = {
#       source = "sf.com/mulesoft/anypoint"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# Example: Fetch information about an existing team
data "anypoint_team" "existing_team" {
  id = "c63f78eb-39c8-4fb2-80df-09f885c480e0" # Replace with a valid team ID
}

# Output the team information
output "existing_team_name" {
  value = data.anypoint_team.existing_team.name
}

output "existing_team_created_at" {
  value = data.anypoint_team.existing_team.created_at
}

output "existing_team_updated_at" {
  value = data.anypoint_team.existing_team.updated_at
}

output "existing_team_parent_team_id" {
  value = data.anypoint_team.existing_team.parent_team_id
}