###############################################################################
# Variables
###############################################################################

# ── Provider credentials (Connected App) ─────────────────────────────────────

variable "anypoint_client_id" {
  description = "Connected App client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
}

variable "anypoint_base_url" {
  description = "Anypoint control-plane URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

# ── Organisation & environment ───────────────────────────────────────────────

variable "organization_id" {
  description = "Organization (business group) ID that owns the gateway and assets"
  type        = string
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Environment ID the bridges are created in"
  type        = string
  default     = "<environment_id>"
}

# ── Self-managed gateway (already deployed) ──────────────────────────────────

variable "self_managed_gateway_id" {
  description = "ID of the existing self-managed Flex Gateway to deploy the bridges to"
  type        = string
  default     = "<self_managed_gateway_id>"
}

variable "gateway_public_url" {
  description = <<-EOT
    How consumers reach this Flex Gateway, without a trailing slash — used to
    build each bridge's consumer_endpoint. The platform does not derive this for
    a self-managed gateway, so set it to whatever fronts your Flex deployment
    (ingress, load balancer, or DNS name).
  EOT
  type        = string
  default     = "https://flex.example.com"
}

variable "client_provider_id" {
  description = <<-EOT
    Optional external client provider used to authenticate applications
    requesting access to a bridge. Leave empty to use Anypoint's built-in
    provider, which API Manager shows as "Anypoint".
  EOT
  type        = string
  default     = ""
}

# ── Bridge: explicit tools (Approach A) ──────────────────────────────────────

variable "explicit_bridge_name" {
  description = "MCP asset name for the explicitly-declared bridge"
  type        = string
  default     = "orders-mcp-bridge"
}

variable "explicit_bridge_port" {
  description = "Gateway port the explicit bridge listens on"
  type        = number
  default     = 8081
}

variable "orders_asset_id" {
  description = "Exchange asset ID of the orders REST API"
  type        = string
  default     = "orders-api"
}

variable "orders_asset_version" {
  description = "Exchange asset version of the orders REST API"
  type        = string
  default     = "1.0.0"
}

variable "orders_upstream_uri" {
  description = "Backend the orders bridge forwards tool calls to"
  type        = string
  default     = "https://orders.internal.example.com"
}

variable "orders_tls_context_id" {
  description = <<-EOT
    Optional TLS context for the orders upstream, as
    "<secretGroupId>/<tlsContextId>". Set it when the upstream is https and the
    gateway must trust the backend certificate, or for mutual TLS. Leave empty
    for a plain http backend.

    Structural: changing it replaces the bridge, because it lives on the
    upstream and routing is only accepted at create time.
  EOT
  type        = string
  default     = ""
}

# ── Bridge: auto-parsed tools (Approach D) ───────────────────────────────────

variable "autoparse_bridge_name" {
  description = "MCP asset name for the spec-parsed bridge"
  type        = string
  default     = "inventory-mcp-bridge"
}

variable "autoparse_bridge_port" {
  description = <<-EOT
    Gateway port the auto-parsed bridge listens on. Must differ from
    explicit_bridge_port: a bridge binds a port and base path, and a root base
    path is a catch-all, so two bridges cannot share a port.
  EOT
  type        = number
  default     = 8082
}

variable "inventory_asset_id" {
  description = "Exchange asset ID of the inventory REST API whose spec is parsed into tools"
  type        = string
  default     = "inventory-api"
}

variable "inventory_asset_version" {
  description = "Exchange asset version of the inventory REST API"
  type        = string
  default     = "1.0.0"
}

variable "inventory_upstream_uri" {
  description = "Backend the inventory bridge forwards tool calls to"
  type        = string
  default     = "https://inventory.internal.example.com"
}
