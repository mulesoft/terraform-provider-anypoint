variable "anypoint_client_id" {
  description = "Anypoint Connected App Client ID"
  type        = string
  sensitive   = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform Base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "region" {
  description = "AWS region for private space"
  type        = string
  default     = "us-east-1"
}

variable "target_organization_id" {
  description = "Target organization ID for explicit organization specification"
  type        = string
}