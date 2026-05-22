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
# Import an existing Agent instance into Terraform state.
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values in the locals block.
#   3. Run: terraform init && terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_agent_instance.imported \
#          <org_id>/<env_id>/<agent_instance_id>
#   4. Run: terraform plan — review the diff and adjust the resource block.
#
# Import ID format:
#   anypoint_agent_instance -> organization_id/environment_id/agent_instance_id
#
# The agent_instance_id is the numeric ID from Anypoint API Manager
# (visible in the URL when viewing the instance, e.g. "16563478").
# ---------------------------------------------------------------------------

# locals {
#   org_id              = "<org_id>"
#   env_id              = "<env_id>"
#   agent_instance_id   = "<agent_instance_id>"  # numeric ID from Anypoint API Manager
# }

# import {
#   to = anypoint_agent_instance.imported
#   id = "${local.org_id}/${local.env_id}/${local.agent_instance_id}"
# }

# resource "anypoint_agent_instance" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#
#   instance_label = "<instance_label>"
#
#   spec = {
#     asset_id = "<agent_asset_id>"
#     group_id = local.org_id
#     version  = "1.0.0"
#   }
#
#   # For a single upstream use upstream_uri (mutually exclusive with routing).
#   upstream_uri = "http://<backend-host>:8080"
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

# output "imported_agent_instance_id" {
#   description = "Numeric ID of the imported Agent instance"
#   value       = anypoint_agent_instance.imported.id
# }
#
# output "imported_agent_instance_status" {
#   description = "Status of the imported Agent instance"
#   value       = anypoint_agent_instance.imported.status
# }
#
# output "imported_agent_instance_asset_id" {
#   description = "Exchange asset ID of the imported Agent instance"
#   value       = anypoint_agent_instance.imported.asset_id
# }
