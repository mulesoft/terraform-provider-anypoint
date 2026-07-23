# ---------------------------------------------------------------------------
# Import an existing self-managed (connected-mode) Flex/Omni Gateway into
# Terraform state.
#
# Steps:
#   1. Uncomment the relevant block below.
#   2. Replace the placeholder values with real IDs.
#      IDs can be found in Anypoint Runtime Manager or via the Anypoint API.
#   3. Run: terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_self_managed_gateway.imported \
#          <environment_id>/<name>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format:
#   Root org:  anypoint_self_managed_gateway -> <env_id>/<name>
#   Sub-org:   anypoint_self_managed_gateway -> <org_id>/<env_id>/<name>
#
# Use the sub-org format when the gateway belongs to a Business Group.
#
# NOTE: registration_token is a one-shot enrollment secret and is NOT
# recoverable on import — it will be null in state after importing. This does
# not affect a gateway whose runtime has already registered.
# ---------------------------------------------------------------------------

# --- Root org (2-part ID) ---
# locals {
#   env_id = "<environment_id>"
#   name   = "<gateway_name>"
# }
#
# import {
#   to = anypoint_self_managed_gateway.imported
#   id = "${local.env_id}/${local.name}"
# }

# --- Sub-org (3-part ID) ---
# locals {
#   org_id = "<organization_id>"
#   env_id = "<environment_id>"
#   name   = "<gateway_name>"
# }
#
# import {
#   to = anypoint_self_managed_gateway.imported
#   id = "${local.org_id}/${local.env_id}/${local.name}"
# }

# resource "anypoint_self_managed_gateway" "imported" {
#   name           = "<gateway_name>"
#   environment_id = "<environment_id>"
#
#   # Optional — set only when the gateway lives in a Business Group (sub-org).
#   # organization_id = "<organization_id>"
# }
