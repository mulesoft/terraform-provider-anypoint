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

variable "connected_app_id" {
  description = "The ID of the connected application to manage scopes for"
  type        = string
  default     = "27ad947a731840b1bf3a03b1efb2d72a"
}

variable "target_organization_id" {
  description = "The organization ID to grant scopes for"
  type        = string
  default     = "<org_id>"
}