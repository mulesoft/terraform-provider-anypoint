###############################################################################
# Variables
###############################################################################

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

# ── Provider credentials (User auth for org/env management) ──────────────────

variable "anypoint_admin_client_id" {
  description = "Anypoint Connected App Client ID (must support password grant)"
  type        = string
  sensitive   = true
  default     = "a66da37ba83d4c599264347952d4d533"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "0de4EA9E5bae4651B599a2071bFDD4E1"
}

variable "anypoint_username" {
  description = "Anypoint Platform username (for admin provider)"
  type        = string
  sensitive   = true
  default     = "ankitsarda_anypointstgx"
}

variable "anypoint_password" {
  description = "Anypoint Platform password (for admin provider)"
  type        = string
  sensitive   = true
  default     = "Dreamz@007"
}

# ── Organisation ─────────────────────────────────────────────────────────────

variable "parent_organization_id" {
  description = "Parent organization ID under which to create the sub-org"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "owner_id" {
  description = "User ID of the organization owner"
  type        = string
  default     = "f7f43384-b33e-470c-ad4c-285aa0c01212"
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

variable "target_id" {
  description = "Target ID"
  type        = string
  default     = "675c4efb-d44e-44cd-ac6f-d5a1128e6236"
}

# ── Infrastructure ───────────────────────────────────────────────────────────

variable "region" {
  description = "AWS region for the Private Space"
  type        = string
  default     = "us-east-2"
}

# ── API ──────────────────────────────────────────────────────────────────────

variable "api_asset_id" {
  description = "Exchange asset ID for the API specification"
  type        = string
  default     = "api-test"
}

variable "api_asset_version" {
  description = "Exchange asset version"
  type        = string
  default     = "1.0.0"
}

# ── Alert ────────────────────────────────────────────────────────────────────

variable "alert_email" {
  description = "Email address to receive API alerts"
  type        = string
  default     = "api-ops@example.com"
}

# ── Policies ─────────────────────────────────────────────────────────────────

variable "mulesoft_policy_group_id" {
  description = "Exchange group ID for MuleSoft-provided policies"
  type        = string
  default     = "68ef9520-24e9-4cf2-b2f5-620025690913"
}
