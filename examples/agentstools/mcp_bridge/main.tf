###############################################################################
# Anypoint MCP Bridge Example
# ===========================
# An MCP *bridge* turns one or more existing REST APIs into an MCP server
# without writing any MCP server code. For each source API you declare the
# tools (REST operations) to expose; the provider:
#   1. generates and publishes an Exchange asset (mcp-metadata.json),
#   2. creates the Flex / Self-Managed Gateway instance with one route per
#      source API (matched by the X-UPSTREAM-NAME header),
#   3. attaches the MCP transcoding policies that map MCP tool calls to the
#      underlying REST calls.
#
# This differs from anypoint_mcp_server, where YOU supply an existing MCP
# server spec asset. A bridge GENERATES that asset from your tool declarations.
#
# Usage:
#   terraform init
#   terraform plan
#   terraform apply
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

###############################################################################
# MCP Bridge - Petstore
# ---------------------
# Exposes a single source REST API (a petstore) as an MCP server. Three REST
# operations become three MCP tools. Path parameters ({petId}) automatically
# become required tool inputs; POST bodies are exposed via has_body = true.
###############################################################################

resource "anypoint_mcp_bridge" "petstore" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "petstore-mcp-bridge"
  port           = 8081
  base_path      = "petstore"

  source_apis = [
    {
      label        = "petstore-api"
      upstream_uri = "https://sandbox.example.com/petstore/v1"
      asset_id     = "petstore-rest-api"
      version      = "1.0.0"

      tools = [
        {
          method       = "GET"
          path         = "/pets"
          description  = "List all pets"
          query_params = ["limit", "offset"]
        },
        {
          method      = "POST"
          path        = "/pets"
          description = "Create a new pet"
          has_body    = true
        },
        {
          method      = "GET"
          path        = "/pets/{petId}"
          description = "Fetch a single pet by ID"
        },
      ]
    },
  ]
}

###############################################################################
# MCP Bridge - Multiple Source APIs
# ---------------------------------
# A bridge can front several REST APIs at once. Each source_api becomes its own
# route + upstream + transcoding policy. Labels must be unique within a bridge.
###############################################################################

resource "anypoint_mcp_bridge" "commerce" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "commerce-mcp-bridge"
  port           = 8082
  base_path      = "commerce"

  source_apis = [
    {
      label        = "orders-api"
      upstream_uri = "https://orders.internal:8080"
      asset_id     = "orders-rest-api"
      version      = "1.0.0"

      tools = [
        {
          method = "GET"
          path   = "/orders/{orderId}"
        },
        {
          method   = "POST"
          path     = "/orders"
          has_body = true
        },
      ]
    },
    {
      label        = "inventory-api"
      upstream_uri = "https://inventory.internal:8080"
      asset_id     = "inventory-rest-api"
      version      = "2.0.0"

      tools = [
        {
          method        = "GET"
          path          = "/inventory/{sku}"
          name          = "check_stock"
          description   = "Check available stock for a SKU"
          header_params = ["X-Warehouse-Id"]
        },
      ]
    },
  ]
}
