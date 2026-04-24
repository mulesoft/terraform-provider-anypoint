variable "anypoint_admin_client_id" {
  description = "Anypoint Connected App Client ID (must support password grant)"
  type        = string
  sensitive   = true
  default     = "a66da37ba83d4c599264347952d4d533"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "0de4EA9E5bae4651B599a2071bFDD4E1"
}

variable "anypoint_admin_username" {
  description = "Anypoint Platform Username"
  type        = string
  sensitive   = true
  default     = "ankitsarda_anypointstgx"
}

variable "anypoint_admin_password" {
  description = "Anypoint Platform Password"
  type        = string
  sensitive   = true
  default     = "Dreamz@007"
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