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
