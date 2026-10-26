###############################################################################
# Anypoint MCP Bridge — auto-created tools
# ========================================
# Parses an Exchange REST API asset into tools via the anypoint_mcp_tools
# data source, then assigns that list to the bridge. You do not declare
# each tool by hand.
#
# For hand-declared tools, see ../explicit/
#
# Managed Flex Gateway note:
#   Managed Flex typically exposes only ports 8081 and 8082. Port + base_path
#   must be unique on the gateway. If explicit/ already used 8081/petstore,
#   this example uses 8082 / petstore-from-spec so both can coexist.
#
# Usage:
#   cp terraform.tfvars.example terraform.tfvars   # fill in values
#   terraform init && terraform plan && terraform apply
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

data "anypoint_mcp_tools" "petstore" {
  organization_id = var.organization_id
  group_id        = var.organization_id
  asset_id        = var.petstore_asset_id
  version         = var.petstore_asset_version

  exclude_methods = ["DELETE"]
}

resource "anypoint_mcp_bridge" "petstore_from_spec" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "petstore-from-spec-mcp-bridge"
  port           = 8082
  base_path      = "petstore-from-spec"

  instance_label = "Petstore MCP (tools from spec)"

  source_apis = [
    {
      label        = "petstore-api"
      upstream_uri = var.petstore_upstream_uri
      asset_id     = var.petstore_asset_id
      version      = var.petstore_asset_version
      group_id     = var.organization_id

      tools = data.anypoint_mcp_tools.petstore.tools
    },
  ]
}
