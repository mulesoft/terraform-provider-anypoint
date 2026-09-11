###############################################################################
# Outputs
###############################################################################

output "petstore_bridge_id" {
  description = "Petstore MCP bridge instance ID"
  value       = anypoint_mcp_bridge.petstore.id
}

output "petstore_bridge_status" {
  description = "Petstore MCP bridge status"
  value       = anypoint_mcp_bridge.petstore.status
}

output "petstore_bridge_asset_id" {
  description = "Generated Exchange asset ID"
  value       = anypoint_mcp_bridge.petstore.asset_id
}

output "commerce_bridge_id" {
  description = "Commerce MCP bridge instance ID"
  value       = anypoint_mcp_bridge.commerce.id
}

output "petstore_auto_bridge_id" {
  description = "Auto-parsed petstore MCP bridge instance ID"
  value       = anypoint_mcp_bridge.petstore_auto.id
}

output "petstore_auto_tool_count" {
  description = "Number of tools parsed from the Exchange spec"
  value       = length(data.anypoint_mcp_tools.petstore.tools)
}
