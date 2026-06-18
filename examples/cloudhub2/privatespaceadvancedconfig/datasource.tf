# Retrieve advanced configuration for a private space
data "anypoint_privatespace_advanced_config" "example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id
}

# Output ingress configuration
output "ingress_protocol" {
  value = data.anypoint_privatespace_advanced_config.example.ingress_configuration.protocol
}

output "read_response_timeout" {
  value = data.anypoint_privatespace_advanced_config.example.ingress_configuration.read_response_timeout
}

output "enable_iam_role" {
  value = data.anypoint_privatespace_advanced_config.example.enable_iam_role
}

# Output log filters
output "log_filters" {
  value = data.anypoint_privatespace_advanced_config.example.ingress_configuration.logs.filters
}
