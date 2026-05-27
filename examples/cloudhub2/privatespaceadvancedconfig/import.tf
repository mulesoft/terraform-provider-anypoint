# Import an existing private space advanced configuration into Terraform.
#
# Steps:
#   1. Replace the placeholders with your actual IDs.
#   2. Uncomment the import and resource blocks.
#   3. Run: terraform init && terraform apply
#      OR use CLI: terraform import anypoint_privatespace_advanced_config.imported <id>
#
# Import ID format:
#   Root org:  anypoint_privatespace_advanced_config -> <private_space_id>
#   Sub-org:   anypoint_privatespace_advanced_config -> <org_id>/<private_space_id>
#
# Use the <org_id>/<private_space_id> format when the private space was created
# in a Business Group (sub-org) rather than the root organization.

# --- Root org (simple ID) ---
# locals {
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
# }
#
# import {
#   to = anypoint_privatespace_advanced_config.imported
#   id = local.private_space_id
# }

# --- Sub-org (composite ID) ---
# locals {
#   org_id           = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
# }
#
# import {
#   to = anypoint_privatespace_advanced_config.imported
#   id = "${local.org_id}/${local.private_space_id}"
# }

# resource "anypoint_privatespace_advanced_config" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = "<organization_id>"
#
#   ingress_configuration = {
#     read_response_timeout = "600"
#     logs = {
#       port_log_level = "INFO"
#       filters        = []
#     }
#   }
#
#   enable_iam_role = true
# }

# output "imported_id" {
#   value = anypoint_privatespace_advanced_config.imported.id
# }
