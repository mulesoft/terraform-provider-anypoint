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
  default     = "https://stgx.anypoint.mulesoft.com"
}

# ── Provider credentials (User auth for org/env management) ──────────────────

variable "anypoint_admin_client_id" {
  description = "Anypoint Connected App Client ID (must support password grant)"
  type        = string
  sensitive   = true
  default     = "<anypoint_admin_client_id>"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_admin_client_secret>"
}

variable "anypoint_username" {
  description = "Anypoint Platform username (for admin provider)"
  type        = string
  sensitive   = true
  default     = "<admin_username_here>"
}

variable "anypoint_password" {
  description = "Anypoint Platform password (for admin provider)"
  type        = string
  sensitive   = true
  default     = "<admin_password_here>"
}

# ── Organisation ─────────────────────────────────────────────────────────────

variable "parent_organization_id" {
  description = "Parent organization ID under which to create the sub-org"
  type        = string
  default     = "<org_id>"
}

variable "owner_id" {
  description = "User ID of the organization owner"
  type        = string
  default     = "f7f43384-b33e-470c-ad4c-285aa0c01212"
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
  default     = "<private_space_id>"
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
