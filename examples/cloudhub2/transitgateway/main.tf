terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# Create Transit Gateway in Private Space
resource "anypoint_transit_gateway" "example" {
  private_space_id         = var.private_space_id
  name                     = var.transit_gateway_name
  resource_share_id        = var.resource_share_id
  resource_share_account   = var.resource_share_account
  routes                   = var.routes
}

# Outputs
output "transit_gateway_id" {
  description = "The ID of the Transit Gateway"
  value       = anypoint_transit_gateway.example.id
}

output "transit_gateway_name" {
  description = "The name of the Transit Gateway"
  value       = anypoint_transit_gateway.example.name
}

output "transit_gateway_spec" {
  description = "The specification of the Transit Gateway"
  value       = anypoint_transit_gateway.example.spec
}

output "transit_gateway_status" {
  description = "The status of the Transit Gateway"
  value       = anypoint_transit_gateway.example.status
}

output "gateway_region" {
  description = "The region of the Transit Gateway"
  value       = anypoint_transit_gateway.example.spec.region
}

output "resource_share_info" {
  description = "Resource share information"
  value = {
    id      = anypoint_transit_gateway.example.spec.resource_share.id
    account = anypoint_transit_gateway.example.spec.resource_share.account
  }
}

output "gateway_status_summary" {
  description = "Summary of the Transit Gateway status"
  value = {
    gateway_status    = anypoint_transit_gateway.example.status.gateway
    attachment_status = anypoint_transit_gateway.example.status.attachment
    tgw_resource      = anypoint_transit_gateway.example.status.tgw_resource
    active_routes     = anypoint_transit_gateway.example.status.routes
  }
} 