# Anypoint Platform Configuration
variable "anypoint_client_id" {
  description = "Anypoint Platform client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
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
  default     = "<org_id>"
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
