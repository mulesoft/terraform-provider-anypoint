# Example: Reading known policy data sources
# Each known policy type has its own dedicated data source with typed configuration.

# Read a rate-limiting policy applied to an API instance.
# When policy_id is omitted, the data source auto-discovers the policy by type.
data "anypoint_api_policy_rate_limiting" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "rate_limiting_config" {
  value = data.anypoint_api_policy_rate_limiting.example.configuration
}

# Read a CORS policy by explicit policy_id
data "anypoint_api_policy_cors" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
  policy_id       = "789"
}

output "cors_public_resource" {
  value = data.anypoint_api_policy_cors.example.configuration.public_resource
}

# Read a JWT validation policy
data "anypoint_api_policy_jwt_validation" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "jwt_signing_method" {
  value = data.anypoint_api_policy_jwt_validation.example.configuration.signing_method
}

# Read an IP allowlist policy
data "anypoint_api_policy_ip_allowlist" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "allowed_ips" {
  value = data.anypoint_api_policy_ip_allowlist.example.configuration.ips
}

# Read a spike-control policy
data "anypoint_api_policy_spike_control" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "spike_max_requests" {
  value = data.anypoint_api_policy_spike_control.example.configuration.maximum_requests
}

# Read a client-id-enforcement policy
data "anypoint_api_policy_client_id_enforcement" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

# Read an outbound policy (message-logging-outbound)
data "anypoint_api_policy_message_logging_outbound" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

# Read a circuit-breaker outbound policy
data "anypoint_api_policy_circuit_breaker" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

# Read an MCP policy
data "anypoint_api_policy_mcp_pii_detector" "example" {
  environment_id  = var.environment_id
  api_instance_id = var.api_instance_id
}

output "mcp_pii_entities" {
  value = data.anypoint_api_policy_mcp_pii_detector.example.configuration.entities
}

variable "environment_id" {
  type = string
}

variable "api_instance_id" {
  type = string
}
