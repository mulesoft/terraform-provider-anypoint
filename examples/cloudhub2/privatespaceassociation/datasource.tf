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

# Example: Fetch information about an existing private space association
# data "anypoint_private_space_associations" "example" {  
#   private_space_id = "f7dcdb6c-017d-4989-8d87-28e8477412e0"
# }

# # Output the association information
# output "associations" {
#   value = data.anypoint_private_space_associations.example.associations
# }

# output "association_environment_id" {
#   value = data.anypoint_private_space_associations.example.environment_id
# } 