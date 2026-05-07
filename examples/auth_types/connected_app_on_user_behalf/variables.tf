variable "anypoint_admin_client_id" {
  description = "Anypoint Connected App Client ID (must support password grant)"
  type        = string
  sensitive   = true
  default     = "<anypoint_admin_client_id>"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_admin_client_secret>"
}

variable "anypoint_admin_username" {
  description = "Anypoint Platform Username"
  type        = string
  sensitive   = true
  default     = "<admin_username_here>"
}

variable "anypoint_admin_password" {
  description = "Anypoint Platform Password"
  type        = string
  sensitive   = true
  default     = "<admin_password_here>"
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
  description = "Target organization ID (must be accessible by the user)"
  type        = string
}