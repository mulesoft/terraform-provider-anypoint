###############################################################################
# Data Source: anypoint_mcp_server
###############################################################################

# Fetch a single MCP server by ID
data "anypoint_mcp_server" "example" {
  id              = var.mcp_server_id
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

# Output the MCP server details
output "mcp_server_status" {
  description = "Current status of the MCP server"
  value       = data.anypoint_mcp_server.example.status
}

output "mcp_server_consumer_endpoint" {
  description = "Consumer-facing endpoint (proxy URI)"
  value       = data.anypoint_mcp_server.example.consumer_endpoint
}

output "mcp_server_upstream_id" {
  description = "Upstream ID for policy configuration"
  value       = data.anypoint_mcp_server.example.upstream_id
}

output "mcp_server_instance_label" {
  description = "Human-readable label for the MCP server"
  value       = data.anypoint_mcp_server.example.instance_label
}

output "mcp_server_deployment" {
  description = "Deployment configuration"
  value       = data.anypoint_mcp_server.example.deployment
}
