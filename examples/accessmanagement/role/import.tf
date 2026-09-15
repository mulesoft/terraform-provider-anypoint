# ---------------------------------------------------------------------------
# Import an existing role group into Terraform state.
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values in the locals block.
#   3. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_role.imported <role_group_id>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format:
#   anypoint_role -> role_group_id
#
# The role_group_id is the UUID shown in the URL when viewing the role in
# Access Management > Roles.
#
# Terminology: what the Anypoint UI calls a "Role" is a role GROUP, and what it
# calls "Permissions" are the entries in the `permissions` attribute below. The
# same naming is used by anypoint_team.
# ---------------------------------------------------------------------------

# locals {
#   role_group_id = "<role_group_id>"
#   org_id        = "<org_id>"
# }

# import {
#   to = anypoint_role.imported
#   id = local.role_group_id
# }

# resource "anypoint_role" "imported" {
#   name            = "<role_name>"
#   description     = "<role_description>"
#   organization_id = local.org_id
#
#   # Permissions are authoritative: any permission not listed here is removed
#   # on apply. Omit the attribute entirely to leave them unmanaged. Discover
#   # valid names with the anypoint_available_permissions data source.
#   permissions = [
#     {
#       name           = "Exchange Viewer"
#       context_params = { org = local.org_id }
#     },
#   ]
# }

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

# output "imported_role_id" {
#   description = "ID of the imported role group"
#   value       = anypoint_role.imported.id
# }
#
# output "imported_role_permissions" {
#   description = "Permissions granted by the imported role group"
#   value       = anypoint_role.imported.permissions
# }
