# terraform {
#   required_providers {
#     anypoint = {
#       source = "sfprod.com/mulesoft/anypoint"
#     }
#   }
# }

# provider "anypoint" {
#   client_id     = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url      = var.anypoint_base_url
# }

# # Example: Fetch information about an existing TLS context
# data "anypoint_tls_context" "example" {
#   id               = "your-tls-context-id"
#   private_space_id = "your-private-space-id"
# }

# # Output the TLS context information
# output "tls_context_name" {
#   value = data.anypoint_tls_context.example.name
# }

# output "tls_context_target" {
#   value = data.anypoint_tls_context.example.target
# }

# output "tls_context_private_space_id" {
#   value = data.anypoint_tls_context.example.private_space_id
# }

# output "tls_context_organization_id" {
#   value = data.anypoint_tls_context.example.organization_id
# }

# output "tls_context_created_at" {
#   value = data.anypoint_tls_context.example.created_at
# }

# output "tls_context_updated_at" {
#   value = data.anypoint_tls_context.example.updated_at
# } 