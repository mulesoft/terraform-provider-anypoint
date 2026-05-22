# Import an existing private space upgrade schedule into Terraform.
#
# Steps:
#   1. Replace the placeholders with your actual values.
#   2. Uncomment the import and resource blocks.
#   3. Run: terraform init && terraform apply
#      OR use CLI: terraform import anypoint_private_space_upgrade.imported <private_space_id>:<date>:<opt_in>
#
# Import ID format:
#   anypoint_private_space_upgrade -> <private_space_id>:<date>:<opt_in>
#   Example: 675c4efb-d44e-44cd-ac6f-d5a1128e6236:2025-08-12:true

# locals {
#   private_space_id = "<private_space_id>"   # e.g. "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   upgrade_date     = "<date>"               # e.g. "2025-08-12"
#   opt_in           = "<opt_in>"             # "true" or "false"
# }

# import {
#   to = anypoint_private_space_upgrade.imported
#   id = "${local.private_space_id}:${local.upgrade_date}:${local.opt_in}"
# }

# resource "anypoint_private_space_upgrade" "imported" {
#   private_space_id = local.private_space_id
#   date             = local.upgrade_date
#   opt_in           = true
# }

# output "imported_id" {
#   value = anypoint_private_space_upgrade.imported.id
# }
