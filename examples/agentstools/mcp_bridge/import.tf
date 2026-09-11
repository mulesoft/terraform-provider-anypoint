# terraform {
#   required_providers {
#     anypoint = {
#       source = "mulesoft/anypoint"
#     }
#   }
# }
#
# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

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
# Import reconstructs source_apis (including tls_context_id and custom
# http_mapping). description and input_schema come back null — they live only
# in the generated asset. Instance settings are recovered.
#
# Terraform >= 1.5 tip: add only the import {} block, then:
#   terraform plan -generate-config-out=generated.tf
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
#   port           = 8081
#   base_path      = "<base_path>"
#
#   instance_label    = "<instance_label>"
#   approval_method   = "manual"
#   consumer_endpoint = "https://<ingress>/<base_path>"
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
