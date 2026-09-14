###############################################################################
# Anypoint MCP Bridge — explicit tools
# =====================================
# An MCP bridge turns one or more existing REST APIs into an MCP server
# without writing MCP server code. This example declares tools explicitly.
#
# For tools parsed automatically from an Exchange REST API spec, see ../auto_tools/
#
# Managed Flex Gateway note:
#   Managed Flex typically exposes only ports 8081 and 8082. Port + base_path
#   must be unique on the gateway. Run one bridge example at a time on the
#   same managed gateway, or use distinct ports/paths.
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

###############################################################################
# MCP Bridge - Petstore (explicit tools)
# --------------------------------------
# Path parameters ({petId}) become required tool inputs; POST bodies use
# has_body = true.
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
