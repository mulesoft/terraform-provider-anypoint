# Import an existing private space upgrade schedule into Terraform.
#
# Steps:
#   1. Copy this file to import.tf (or paste the block into your existing .tf files)
#   2. Replace the placeholders with your actual values
#   3. Add a matching resource block, or run:
#        terraform plan -generate-config-out=generated.tf
#   4. Run: terraform apply
#
# Import ID format:
#   Root org:  anypoint_private_space_upgrade -> <private_space_id>:<date>:<opt_in>
#   Sub-org:   anypoint_private_space_upgrade -> <org_id>:<private_space_id>:<date>:<opt_in>
#
# Use the <org_id>:<private_space_id>:<date>:<opt_in> format when the private space was
# created in a Business Group (sub-org) rather than the root organization.

# --- Root org (3-part ID) ---
# locals {
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   upgrade_date     = "2025-08-12"
#   opt_in           = "true"
# }
#
# import {
#   to = anypoint_private_space_upgrade.imported
#   id = "${local.private_space_id}:${local.upgrade_date}:${local.opt_in}"
# }

# --- Sub-org (4-part ID) ---
# locals {
#   org_id           = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   upgrade_date     = "2025-08-12"
#   opt_in           = "true"
# }
#
# import {
#   to = anypoint_private_space_upgrade.imported
#   id = "${local.org_id}:${local.private_space_id}:${local.upgrade_date}:${local.opt_in}"
# }

# resource "anypoint_private_space_upgrade" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = local.org_id
#   date             = local.upgrade_date
#   opt_in           = true
# }

# output "imported_id" {
#   value = anypoint_private_space_upgrade.imported.id
# }
