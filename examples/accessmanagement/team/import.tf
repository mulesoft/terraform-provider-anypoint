# Import an existing Anypoint team into Terraform state.
#
# Steps:
#   1. Replace the placeholder with your actual team ID (UUID).
#   2. Uncomment the import block and the resource block below.
#   3. Run: terraform init && terraform apply
#      OR use the CLI: terraform import anypoint_team.imported <team_id>
#
# Import ID format:
#   anypoint_team -> <team_id>
#   (The organization is resolved from the provider's configured org, or set
#   explicitly via the organization_id attribute on the resource.)

# locals {
#   team_id = "<team_id>"  # e.g. "c63f78eb-39c8-4fb2-80df-09f885c480e0"
# }

# import {
#   to = anypoint_team.imported
#   id = local.team_id
# }

# resource "anypoint_team" "imported" {
#   organization_id = var.org_id
#   name            = "<team_name>"
#   parent_team     = "<parent_team_name>"  # omit for root-level teams
#   team_type       = "internal"
# }

# output "imported_team_id" {
#   value = anypoint_team.imported.id
# }
