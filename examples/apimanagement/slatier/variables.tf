# ─────────────────────────────────────────────────────────────
# Provider Configuration Variables
# ─────────────────────────────────────────────────────────────

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

# ─────────────────────────────────────────────────────────────
# Resource Configuration Variables
# ─────────────────────────────────────────────────────────────

variable "organization_id" {
  description = "Organization ID where the API instance exists"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Environment ID where the API instance is deployed"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
  # Example: "5f8f9a0e-1234-5678-90ab-cdef12345678"
}

variable "api_instance_id" {
  description = "API Instance ID to create SLA tiers for (numeric ID)"
  type        = string
  default     = "4659463"
  # Example: "12345678"
}
