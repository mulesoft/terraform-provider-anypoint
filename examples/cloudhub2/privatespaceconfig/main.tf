terraform {
  required_providers {
    anypoint = {
      source  = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# # Minimal: private space + network only
# resource "anypoint_private_space_config" "basic_minimal" {
#   name = "${var.private_space_name}-basic"
# }

# # Minimal: private space + network only
# resource "anypoint_private_space_config" "basic" {
#   name = var.private_space_name

#   network {
#     region     = var.region_id
#     cidr_block = var.cidr_block
#   }
# }

# # Full: private space + network + firewall rules
# resource "anypoint_private_space_config" "full" {
#   name            = "${var.private_space_name}-full-v2"
#   enable_egress   = true
#   enable_iam_role = false

#   network {
#     region         = var.region_id
#     cidr_block     = var.cidr_block
#     reserved_cidrs = var.reserved_cidrs
#   }

#   firewall_rules = [
#     {
#       cidr_block = "0.0.0.0/0"
#       protocol   = "tcp"
#       from_port  = 30500
#       to_port    = 32500
#       type       = "inbound"
#     },
#     {
#       cidr_block = "0.0.0.0/0"
#       protocol   = "tcp"
#       from_port  = 0
#       to_port    = 65535
#       type       = "outbound"
#     },
#   ]
# }

# output "private_space_id" {
#   description = "The ID of the private space"
#   value       = anypoint_private_space_config.full.id
# }

# output "private_space_status" {
#   description = "The status of the private space"
#   value       = anypoint_private_space_config.full.status
# }

# output "network_region" {
#   description = "The region of the private network"
#   value       = anypoint_private_space_config.full.network.region
# }

# output "network_dns_target" {
#   description = "The DNS target for the private network"
#   value       = anypoint_private_space_config.full.network.dns_target
# }

# output "network_inbound_static_ips" {
#   description = "Inbound static IPs for the private network"
#   value       = anypoint_private_space_config.full.network.inbound_static_ips
# }

# output "network_outbound_static_ips" {
#   description = "Outbound static IPs for the private network"
#   value       = anypoint_private_space_config.full.network.outbound_static_ips
# }
