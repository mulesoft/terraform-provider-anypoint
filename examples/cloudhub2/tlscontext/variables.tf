# Anypoint Platform Configuration
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

# Private Space Configuration
variable "private_space_id" {
  description = "The ID of the private space where the TLS context will be created"
  type        = string
  default     = "8469150e-4af2-425b-af48-8df0e5c9e285"
}

variable "organization_id" {
  description = "The ID of the organization"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

# Certificate / Key file paths
variable "certificate_file" {
  description = "Path to the PEM certificate file"
  type        = string
  default     = null
}

variable "key_file" {
  description = "Path to the PEM private key file"
  type        = string
  default     = null
}
