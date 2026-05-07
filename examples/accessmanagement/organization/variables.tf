# Anypoint Platform Configuration
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

variable "organization_id" {
  description = "Parent Organization ID (Salesforce org)"
  type        = string
  default     = "<org_id>"
}

variable "owner_user_id" {
  description = "Owner User ID for the sub-organization (must be an existing user)"
  type        = string
  default     = "f7f43384-b33e-470c-ad4c-285aa0c01212"
}

variable "sub_org_name" {
  description = "Name of the sub-organization to create"
  type        = string
  default     = "terraform-suborg-example-renamed-1234"
}

variable "parent_organization_id" {
  description = "ID of the parent organization"
  type        = string
  default     = "<org_id>"

  validation {
    condition     = can(regex("^[0-9a-f-]+$", var.parent_organization_id))
    error_message = "Parent organization ID must be a valid UUID format."
  }
}