###############################################################################
# Variables
###############################################################################

# ── Provider credentials ─────────────────────────────────────────────────────

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
  default     = "https://stgx.anypoint.mulesoft.com"
}

# ── Organization & Environment ───────────────────────────────────────────────

variable "organization_id" {
  description = "Organization ID"
  type        = string
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
  default     = "<private_space_id>"
}

# ── Gateway ──────────────────────────────────────────────────────────────────

variable "gateway_id" {
  description = "Omni Gateway ID for agent deployment"
  type        = string
  default     = "b123b2eb-35aa-454c-9750-dff9e2d218c9"
}

# ── Agent Asset ──────────────────────────────────────────────────────────────

variable "agent_asset_id" {
  description = "Exchange asset ID for the agent specification"
  type        = string
  default     = "ExampleBedrockA2AAgent"
}

variable "agent_asset_version" {
  description = "Exchange asset version"
  type        = string
  default     = "1.0.0"
}
