# terraform {
#   required_providers {
#     anypoint = {
#       source = "sf.com/mulesoft/anypoint"
#       version = "0.1.0"
#     }
#   }
# }

# provider "anypoint" {
#   client_id = var.anypoint_client_id
#   client_secret = var.anypoint_client_secret
#   base_url = var.anypoint_base_url
# }

# # Example 1: Simple role group without external names
# resource "anypoint_rolegroup" "simple_example" {
#   name        = "Organization Administrators Terraform Updated"
#   description = "Administrators for the organization"
# }

# # Example 2: Role group with external names
# # resource "anypoint_rolegroup" "external_example" {
# #   name        = "External Administrators"
# #   description = "External group administrators"
  
# #   external_names = [
# #     {
# #       external_group_name = "administrators"
# #       provider_id         = "2e50e859-0042-46ff-8cf8-1ad6f0c78b67"
# #     },
# #     {
# #       external_group_name = "admins"
# #       provider_id         = "2e50e859-0042-46ff-8cf8-1ad6f0c78b67"
# #     }
# #   ]
# # }

# # Output examples
# output "simple_rolegroup_id" {
#   description = "The ID of the simple role group"
#   value       = {
#     id          = anypoint_rolegroup.simple_example.id
#     name        = anypoint_rolegroup.simple_example.name
#     description = anypoint_rolegroup.simple_example.description
#     org_id      = anypoint_rolegroup.simple_example.org_id
#     editable    = anypoint_rolegroup.simple_example.editable
#     created_at  = anypoint_rolegroup.simple_example.created_at
#     updated_at  = anypoint_rolegroup.simple_example.updated_at
#   }
# }

# # output "external_rolegroup_details" {
# #   description = "Details of the external role group"
# #   value = {
# #     id          = anypoint_rolegroup.external_example.id
# #     name        = anypoint_rolegroup.external_example.name
# #     description = anypoint_rolegroup.external_example.description
# #     org_id      = anypoint_rolegroup.external_example.org_id
# #     editable    = anypoint_rolegroup.external_example.editable
# #     created_at  = anypoint_rolegroup.external_example.created_at
# #     updated_at  = anypoint_rolegroup.external_example.updated_at
# #   }
# # } 