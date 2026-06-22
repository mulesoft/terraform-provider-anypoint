# Retrieve private space configuration details
data "anypoint_private_space_config" "example" {
  id              = var.private_space_id
  organization_id = var.organization_id
}

# Output the private space status
output "private_space_status" {
  value = data.anypoint_private_space_config.example.status
}

# Output network configuration
output "network_region" {
  value = data.anypoint_private_space_config.example.network.region
}

output "network_cidr_block" {
  value = data.anypoint_private_space_config.example.network.cidr_block
}

# Output firewall rules
output "firewall_rules" {
  value = data.anypoint_private_space_config.example.firewall_rules
}
