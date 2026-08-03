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

# ── Organization & Environment ───────────────────────────────────────────────

variable "organization_id" {
  description = "Organization ID"
  type        = string
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
  default     = "<env_id>"
}

# ── Gateway ──────────────────────────────────────────────────────────────────

variable "gateway_id" {
  description = "Flex / Self-Managed Gateway UUID to deploy the MCP bridge to"
  type        = string
  default     = "<gateway_id>"
}
