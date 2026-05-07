###############################################################################
# Outputs
###############################################################################

output "organization_id" {
  description = "Commerce Business Unit organization ID"
  value       = anypoint_organization.commerce_bu.id
}

output "sandbox_environment_id" {
  description = "Sandbox environment ID"
  value       = anypoint_environment.sandbox.id
}

output "production_environment_id" {
  description = "Production environment ID"
  value       = anypoint_environment.production.id
}

output "private_space_id" {
  description = "Private Space ID"
  value       = anypoint_private_space_config.private_space.id
}

output "omni_gateway_id" {
  description = "Managed Omni Gateway ID"
  value       = anypoint_managed_omni_gateway.commerce-gateway.id
}

# output "orders_api_id" {
#   description = "Orders API instance ID in Sandbox"
#   value       = anypoint_api_instance.orders_api.id
# }

# output "payments_api_id" {
#   description = "Payments API instance ID in Sandbox"
#   value       = anypoint_api_instance.payments_api.id
# }

