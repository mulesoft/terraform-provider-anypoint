variable "anypoint_client_id" {
  description = "Anypoint Connected App Client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Anypoint Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
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