# Import an existing private space association into Terraform.
#
# Steps:
#   1. Replace the placeholders with your actual IDs.
#   2. Uncomment the import and resource blocks.
#   3. Run: terraform init && terraform apply
#      OR use CLI: terraform import anypoint_private_space_association.imported <private_space_id>
#
# Import ID format:
#   anypoint_private_space_association -> <private_space_id>

# locals {
#   private_space_id = "<private_space_id>"   # e.g. "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
# }

# import {
#   to = anypoint_private_space_association.imported
#   id = local.private_space_id
# }

# resource "anypoint_private_space_association" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = "<organization_id>"
#
#   associations = [
#     {
#       id              = "<association_id>"
#       organization_id = "<associated_org_id>"
#       environment     = "all"
#     }
#   ]
# }

# output "imported_id" {
#   value = anypoint_private_space_association.imported.id
# }
