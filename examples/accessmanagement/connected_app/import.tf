# ---------------------------------------------------------------------------
# Import an existing Connected App into Terraform state.
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values in the locals block.
#   3. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_connected_app.imported \
#          <org_id>:<client_id>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format (note the COLON separator, not a slash):
#   anypoint_connected_app -> organization_id:client_id
#                          -> client_id            (organization inferred from
#                                                   the provider credentials)
#
# The client_id is shown on the Connected App's page in Access Management.
#
# NOTE: client_secret is NOT recoverable on import — the platform never returns
# it after creation. It reads back empty, which is expected and not drift. Put
# the real value in your configuration if you need Terraform to manage it.
# ---------------------------------------------------------------------------

# locals {
#   org_id    = "<org_id>"
#   client_id = "<client_id>"
# }

# import {
#   to = anypoint_connected_app.imported
#   id = "${local.org_id}:${local.client_id}"
# }

# resource "anypoint_connected_app" "imported" {
#   name            = "<connected_app_name>"
#   organization_id = local.org_id
#
#   # Grant types determine where scopes live. "client_credentials" apps carry
#   # scopes as a separate collection; "authorization_code" apps store them on
#   # the app body. The provider handles both transparently.
#   grant_types = ["client_credentials"]
#   audience    = "internal"
#   enabled     = true
#
#   # Scopes are authoritative: any scope not listed here is removed on apply.
#   # Omit the attribute entirely to leave them unmanaged.
#   scopes = [
#     {
#       scope          = "Exchange Viewer"
#       context_params = { org = local.org_id }
#     },
#   ]
# }

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

# output "imported_connected_app_id" {
#   description = "Client ID of the imported connected app"
#   value       = anypoint_connected_app.imported.id
# }
#
# output "imported_connected_app_scopes" {
#   description = "Scopes currently granted to the imported connected app"
#   value       = anypoint_connected_app.imported.scopes
# }
