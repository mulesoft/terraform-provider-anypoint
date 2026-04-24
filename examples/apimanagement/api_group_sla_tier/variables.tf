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

# ── Organisation & Environment ───────────────────────────────────────────────

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Environment ID where the API Group instance lives"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

# ── API Group Instance ───────────────────────────────────────────────────────

variable "group_instance_id" {
  description = <<-EOT
    Numeric ID of the API Group instance to attach SLA tiers to.
    Find this value in the API Manager URL when viewing the group instance:
      .../apimanager/…/groupInstances/<ID>/tiers
    Or retrieve it from the anypoint_api_group resource via its versions[*].id attribute.
  EOT
  type        = string
  default     = "565469"
}
