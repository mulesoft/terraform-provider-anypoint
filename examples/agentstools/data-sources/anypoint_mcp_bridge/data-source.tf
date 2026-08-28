###############################################################################
# Data Sources: anypoint_mcp_bridges (list) + anypoint_mcp_bridge (single)
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
    }
  }
}

# List every MCP bridge in the environment (plain MCP servers are filtered out).
data "anypoint_mcp_bridges" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "mcp_bridge_ids" {
  description = "IDs of all MCP bridges in the environment"
  value       = [for b in data.anypoint_mcp_bridges.all.bridges : b.id]
}

# Fetch a single MCP bridge by ID, including its source APIs and reconstructed tools.
data "anypoint_mcp_bridge" "example" {
  id              = var.mcp_bridge_id
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "mcp_bridge_status" {
  description = "Current status of the MCP bridge"
  value       = data.anypoint_mcp_bridge.example.status
}

output "mcp_bridge_gateway_id" {
  description = "The Flex Gateway the bridge is deployed to"
  value       = data.anypoint_mcp_bridge.example.gateway_id
}

output "mcp_bridge_source_tools" {
  description = "Each source API and the MCP tools exposed for it"
  value = [
    for s in data.anypoint_mcp_bridge.example.source_apis : {
      label    = s.label
      asset_id = s.asset_id
      version  = s.version
      tools    = [for t in s.tools : t.name]
    }
  ]
}
