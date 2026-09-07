###############################################################################
# Anypoint Terraform Provider – MCP Bridge Demo
# ==============================================
# Turn REST APIs you already run into an MCP server that agents can call,
# on a self-managed Flex Gateway that is already deployed:
#
#   Step 1 → Look up the existing self-managed gateway
#   Step 2 → Build a bridge with tools declared explicitly (Approach A)
#   Step 3 → Build a bridge with tools parsed from the spec (Approach D)
#   Step 4 → Govern the bridge with an SLA tier and a policy
#   Step 5 → Read the bridges back
#   Step 6 → Adopt an existing bridge and its generated asset
#
# Usage:
#   terraform init
#   terraform plan       ← preview what will be created
#   terraform apply      ← provision everything
#   terraform destroy    ← tear it all down
#
# ── One bridge per port ──────────────────────────────────────────────────────
# A bridge binds a port and base path on its gateway, and a root base path is a
# catch-all, so two bridges cannot share a port. A self-managed gateway lets you
# expose as many ports as you like, so the two bridges below sit on one gateway
# at 8081 and 8082. (A shared-space managed gateway only exposes 8081, so there
# each bridge needs its own gateway.)
###############################################################################

terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
    }
  }
}

# Connected App (client credentials). The app needs Manage Servers, Read Servers
# and View Organization for the gateway pre-flight, the API Manager scopes
# (Manage APIs Configuration, Manage Policies, Deploy API Proxies) and Exchange
# contributor rights to publish the generated asset.
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

###############################################################################
# Step 1 – The existing self-managed gateway
# ---------------------------------------
# Nothing is created here. The gateway is already registered and running; we
# only read it so the bridges can reference it and so a broken gateway shows up
# as a plan-time failure rather than a confusing apply error.
#
# The gateway must be connected and serving before apply. A disconnected gateway
# returns GatewayNotReadyError; the provider retries briefly, but one that never
# comes up fails the create.
###############################################################################

data "anypoint_self_managed_gateway" "flex" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  id              = var.self_managed_gateway_id
}

###############################################################################
# Step 2 – Bridge with explicit tools (Approach A)
# ---------------------------------------
# Declare exactly which REST operations become tools, and what each one looks
# like to the agent. Use this when you want control over the tool surface: the
# names, the contracts, and how each call reaches your backend.
#
# The provider publishes a generated Exchange asset describing the tools,
# creates the gateway instance with one route per source API, and attaches the
# MCP transcoding policies that turn tool calls into REST calls.
###############################################################################

resource "anypoint_mcp_bridge" "explicit" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = data.anypoint_self_managed_gateway.flex.id

  # Becomes the generated asset's name and (sanitized) ID, and the instance name
  # in API Manager. Changing it replaces the bridge.
  mcp_asset_name = var.explicit_bridge_name

  # The proxy URI is http://0.0.0.0:<port>/<base_path>. Both force replacement.
  # A leading slash on base_path is not significant: "mcp" and "/mcp" match.
  port      = var.explicit_bridge_port
  base_path = "mcp"

  # ── Optional instance settings — all applied in place ──────────────────────

  # Shown in API Manager. Worth setting when several instances share one asset.
  instance_label = "Orders MCP (explicit tools)"

  # "manual" holds client access requests for review instead of auto-approving.
  # On its own it does not block traffic — pair it with a policy enforcing
  # client credentials (anypoint_api_policy_client_id_enforcement) to actually
  # restrict access to approved applications.
  approval_method = "manual"

  # The public URL clients use to reach the bridge. It is NOT derived
  # automatically: leave it unset and API Manager shows an empty consumer
  # endpoint. Point it at however your Flex Gateway is published.
  consumer_endpoint = "${var.gateway_public_url}/mcp"

  # External client provider for application authentication. Leave null to use
  # Anypoint's built-in provider, which the UI shows as "Anypoint".
  provider_id = var.client_provider_id != "" ? var.client_provider_id : null

  source_apis = [
    {
      # The route label, and the X-UPSTREAM-NAME header value that selects it.
      # Unique within the bridge. Changing it replaces the bridge.
      label        = "orders"
      upstream_uri = var.orders_upstream_uri
      asset_id     = var.orders_asset_id
      version      = var.orders_asset_version
      group_id     = var.organization_id

      # TLS for the outbound hop to your backend, as
      # "<secretGroupId>/<tlsContextId>". Set it when the upstream is https and
      # the gateway must trust the backend certificate, or for mutual TLS.
      # Structural: changing it replaces the bridge.
      tls_context_id = var.orders_tls_context_id != "" ? var.orders_tls_context_id : null

      tools = [
        # ── 1. The minimum ────────────────────────────────────────────────────
        # Name defaults to <method>_<slug(path)>, so this is "get_orders", and
        # the description defaults to the name.
        {
          method = "GET"
          path   = "/orders"
        },

        # ── 2. Path and query parameters ──────────────────────────────────────
        # Path parameters become required tool inputs automatically; query
        # parameters become optional ones.
        {
          method       = "GET"
          path         = "/orders/{orderId}"
          name         = "get_order"
          description  = "Fetch a single order by its identifier."
          query_params = ["include_lines"]
        },

        # ── 3. A request body and a header ────────────────────────────────────
        {
          method        = "POST"
          path          = "/orders"
          name          = "create_order"
          description   = "Place a new order."
          has_body      = true
          header_params = ["X-Request-Id"]
        },

        # ── 4. A hand-written input schema ────────────────────────────────────
        # By default the schema is derived from the parameters above with every
        # property typed "string". Supply input_schema when the agent needs a
        # real type, an enum, or per-property guidance. It is published exactly
        # as written — not merged with the derived schema — so it must describe
        # every input the tool accepts.
        #
        # Editing it republishes the generated asset and advances asset_version.
        # It is not recovered by terraform import: it lives only in the asset
        # metadata, which import does not read.
        {
          method       = "GET"
          path         = "/orders/search"
          name         = "search_orders"
          description  = "Search orders by status and size."
          query_params = ["status", "limit"]

          input_schema = jsonencode({
            type = "object"
            properties = {
              status = {
                type        = "string"
                description = "Only return orders in this state."
                enum        = ["pending", "shipped", "cancelled"]
              }
              limit = {
                type        = "integer"
                description = "Maximum number of orders to return."
              }
            }
            required = ["status"]
          })
        },

        # ── 5. A hand-written HTTP mapping ────────────────────────────────────
        # By default each parameter goes upstream under its own name as
        # #[vars.params['<name>']], and path placeholders come from the path.
        # Set http_mapping when the backend expects a different name, a computed
        # value, or a particular body.
        #
        # Each section is independent — omit one and it stays derived — so
        # changing only the body does not mean restating every parameter. An
        # empty list is meaningful and sends none of that kind.
        #
        # Editing it re-syncs the transcoding policy but does NOT advance
        # asset_version, and it IS recovered by terraform import.
        {
          method        = "PUT"
          path          = "/orders/{orderId}"
          name          = "update_order"
          description   = "Update an existing order."
          has_body      = true
          query_params  = ["dry_run"]
          header_params = ["api_key"]

          http_mapping = {
            # The agent says "dry_run"; the backend expects "dryRun".
            query_params = [
              { key = "dryRun", value = "#[vars.params['dry_run']]" },
            ]

            # Derived from the path when omitted; spelled out here for clarity.
            uri_params = [
              { key = "orderId", value = "#[vars.params['orderId']]" },
            ]

            # Feed a backend header from a differently-named tool input.
            headers = [
              { key = "X-Api-Key", value = "#[vars.params['api_key']]" },
            ]

            # Defaults to #[vars.params.body] when has_body is true.
            # Set to "" to send no body at all.
            body = "#[vars.params.body]"
          }
        },
      ]
    },
  ]
}

###############################################################################
# Step 3 – Bridge with auto-parsed tools (Approach D)
# ---------------------------------------
# Instead of hand-declaring operations, let the data source read the API's
# Exchange spec and produce the tool list. It returns the same object shape the
# resource takes, so it can be assigned straight through.
#
# Parsing stays out of the write path: a spec the provider cannot read fails at
# plan, rather than half-building a bridge.
#
# input_schema and http_mapping are absent here — neither can be derived from a
# spec. To mix the two approaches, concat() hand-written tools onto the parsed
# list.
###############################################################################

data "anypoint_mcp_tools" "inventory" {
  organization_id = var.organization_id
  group_id        = var.organization_id
  asset_id        = var.inventory_asset_id
  version         = var.inventory_asset_version

  # Trim a large spec down to the operations worth exposing.
  # exclude_methods    = ["DELETE"]
  # exclude_tool_names = ["get_health"]
}

resource "anypoint_mcp_bridge" "autoparse" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  gateway_id      = data.anypoint_self_managed_gateway.flex.id

  mcp_asset_name = var.autoparse_bridge_name

  # A different port from the bridge in Step 2 — see the note in the header.
  port      = var.autoparse_bridge_port
  base_path = "mcp"

  instance_label    = "Inventory MCP (parsed from spec)"
  consumer_endpoint = "${var.gateway_public_url}:${var.autoparse_bridge_port}/mcp"

  source_apis = [
    {
      label        = "inventory"
      upstream_uri = var.inventory_upstream_uri
      asset_id     = var.inventory_asset_id
      version      = var.inventory_asset_version
      group_id     = var.organization_id

      # The whole point of Approach D.
      tools = data.anypoint_mcp_tools.inventory.tools
    },
  ]
}

###############################################################################
# Step 4 – Governance on the bridge
# ---------------------------------------
# A bridge is an API Manager instance, so everything that attaches to an
# instance works here too: SLA tiers, policies, contracts.
###############################################################################

resource "anypoint_api_instance_sla_tier" "gold" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_bridge.explicit.id

  name         = "gold"
  description  = "Partner tier for the orders MCP bridge."
  auto_approve = false

  limits = [
    {
      maximum_requests            = 1000
      time_period_in_milliseconds = 60000
      visible                     = true
    },
  ]
}

resource "anypoint_api_policy_rate_limiting" "orders_rate_limit" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_mcp_bridge.explicit.id

  configuration = {
    rate_limits = [
      {
        maximum_requests            = 100
        time_period_in_milliseconds = 60000
      }
    ]
    expose_headers = false
    clusterizable  = true
  }
}

###############################################################################
# Step 5 – Reading the bridges back
# ---------------------------------------
# Both data sources are read-only and do not depend on the gateway being up, so
# they still work when a bridge is inactive or the Flex container is down.
###############################################################################

data "anypoint_mcp_bridge" "orders" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  id              = anypoint_mcp_bridge.explicit.id
}

data "anypoint_mcp_bridges" "all" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
}

###############################################################################
# Step 6 – Adopting what already exists
# ---------------------------------------
# Uncomment, fill in the numeric bridge ID from the API Manager URL, then:
#
#   terraform plan -generate-config-out=generated.tf
#
# Import always produces Approach A, however the bridge was built: the platform
# stores only the materialized tools, with no record that they came from a spec
# parse. A tool on the default request mapping returns in the concise
# query_params / header_params form; one with a custom mapping returns with an
# explicit http_mapping block. Tool description and input_schema come back null
# — adding them back is a REAL change, not cosmetic: the next plan updates the
# tools in place and republishes the asset at a new version. Expected, and the
# plan is clean afterwards.
#
# Destroying a bridge removes its instance and policies but deliberately leaves
# the generated Exchange asset published, so clients still on that version keep
# working. Import the asset as well if you want Terraform to own its lifecycle;
# note the version is pinned, so a later tool edit moves the bridge on while
# this resource keeps tracking the version you adopted.
###############################################################################

# import {
#   to = anypoint_mcp_bridge.adopted
#   id = "${var.organization_id}/${var.environment_id}/<mcp_bridge_id>"
# }

# import {
#   to = anypoint_exchange_asset.adopted_asset
#   id = "${var.organization_id}/${var.explicit_bridge_name}/1.0.0"
# }
