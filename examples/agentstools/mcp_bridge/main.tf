###############################################################################
# Anypoint MCP Bridge Example
# ===========================
# An MCP bridge turns one or more existing REST APIs into an MCP server
# without writing MCP server code. For each source API you declare the tools
# (REST operations) to expose; the provider publishes a generated Exchange
# asset, creates the gateway instance, and attaches the MCP transcoding
# policies.
#
# Differs from anypoint_mcp_server, where you supply an existing MCP server
# spec asset. A bridge generates that asset from your tool declarations.
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
# MCP Bridge - Petstore (explicit tools)
# --------------------------------------
# One REST API exposed as an MCP server. Path parameters ({petId}) become
# required tool inputs; POST bodies use has_body = true.
###############################################################################

resource "anypoint_mcp_bridge" "petstore" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "petstore-mcp-bridge"
  port           = 8081
  base_path      = "petstore"

  instance_label    = "Petstore MCP"
  approval_method   = "manual"
  consumer_endpoint = var.consumer_endpoint != "" ? var.consumer_endpoint : null

  source_apis = [
    {
      label        = "petstore-api"
      upstream_uri = var.petstore_upstream_uri
      asset_id     = var.petstore_asset_id
      version      = var.petstore_asset_version
      group_id     = var.organization_id

      # Outbound TLS as "<secretGroupId>/<tlsContextId>". Omit for plain http.
      # tls_context_id = "<secretGroupId>/<tlsContextId>"

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

        # Optional: replace the derived string-only input schema.
        {
          method       = "GET"
          path         = "/pets/findByStatus"
          name         = "find_pets_by_status"
          description  = "Find pets by status"
          query_params = ["status", "limit"]

          input_schema = jsonencode({
            type = "object"
            properties = {
              status = {
                type = "string"
                enum = ["available", "pending", "sold"]
              }
              limit = {
                type = "integer"
              }
            }
            required = ["status"]
          })
        },

        # Optional: map tool inputs to different upstream names.
        {
          method        = "PUT"
          path          = "/pets/{petId}"
          name          = "update_pet"
          description   = "Update an existing pet"
          has_body      = true
          query_params  = ["dry_run"]
          header_params = ["api_key"]

          http_mapping = {
            query_params = [
              { key = "dryRun", value = "#[vars.params['dry_run']]" },
            ]
            uri_params = [
              { key = "petId", value = "#[vars.params['petId']]" },
            ]
            headers = [
              { key = "X-Api-Key", value = "#[vars.params['api_key']]" },
            ]
            body = "#[vars.params.body]"
          }
        },
      ]
    },
  ]
}

###############################################################################
# MCP Bridge - Multiple Source APIs
# ---------------------------------
# Each source_api becomes its own route and upstream. Labels must be unique
# within the bridge. Uses a different port from the petstore bridge above.
###############################################################################

resource "anypoint_mcp_bridge" "commerce" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "commerce-mcp-bridge"
  port           = 8082
  base_path      = "commerce"

  instance_label = "Commerce MCP"

  source_apis = [
    {
      label        = "orders-api"
      upstream_uri = "https://orders.internal:8080"
      asset_id     = "orders-rest-api"
      version      = "1.0.0"
      group_id     = var.organization_id

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
      group_id     = var.organization_id

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

###############################################################################
# MCP Bridge - Tools from Exchange spec
# -------------------------------------
# Parse an Exchange REST API spec into tools, then assign them to the bridge.
###############################################################################

data "anypoint_mcp_tools" "petstore" {
  organization_id = var.organization_id
  group_id        = var.organization_id
  asset_id        = var.petstore_asset_id
  version         = var.petstore_asset_version

  exclude_methods = ["DELETE"]
}

resource "anypoint_mcp_bridge" "petstore_auto" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = var.gateway_id

  mcp_asset_name = "petstore-auto-mcp-bridge"
  port           = 8083
  base_path      = "petstore-auto"

  instance_label = "Petstore MCP (auto-parsed tools)"

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
