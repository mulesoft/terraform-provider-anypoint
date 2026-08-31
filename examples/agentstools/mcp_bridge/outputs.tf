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
  description = "Generated Exchange asset ID for the petstore bridge"
  value       = anypoint_mcp_bridge.petstore.asset_id
}

output "petstore_bridge_asset_version" {
  description = "Generated Exchange asset version (bumps on tool updates)"
  value       = anypoint_mcp_bridge.petstore.asset_version
}

output "petstore_bridge_consumer_endpoint" {
  description = "Consumer-facing MCP endpoint URI for the petstore bridge"
  value       = anypoint_mcp_bridge.petstore.consumer_endpoint
}

output "commerce_bridge_id" {
  description = "Commerce MCP bridge instance ID"
  value       = anypoint_mcp_bridge.commerce.id
}

output "commerce_bridge_status" {
  description = "Commerce MCP bridge status"
  value       = anypoint_mcp_bridge.commerce.status
}
