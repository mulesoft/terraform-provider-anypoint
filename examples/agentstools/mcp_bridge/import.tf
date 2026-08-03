# ---------------------------------------------------------------------------
# Import an existing MCP bridge into Terraform state.
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values in the locals block.
#   3. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_mcp_bridge.imported \
#          <org_id>/<env_id>/<mcp_bridge_id>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format:
#   anypoint_mcp_bridge -> organization_id/environment_id/mcp_bridge_id
#
# The mcp_bridge_id is the numeric ID from Anypoint API Manager
# (visible in the URL when viewing the instance, e.g. "21058094").
#
# On import the provider reconstructs source_apis (labels, upstreams, tools)
# from the live routing + transcoding policies. Tool descriptions live only in
# the generated asset metadata and are left null after import; add them back to
# your config as desired (they will not cause a spurious diff).
#
# TIP (Terraform >= 1.5): to have Terraform WRITE the resource block for you,
# add only the import {} block below, then run:
#     terraform plan -generate-config-out=generated.tf
# and review generated.tf. It will contain every source API and tool (with
# query_params / has_body) reconstructed from the live bridge.
# ---------------------------------------------------------------------------

# locals {
#   org_id        = "<org_id>"
#   env_id        = "<env_id>"
#   mcp_bridge_id = "<mcp_bridge_id>"  # numeric ID from Anypoint API Manager
# }

# import {
#   to = anypoint_mcp_bridge.imported
#   id = "${local.org_id}/${local.env_id}/${local.mcp_bridge_id}"
# }

# resource "anypoint_mcp_bridge" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   gateway_id      = "<gateway_id>"
#
#   mcp_asset_name = "<mcp_asset_name>"
#   base_path   = "<base_path>"
#
#   source_apis = [
#     {
#       label        = "<source_api_label>"
#       upstream_uri = "https://<backend-host>"
#       asset_id     = "<source_rest_api_asset_id>"
#       version      = "1.0.0"
#       tools = [
#         { method = "GET", path = "/example/{id}" },
#       ]
#     },
#   ]
# }
