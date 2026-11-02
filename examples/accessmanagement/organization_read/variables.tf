variable "anypoint_client_id" {
  type        = string
  description = "Client ID of a standard (client-credentials) connected app with the profile and read:organization scopes."
}

variable "anypoint_client_secret" {
  type        = string
  sensitive   = true
  description = "Client secret of the connected app."
}

variable "anypoint_base_url" {
  type        = string
  description = "Anypoint control plane base URL (e.g. https://anypoint.mulesoft.com or https://eu1.anypoint.mulesoft.com)."
  default     = "https://anypoint.mulesoft.com"
}

variable "organization_id" {
  type        = string
  description = "ID of the organization (business group) to read."
}
