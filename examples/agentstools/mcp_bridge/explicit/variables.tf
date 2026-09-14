###############################################################################
# Variables
###############################################################################

# ── Provider credentials (Connected App) ─────────────────────────────────────

variable "anypoint_client_id" {
  description = "Connected App client ID"
  type        = string
  sensitive   = true
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
}

variable "anypoint_base_url" {
  description = "Anypoint control-plane URL (use https://stgx.anypoint.mulesoft.com for STGX)"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

# ── Organization & Environment ───────────────────────────────────────────────

variable "organization_id" {
  description = "Organization ID"
  type        = string
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
}

# ── Gateway ──────────────────────────────────────────────────────────────────

variable "gateway_id" {
  description = "Flex / Self-Managed Gateway UUID"
  type        = string
}

# ── Optional instance settings ───────────────────────────────────────────────

variable "consumer_endpoint" {
  description = "Public URL clients use to reach the petstore bridge. Leave empty to omit."
  type        = string
  default     = ""
}

# ── Petstore source API ──────────────────────────────────────────────────────

variable "petstore_asset_id" {
  description = "Exchange asset ID of the petstore REST API"
  type        = string
  default     = "tf-demo-rest-api"
}

variable "petstore_asset_version" {
  description = "Exchange asset version of the petstore REST API"
  type        = string
  default     = "1.0.0"
}

variable "petstore_upstream_uri" {
  description = "Backend the petstore bridge forwards tool calls to"
  type        = string
  default     = "https://sandbox.example.com/petstore/v1"
}
