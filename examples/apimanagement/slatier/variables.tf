# ─────────────────────────────────────────────────────────────
# Provider Configuration Variables
# ─────────────────────────────────────────────────────────────

variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App Client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
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
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Environment ID where the API instance is deployed"
  type        = string
  default     = "<private_space_id>"
  # Example: "5f8f9a0e-1234-5678-90ab-cdef12345678"
}

variable "api_instance_id" {
  description = "API Instance ID to create SLA tiers for (numeric ID)"
  type        = string
  default     = "4659463"
  # Example: "12345678"
}
