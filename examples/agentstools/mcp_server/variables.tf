###############################################################################
# Variables
###############################################################################

# ── Provider credentials ─────────────────────────────────────────────────────

# ── Provider credentials (Connected App) ─────────────────────────────────────

variable "anypoint_client_id" {
  description = "Connected App client ID"
  type        = string
  sensitive   = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint control-plane URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

# ── Organization & Environment ───────────────────────────────────────────────

variable "organization_id" {
  description = "Organization ID"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

# ── Gateway ──────────────────────────────────────────────────────────────────

variable "gateway_id" {
  description = "Flex Gateway ID for agent deployment"
  type        = string
  default     = "b123b2eb-35aa-454c-9750-dff9e2d218c9"
}

# ── MCP Server Asset ─────────────────────────────────────────────────────────

variable "mcp_asset_id" {
  description = "Exchange asset ID for the MCP server specification"
  type        = string
  default     = "Atlassian-MCP-Server"
}

variable "mcp_asset_version" {
  description = "Exchange asset version"
  type        = string
  default     = "1.0.0"
}
