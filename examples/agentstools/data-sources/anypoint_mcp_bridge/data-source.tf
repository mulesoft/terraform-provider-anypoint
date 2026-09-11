###############################################################################
# Data Sources: anypoint_mcp_bridges (list) + anypoint_mcp_bridge (single)
###############################################################################

data "anypoint_mcp_bridges" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "mcp_bridges" {
  description = "MCP bridges in the environment"
  value = [
    for b in data.anypoint_mcp_bridges.all.bridges : {
      id              = b.id
      asset_id        = b.asset_id
      instance_label  = b.instance_label
      approval_method = b.approval_method
      endpoint_uri    = b.endpoint_uri
      status          = b.status
    }
  ]
}

data "anypoint_mcp_bridge" "example" {
  id              = var.mcp_bridge_id
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

output "mcp_bridge" {
  description = "Single MCP bridge details"
  value = {
    instance_label    = data.anypoint_mcp_bridge.example.instance_label
    approval_method   = data.anypoint_mcp_bridge.example.approval_method
    consumer_endpoint = data.anypoint_mcp_bridge.example.consumer_endpoint
    provider_id       = data.anypoint_mcp_bridge.example.provider_id
    proxy_uri         = data.anypoint_mcp_bridge.example.proxy_uri
    status            = data.anypoint_mcp_bridge.example.status
    gateway_id        = data.anypoint_mcp_bridge.example.gateway_id
    source_apis       = data.anypoint_mcp_bridge.example.source_apis
  }
}
