###############################################################################
# Outputs
###############################################################################

output "bridge_id" {
  description = "MCP bridge instance ID"
  value       = anypoint_mcp_bridge.petstore_from_spec.id
}

output "bridge_status" {
  description = "MCP bridge status"
  value       = anypoint_mcp_bridge.petstore_from_spec.status
}

output "bridge_asset_id" {
  description = "Generated Exchange asset ID"
  value       = anypoint_mcp_bridge.petstore_from_spec.asset_id
}

output "tool_count" {
  description = "Number of tools parsed from the Exchange spec"
  value       = length(data.anypoint_mcp_tools.petstore.tools)
}
