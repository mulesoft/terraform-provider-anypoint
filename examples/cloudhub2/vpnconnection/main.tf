terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
    time = {
      source  = "hashicorp/time"
      version = "0.13.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

# First create a private space with network
# resource "anypoint_private_space_config" "private_space" {
#   name = "example-private-space-vpn"
#   network {
#     region     = var.region_id
#     cidr_block = var.cidr_block
#   }
# }

# # Create VPN connection
# resource "anypoint_vpn_connection" "example" {
#   depends_on       = [anypoint_private_space_config.private_space]
#   private_space_id = anypoint_private_space_config.private_space.id
#   name             = var.connection_name
  
#   vpns = [
#     {
#       local_asn         = var.local_asn
#       remote_asn        = var.remote_asn
#       remote_ip_address = var.remote_ip_address
#       static_routes     = []
      
#       vpn_tunnels = [
#         {
#           psk           = var.psk_1
#           ptp_cidr      = var.ptp_cidr_1
#           startup_action = var.startup_action
#         },
#         {
#           psk           = var.psk_2
#           ptp_cidr      = var.ptp_cidr_2
#           startup_action = var.startup_action
#         }
#       ]
#     }
#   ]
# }

# # Outputs
# output "private_space_id" {
#   description = "The ID of the private space"
#   value       = anypoint_private_space_config.private_space.id
# }

# output "private_space_name" {
#   description = "The name of the private space"
#   value       = anypoint_private_space_config.private_space.name
# }

# output "network_cidr" {
#   description = "The CIDR block of the private network"
#   value       = anypoint_private_space_config.private_space.network.cidr_block
# }

# output "vpn_connection_id" {
#   description = "The ID of the VPN connection"
#   value       = anypoint_vpn_connection.example.id
# }

# output "vpn_connection_name" {
#   description = "The name of the VPN connection"
#   value       = anypoint_vpn_connection.example.name
# }

# output "vpn_connection_status" {
#   description = "The status of the VPN connection"
#   value       = length(anypoint_vpn_connection.example.vpns) > 0 ? anypoint_vpn_connection.example.vpns[0].vpn_connection_status : null
# }

# output "vpn_connection_details" {
#   description = "Details of the VPN connection"
#   value       = anypoint_vpn_connection.example.vpns
# }

# Create a private space with network
resource "anypoint_private_space_config" "private_space_custom_org" {
  name            = var.private_space_name
  organization_id = var.organization_id

  network {
    region     = var.region_id
    cidr_block = "10.0.0.0/22"
  }
}

# Introduce a delay to ensure the private network is fully initialized
resource "time_sleep" "wait_for_network_custom_org" {
  depends_on      = [anypoint_private_space_config.private_space_custom_org]
  create_duration = "10s"
}

# Create VPN connection
resource "anypoint_vpn_connection" "example_custom_org" {
  depends_on       = [time_sleep.wait_for_network_custom_org]
  private_space_id = anypoint_private_space_config.private_space_custom_org.id
  organization_id = var.organization_id
  name             = var.connection_name
  
  vpns = [
    {
      local_asn         = var.local_asn
      remote_asn        = var.remote_asn
      remote_ip_address = var.remote_ip_address
      static_routes     = []
      
      vpn_tunnels = [
        {
          psk           = var.psk_1
          ptp_cidr      = var.ptp_cidr_1
          startup_action = var.startup_action
        },
        {
          psk           = var.psk_2
          ptp_cidr      = var.ptp_cidr_2
          startup_action = var.startup_action
        }
      ]
    }
  ]
}