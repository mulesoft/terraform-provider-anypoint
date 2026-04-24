variable "anypoint_client_id" {
  description = "Anypoint Platform client ID"
  type        = string
  sensitive   = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "private_space_id" {
  description = "The ID of the private space to configure"
  type        = string
}

variable "private_space_id_custom_org" {
  description = "The ID of the private space in a custom organization"
  type        = string  
}

variable "custom_organization_id" {
  description = "Custom organization ID for multi-org scenarios"
  type        = string  
}