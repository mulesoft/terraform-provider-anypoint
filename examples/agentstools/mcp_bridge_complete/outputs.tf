###############################################################################
# Outputs
###############################################################################

output "explicit_bridge_id" {
  description = "Numeric ID of the explicitly-declared bridge, as it appears in the API Manager URL"
  value       = anypoint_mcp_bridge.explicit.id
}

output "explicit_bridge_asset" {
  description = "Exchange asset the explicit bridge generated. The version advances on every tool edit."
  value       = "${anypoint_mcp_bridge.explicit.asset_id}/${anypoint_mcp_bridge.explicit.asset_version}"
}

output "explicit_bridge_status" {
  description = <<-EOT
    Bridge status. Often "unregistered" straight after create, until the gateway
    picks the deployment up — that is expected, not a failure.
  EOT
  value       = anypoint_mcp_bridge.explicit.status
}

output "explicit_bridge_settings" {
  description = "The optional instance settings, read back from the platform rather than echoed from config"
  value = {
    instance_label    = data.anypoint_mcp_bridge.orders.instance_label
    approval_method   = data.anypoint_mcp_bridge.orders.approval_method
    consumer_endpoint = data.anypoint_mcp_bridge.orders.consumer_endpoint
    provider_id       = data.anypoint_mcp_bridge.orders.provider_id
  }
}

output "explicit_bridge_tools" {
  description = <<-EOT
    Tools as the platform stores them. http_mapping is populated only for tools
    whose mapping differs from the default; input_schema is absent because it
    lives in the generated asset metadata, not on the gateway.
  EOT
  value       = data.anypoint_mcp_bridge.orders.source_apis
}

output "autoparse_bridge_id" {
  description = "Numeric ID of the spec-parsed bridge"
  value       = anypoint_mcp_bridge.autoparse.id
}

output "autoparsed_tools" {
  description = "What the data source parsed out of the inventory API spec, before it reached the bridge"
  value       = data.anypoint_mcp_tools.inventory.tools
}

output "sla_tier_id" {
  description = "SLA tier attached to the orders bridge"
  value       = anypoint_api_instance_sla_tier.gold.id
}

output "rate_limit_policy_id" {
  description = "Rate-limiting policy attached to the orders bridge"
  value       = anypoint_api_policy_rate_limiting.orders_rate_limit.id
}

output "all_bridges" {
  description = "Every MCP bridge in the environment, including any created outside Terraform"
  value = [for b in data.anypoint_mcp_bridges.all.bridges : {
    id              = b.id
    asset_id        = b.asset_id
    instance_label  = b.instance_label
    approval_method = b.approval_method
    status          = b.status
  }]
}
