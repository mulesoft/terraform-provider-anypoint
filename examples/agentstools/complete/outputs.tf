###############################################################################
# Outputs
###############################################################################

# ── MCP Servers ──────────────────────────────────────────────────────────────

output "atlassian_mcp_id" {
  description = "Atlassian MCP server ID"
  value       = anypoint_mcp_server.atlassian.id
}

output "atlassian_mcp_status" {
  description = "Atlassian MCP server status"
  value       = anypoint_mcp_server.atlassian.status
}

output "salesforce_mcp_id" {
  description = "Salesforce MCP server ID"
  value       = anypoint_mcp_server.salesforce.id
}

output "salesforce_mcp_status" {
  description = "Salesforce MCP server status"
  value       = anypoint_mcp_server.salesforce.status
}

output "database_mcp_id" {
  description = "Database MCP server ID"
  value       = anypoint_mcp_server.database.id
}

output "database_mcp_status" {
  description = "Database MCP server status"
  value       = anypoint_mcp_server.database.status
}

# ── Agent Instances ──────────────────────────────────────────────────────────

output "customer_support_agent_id" {
  description = "Customer support agent ID"
  value       = anypoint_agent_instance.customer_support.id
}

output "customer_support_agent_status" {
  description = "Customer support agent status"
  value       = anypoint_agent_instance.customer_support.status
}

output "sales_agent_id" {
  description = "Sales agent ID"
  value       = anypoint_agent_instance.sales.id
}

output "sales_agent_status" {
  description = "Sales agent status"
  value       = anypoint_agent_instance.sales.status
}

output "analytics_agent_id" {
  description = "Analytics agent ID"
  value       = anypoint_agent_instance.analytics.id
}

output "analytics_agent_status" {
  description = "Analytics agent status"
  value       = anypoint_agent_instance.analytics.status
}

# ── Data Source Outputs ──────────────────────────────────────────────────────

output "total_agent_instances" {
  description = "Total number of agent instances deployed"
  value       = length(data.anypoint_agent_instances.all.instances)
}

output "total_mcp_servers" {
  description = "Total number of MCP servers deployed"
  value       = length(data.anypoint_mcp_servers.all.servers)
}

output "all_agent_instance_ids" {
  description = "List of all agent instance IDs"
  value       = [for inst in data.anypoint_agent_instances.all.instances : inst.id]
}

output "all_mcp_server_ids" {
  description = "List of all MCP server IDs"
  value       = [for server in data.anypoint_mcp_servers.all.servers : server.id]
}
