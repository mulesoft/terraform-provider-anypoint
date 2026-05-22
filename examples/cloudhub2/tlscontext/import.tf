# Import an existing TLS context into Terraform.
#
# Steps:
#   1. Replace the placeholders with your actual IDs.
#   2. Uncomment the import and resource blocks.
#   3. Run: terraform init && terraform apply
#      OR use CLI: terraform import anypoint_tls_context.imported <private_space_id>:<tls_context_id>
#
# Import ID format:
#   anypoint_tls_context -> <private_space_id>:<tls_context_id>
#   Example: 675c4efb-d44e-44cd-ac6f-d5a1128e6236:a1b2c3d4-e5f6-7890-abcd-ef1234567890

# locals {
#   private_space_id = "<private_space_id>"   # e.g. "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   tls_context_id   = "<tls_context_id>"     # e.g. "c9f3a2b1-1234-5678-90ab-cdef01234567"
# }

# import {
#   to = anypoint_tls_context.imported
#   id = "${local.private_space_id}:${local.tls_context_id}"
# }

# resource "anypoint_tls_context" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = "<organization_id>"
#   name             = "<tls_context_name>"
#   keystore_type    = "PEM"   # "PEM" or "JKS"
# }

# output "imported_id" {
#   value = anypoint_tls_context.imported.id
# }
