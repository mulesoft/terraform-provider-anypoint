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
# Import an existing SLA tier into Terraform state.
#
# Import ID format:
#   organization_id/environment_id/api_instance_id/<tier_name_or_tier_id>
#
# The last segment accepts either:
#   - The tier name  (visible in the Anypoint UI)         e.g. "Gold"
#   - The numeric ID (from the API response)              e.g. "1656119"
#
# Using the name is recommended — it is visible in the UI without any API calls.
#
# Steps:
#   1. Uncomment all blocks below.
#   2. Replace the placeholder values in the locals block.
#   3. Run: terraform init && terraform plan -generate-config-out=generated.tf
#   4. Review generated.tf, then run: terraform apply
#      OR use the CLI import command (Terraform < 1.5):
#        terraform import anypoint_api_instance_sla_tier.imported \
#          <org_id>/<env_id>/<api_instance_id>/<tier_name>
# ---------------------------------------------------------------------------

# locals {
#   org_id          = "<org_id>"
#   env_id          = "<env_id>"
#   api_instance_id = "<api_instance_id>"   # numeric API instance ID
#   tier_name       = "<tier_name>"         # as shown in the Anypoint UI
# }

# import {
#   to = anypoint_api_instance_sla_tier.imported
#   id = "${local.org_id}/${local.env_id}/${local.api_instance_id}/${local.tier_name}"
# }

# resource "anypoint_api_instance_sla_tier" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   api_instance_id = local.api_instance_id
#
#   name        = "<tier_name>"
#   description = "<tier_description>"
#
#   # At least one limit entry is required.
#   # After import, run `terraform show` to discover the actual limits.
#   limits = [
#     {
#       time_period_in_milliseconds = 60000   # 1 minute
#       maximum_requests            = 100
#       visible                     = true
#     }
#   ]
#
#   auto_approve = true
#   status       = "ACTIVE"   # "ACTIVE" | "DEPRECATED"
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

# output "imported_sla_tier_id" {
#   description = "Numeric ID of the imported SLA tier"
#   value       = anypoint_api_instance_sla_tier.imported.id
# }
#
# output "imported_sla_tier_name" {
#   description = "Name of the imported SLA tier"
#   value       = anypoint_api_instance_sla_tier.imported.name
# }
#
# output "imported_sla_tier_status" {
#   description = "Status of the imported SLA tier"
#   value       = anypoint_api_instance_sla_tier.imported.status
# }
