# Import an existing TLS context into Terraform.
#
# Steps:
#   1. Copy this file to import.tf (or paste the block into your existing .tf files)
#   2. Replace the placeholders with your actual IDs
#   3. Add a matching resource block, or run:
#        terraform plan -generate-config-out=generated.tf
#   4. Run: terraform apply
#
# Import ID format:
#   Root org:  anypoint_tls_context -> <private_space_id>:<tls_context_id>
#   Sub-org:   anypoint_tls_context -> <org_id>:<private_space_id>:<tls_context_id>
#
# Use the <org_id>:<private_space_id>:<tls_context_id> format when the private space was
# created in a Business Group (sub-org) rather than the root organization.

# --- Root org (2-part ID) ---
# locals {
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   tls_context_id   = "c9f3a2b1-1234-5678-90ab-cdef01234567"
# }
#
# import {
#   to = anypoint_tls_context.imported
#   id = "${local.private_space_id}:${local.tls_context_id}"
# }

# --- Sub-org (3-part ID) ---
# locals {
#   org_id           = "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
#   private_space_id = "849c361b-da3e-4c7d-9c68-a5784bb4dc58"
#   tls_context_id   = "c9f3a2b1-1234-5678-90ab-cdef01234567"
# }
#
# import {
#   to = anypoint_tls_context.imported
#   id = "${local.org_id}:${local.private_space_id}:${local.tls_context_id}"
# }

# resource "anypoint_tls_context" "imported" {
#   private_space_id = local.private_space_id
#   organization_id  = local.org_id
#   name             = "<tls_context_name>"
#   keystore_type    = "PEM"   # "PEM" or "JKS"
# }

# output "imported_id" {
#   value = anypoint_tls_context.imported.id
# }
