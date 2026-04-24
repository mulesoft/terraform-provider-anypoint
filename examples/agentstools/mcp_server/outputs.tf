###############################################################################
# Outputs
###############################################################################

output "atlassian_mcp_id" {
  description = "Atlassian MCP server instance ID"
  value       = anypoint_mcp_server.atlassian_mcp.id
}

output "atlassian_mcp_status" {
  description = "Atlassian MCP server instance status"
  value       = anypoint_mcp_server.atlassian_mcp.status
}

output "atlassian_mcp_base_path" {
  description = "Atlassian MCP server base path"
  value       = anypoint_mcp_server.atlassian_mcp.endpoint.base_path
}

output "salesforce_mcp_id" {
  description = "Salesforce MCP server instance ID"
  value       = anypoint_mcp_server.salesforce_mcp.id
}

output "salesforce_mcp_status" {
  description = "Salesforce MCP server instance status"
  value       = anypoint_mcp_server.salesforce_mcp.status
}

output "enterprise_tools_mcp_id" {
  description = "Enterprise tools MCP server instance ID"
  value       = anypoint_mcp_server.enterprise_tools_mcp.id
}

output "enterprise_tools_mcp_status" {
  description = "Enterprise tools MCP server instance status"
  value       = anypoint_mcp_server.enterprise_tools_mcp.status
}
