# Import an existing Anypoint environment into Terraform state.
#
# Steps:
#   1. Replace the placeholder with your actual environment ID (UUID).
#   2. Uncomment the import block and the resource block below.
#   3. Run: terraform init && terraform apply
#      OR use the CLI: terraform import anypoint_environment.imported <env_id>
#
# Import ID format:
#   anypoint_environment -> <environment_id>
#   (The organization is resolved from the provider's configured org, or set
#   explicitly via the organization_id attribute on the resource.)

# locals {
#   env_id = "<env_id>"  # e.g. "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
# }

# import {
#   provider = anypoint.admin
#   to       = anypoint_environment.imported
#   id       = local.env_id
# }

# resource "anypoint_environment" "imported" {
#   provider = anypoint.admin
#
#   organization_id = var.organization_id
#   name            = "<environment_name>"
#   type            = "sandbox"      # one of: "design", "sandbox", "production"
#   is_production   = false
# }

# output "imported_env_id" {
#   value = anypoint_environment.imported.id
# }
