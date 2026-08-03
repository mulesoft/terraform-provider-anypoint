###############################################################################
# MCP Bridge - Auto-derived tools (DS-hybrid / "Approach D")
# ---------------------------------------------------------
# Instead of hand-declaring every tool, use the anypoint_mcp_tools DATA SOURCE
# to parse the source REST API's Exchange spec (OpenAPI/Swagger or RAML) into a
# ready-to-use tool list, then assign it straight to the source API's `tools`.
#
# The parser is read-only: a spec that cannot be parsed fails `plan` cleanly
# instead of half-building a bridge. Use exclude_methods / exclude_tool_names to
# trim the auto-derived set. To rename or re-describe individual tools, fall back
# to declaring them explicitly (see main.tf).
###############################################################################

data "anypoint_mcp_tools" "petstore" {
  organization_id = var.organization_id
  asset_id        = "petstore-rest-api"
  version         = "1.0.0"

  # Optional: drop operations you don't want exposed as tools.
  exclude_methods = ["DELETE"]
}

resource "anypoint_mcp_bridge" "petstore_auto" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "petstore-auto-mcp-bridge"
  port           = 8083
  base_path      = "petstore-auto"

  source_apis = [
    {
      label        = "petstore-api"
      upstream_uri = "https://sandbox.example.com/petstore/v1"
      asset_id     = "petstore-rest-api"
      version      = "1.0.0"

      # Every parsed operation becomes a tool. No hand-written tool blocks.
      tools = data.anypoint_mcp_tools.petstore.tools
    },
  ]
}

output "petstore_auto_spec_type" {
  value = data.anypoint_mcp_tools.petstore.spec_type
}

output "petstore_auto_tool_count" {
  value = length(data.anypoint_mcp_tools.petstore.tools)
}
