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
# Import an existing MCP server into Terraform state.
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values in the locals block.
#   3. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_mcp_server.imported \
#          <org_id>/<env_id>/<mcp_server_id>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format:
#   anypoint_mcp_server -> organization_id/environment_id/mcp_server_id
#
# The mcp_server_id is the numeric ID from Anypoint API Manager
# (visible in the URL when viewing the instance, e.g. "16563478").
# ---------------------------------------------------------------------------

# locals {
#   org_id        = "<org_id>"
#   env_id        = "<env_id>"
#   mcp_server_id = "<mcp_server_id>"  # numeric ID from Anypoint API Manager
# }

# import {
#   to = anypoint_mcp_server.imported
#   id = "${local.org_id}/${local.env_id}/${local.mcp_server_id}"
# }

# resource "anypoint_mcp_server" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#
#   instance_label = "<instance_label>"
#
#   spec = {
#     asset_id = "<mcp_server_asset_id>"
#     group_id = local.org_id
#     version  = "1.0.0"
#   }
#
#   # For a single upstream use upstream_uri (mutually exclusive with routing).
#   upstream_uri = "http://<mcp-server-host>:8080"
#
#   endpoint = {
#     base_path = "/<base_path>"
#   }
# }

# ---------------------------------------------------------------------------
# Variables
# ---------------------------------------------------------------------------

# variable "anypoint_client_id" {
#   description = "Anypoint Platform Connected App client ID"
#   type        = string
# }
#
# variable "anypoint_client_secret" {
#   description = "Anypoint Platform Connected App client secret"
#   type        = string
#   sensitive   = true
# }
#
# variable "anypoint_base_url" {
#   description = "Anypoint Platform base URL"
#   type        = string
#   default     = "https://anypoint.mulesoft.com"
# }

# ---------------------------------------------------------------------------
# Outputs
# ---------------------------------------------------------------------------

# output "imported_mcp_server_id" {
#   description = "Numeric ID of the imported MCP server"
#   value       = anypoint_mcp_server.imported.id
# }
#
# output "imported_mcp_server_status" {
#   description = "Status of the imported MCP server"
#   value       = anypoint_mcp_server.imported.status
# }
#
# output "imported_mcp_server_asset_id" {
#   description = "Exchange asset ID of the imported MCP server"
#   value       = anypoint_mcp_server.imported.asset_id
# }
#
# output "imported_mcp_server_upstream_id" {
#   description = "Server-assigned upstream ID of the imported MCP server"
#   value       = anypoint_mcp_server.imported.upstream_id
# }
