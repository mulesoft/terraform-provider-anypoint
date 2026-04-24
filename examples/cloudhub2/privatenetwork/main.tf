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

# # First create a private space
resource "anypoint_private_space" "private_space" {
  name   = "example-private-space-with-network"
  region = var.region_id
}

# Then configure its network
resource "anypoint_private_network" "private_network" {
  private_space_id = anypoint_private_space.private_space.id
  region           = var.region_id
  cidr_block       = var.cidr_block
  # reserved_cidrs   = var.reserved_cidrs
}

resource "anypoint_private_network" "private_network_custom_org" {
  private_space_id = var.private_space_id
  organization_id = var.organization_id
  region           = var.region_id
  cidr_block       = "10.0.0.0/18"
}

# Outputs
# output "private_space_id" {
#   description = "The ID of the private space"
#   value       = anypoint_private_space.private_space.id
# }

output "private_network_id" {
  description = "The ID of the private network"
  value       = anypoint_private_network.private_network.id
}

output "private_network_name" {
  description = "The name of the private network"
  value       = anypoint_private_network.private_network.name
}

output "private_network_cidr_block" {
  description = "The CIDR block of the private network"
  value       = anypoint_private_network.private_network.cidr_block
}

output "private_network_region" {
  description = "The region of the private network"
  value       = anypoint_private_network.private_network.region
}

output "private_network_dns_target" {
  description = "The DNS target for the private network"
  value       = anypoint_private_network.private_network.dns_target
}

output "private_network_reserved_cidrs" {
  description = "The reserved CIDRs for the private network"
  value       = anypoint_private_network.private_network.reserved_cidrs
}

output "private_network_inbound_static_ips" {
  description = "The inbound static IPs for the private network"
  value       = anypoint_private_network.private_network.inbound_static_ips
}

output "private_network_inbound_internal_static_ips" {
  description = "The inbound internal static IPs for the private network"
  value       = anypoint_private_network.private_network.inbound_internal_static_ips
}

output "private_network_outbound_static_ips" {
  description = "The outbound static IPs for the private network"
  value       = anypoint_private_network.private_network.outbound_static_ips
} 