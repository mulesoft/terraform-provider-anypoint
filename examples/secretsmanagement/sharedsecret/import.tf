# Import an existing Anypoint Secret Group Shared Secret into Terraform state.
#
# Steps:
#   1. Replace the placeholder IDs below with your actual values.
#   2. Uncomment the import block and resource block.
#   3. Run: terraform init && terraform apply
#      OR use CLI: terraform import anypoint_secret_group_shared_secret.imported <import_id>
#
# Import ID format:
#   anypoint_secret_group_shared_secret -> organization_id/environment_id/secret_group_id/shared_secret_id

# terraform {
#   required_providers {
#     anypoint = {
#       source = "mulesoft/anypoint"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# locals {
#   org_id           = "<org_id>"
#   env_id           = "<env_id>"
#   secret_group_id  = "<secret_group_id>"
#   shared_secret_id = "<shared_secret_id>"
# }

# import {
#   to = anypoint_secret_group_shared_secret.imported
#   id = "${local.org_id}/${local.env_id}/${local.secret_group_id}/${local.shared_secret_id}"
# }

# resource "anypoint_secret_group_shared_secret" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   secret_group_id = local.secret_group_id
#   name            = "<name>"
#   type            = "UsernamePassword"  # e.g. "UsernamePassword", "S3Credential"
#   # Add type-specific fields as needed, for example:
#   # username = "<username>"
#   # password = "<password>"   # sensitive; not populated after import
# }

# output "imported_id" {
#   value = anypoint_secret_group_shared_secret.imported.id
# }

# variable "anypoint_client_id" {
#   description = "Connected App client ID"
#   type        = string
#   sensitive   = true
#   default     = "<anypoint_connected_app_client_id>"
# }

# variable "anypoint_client_secret" {
#   description = "Connected App client secret"
#   type        = string
#   sensitive   = true
#   default     = "<anypoint_connected_app_client_secret>"
# }

# variable "anypoint_base_url" {
#   description = "Anypoint control-plane URL"
#   type        = string
#   default     = "https://anypoint.mulesoft.com"
# }
