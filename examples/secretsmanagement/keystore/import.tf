# Import an existing Anypoint Secret Group Keystore into Terraform state.
#
# Steps:
#   1. Replace the placeholder IDs below with your actual values.
#   2. Uncomment the import block and resource block.
#   3. Run: terraform init && terraform apply
#      OR use CLI: terraform import anypoint_secret_group_keystore.imported <import_id>
#
# Import ID format:
#   anypoint_secret_group_keystore -> organization_id/environment_id/secret_group_id/keystore_id
#
# Note: File content fields (certificate_base64, key_base64, keystore_file_base64, etc.)
#       are write-only and will not be populated after import. Set them manually in the
#       resource block to avoid drift on the next plan.

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
#   org_id          = "<org_id>"
#   env_id          = "<env_id>"
#   secret_group_id = "<secret_group_id>"
#   keystore_id     = "<keystore_id>"
# }

# import {
#   to = anypoint_secret_group_keystore.imported
#   id = "${local.org_id}/${local.env_id}/${local.secret_group_id}/${local.keystore_id}"
# }

# resource "anypoint_secret_group_keystore" "imported" {
#   organization_id = local.org_id
#   environment_id  = local.env_id
#   secret_group_id = local.secret_group_id
#   name            = "<name>"
#   type            = "PEM"   # "PEM", "JKS", or "PKCS12"
#   # File content fields are write-only; set them to avoid drift after import.
#   # For PEM type:
#   #   certificate_base64 = base64encode(file("<path/to/cert.pem>"))
#   #   key_base64         = base64encode(file("<path/to/key.pem>"))
#   # For JKS/PKCS12 type:
#   #   keystore_file_base64 = filebase64("<path/to/keystore.jks>")
#   #   store_passphrase     = "<store_passphrase>"
#   #   key_passphrase       = "<key_passphrase>"
#   #   alias                = "<alias>"
# }

# output "imported_id" {
#   value = anypoint_secret_group_keystore.imported.id
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
