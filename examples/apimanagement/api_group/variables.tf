# ── Provider credentials ────────────────────────────────────────────────────

variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App Client ID"
  type        = string
  sensitive   = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform Base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

# ── Organisation & Environments ─────────────────────────────────────────────

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Primary environment ID (e.g. Sandbox)"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

variable "staging_environment_id" {
  description = "Secondary environment ID used for the multi-version group example"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

# ── API Instances ────────────────────────────────────────────────────────────

variable "api_instance_ids" {
  description = "Numeric IDs of the API instances to include in group version v1"
  type        = list(number)
  default     = [4675482, 4675479]
}

variable "api_instance_ids_v2" {
  description = "Numeric IDs of the API instances to include in group version v2 (primary env)"
  type        = list(number)
  default     = [4675479]
}

variable "staging_api_instance_ids" {
  description = "Numeric IDs of the API instances in the staging environment"
  type        = list(number)
  default     = [4675479]
}
