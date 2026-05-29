# Import an existing private space association into Terraform.
#
# Steps:
#   1. Copy this file to import.tf (or paste the block into your existing .tf files)
#   2. Replace the placeholders with your actual IDs
#   3. Add a matching resource block, or run:
#        terraform plan -generate-config-out=generated.tf
#   4. Run: terraform apply
#
# Import ID format:
#   Root org:  anypoint_private_space_association -> <private_space_id>
#   Sub-org:   anypoint_private_space_association -> <org_id>/<private_space_id>
#
# Use the <org_id>/<private_space_id> format when the private space was created
# in a Business Group (sub-org) rather than the root organization.

# --- Root org (simple ID) ---
# locals {
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
# }
#
# import {
#   to = anypoint_private_space_association.imported
#   id = local.private_space_id
# }

# --- Sub-org (composite ID) ---
# locals {
#   org_id           = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
# }
#
# import {
#   to = anypoint_private_space_association.imported
#   id = "${local.org_id}/${local.private_space_id}"
# }

# resource "anypoint_private_space_association" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = local.org_id
#
#   associations = [
#     {
#       organization_id = "<associated_org_id>"
#       environment     = "all"
#     }
#   ]
# }

# output "imported_id" {
#   value = anypoint_private_space_association.imported.id
# }
