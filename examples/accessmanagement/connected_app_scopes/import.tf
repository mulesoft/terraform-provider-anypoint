# Import an existing Anypoint Connected App's scopes into Terraform state.
#
# Steps:
#   1. Replace the placeholder with your actual connected app ID (UUID).
#   2. Uncomment the import block and the resource block below.
#   3. Run: terraform init && terraform apply
#      OR use the CLI:
#        terraform import anypoint_connected_app_scopes.imported <connected_app_id>
#
# Import ID format:
#   anypoint_connected_app_scopes -> <connected_app_id>
#   After import, both `id` and `connected_app_id` are set to the same UUID.
#
# Note: This resource requires user (username/password) credentials on the
# provider because the Connected App Scopes API uses password-grant auth.

# locals {
#   connected_app_id = "<connected_app_id>"  # e.g. "a1b2c3d4-e5f6-7890-abcd-ef1234567890"
# }

# import {
#   provider = anypoint.admin
#   to       = anypoint_connected_app_scopes.imported
#   id       = local.connected_app_id
# }

# resource "anypoint_connected_app_scopes" "imported" {
#   provider         = anypoint.admin
#   connected_app_id = local.connected_app_id
#
#   scopes = [
#     {
#       scope          = "admin:cloudhub"
#       context_params = {}
#     },
#     # Add additional scopes that were assigned to this connected app.
#   ]
# }

# output "imported_connected_app_scopes_id" {
#   value = anypoint_connected_app_scopes.imported.id
# }
