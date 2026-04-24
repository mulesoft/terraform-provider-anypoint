# Example: Fetch information about an existing private network (private space network configuration)
# data "anypoint_private_network" "example" {
#   id = "93b81521-9fcc-429a-9f29-976d65d1e929"  # Note: This uses the private space ID
# }

# # Output the private network information
# output "exisiting_private_network_name" {
#   value = data.anypoint_private_network.example.name
# }

# output "private_network_cidr_block" {
#   value = data.anypoint_private_network.example.cidr_block
# }

# output "private_network_region" {
#   value = data.anypoint_private_network.example.region
# }

# output "private_network_dns_target" {
#   value = data.anypoint_private_network.example.dns_target
# } 