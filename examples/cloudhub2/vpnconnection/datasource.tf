# # Data source to fetch a VPN connection by ID
# data "anypoint_vpn_connection" "example" {
#   private_space_id = var.private_space_id
#   connection_id    = var.vpn_connection_id
# }
# 
# # Output the VPN connection details
# output "exisiting_vpn_connection_details" {
#   value = data.anypoint_vpn_connection.example
# }
# 
# # Output specific VPN connection information
# output "existing_vpn_connection_name" {
#   value = data.anypoint_vpn_connection.example.name
# }
# 
# output "existing_vpn_connection_id" {
#   value = data.anypoint_vpn_connection.example.id
# }
# 
# # Output VPN configurations
# output "existing_vpn_configurations" {
#   value = data.anypoint_vpn_connection.example.vpns
# }
# 
# # Output VPN count
# output "existing_vpn_count" {
#   value = length(data.anypoint_vpn_connection.example.vpns)
# }
# 
# # Example of extracting specific VPN information
# output "existing_vpn_statuses" {
#   value = [
#     for vpn in data.anypoint_vpn_connection.example.vpns :
#     {
#       name   = vpn.name
#       status = vpn.connection_status
#       asn    = vpn.local_asn
#     }
#   ]
# }
# 
# # Example of filtering VPNs by status
# locals {
#   available_vpns = [
#     for vpn in data.anypoint_vpn_connection.example.vpns :
#     vpn if vpn.connection_status == "available"
#   ]
# }
# 
# output "existing_available_vpns" {
#   value = local.available_vpns
# }
# 
# # Example of extracting tunnel information
# output "existing_tunnel_configurations" {
#   value = {
#     for vpn in data.anypoint_vpn_connection.example.vpns :
#     vpn.name => vpn.vpn_tunnels
#   }
# } 