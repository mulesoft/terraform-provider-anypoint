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

variable "connected_app_id" {
  description = "The ID of the connected application to manage scopes for"
  type        = string
  default     = "27ad947a731840b1bf3a03b1efb2d72a"
}

variable "target_organization_id" {
  description = "The organization ID to grant scopes for"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}