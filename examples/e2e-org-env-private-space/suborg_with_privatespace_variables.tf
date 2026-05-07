# Variables for Sub-Organization with Private Space Complete Flow

###############################################################################
# Provider Configuration - Admin Credentials
###############################################################################

variable "anypoint_admin_client_id" {
  description = "Anypoint Platform Admin Connected App Client ID"
  type        = string
  default     = "<anypoint_admin_client_id>"
}

variable "anypoint_admin_client_secret" {
  description = "Anypoint Platform Admin Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_admin_client_secret>"
}

variable "anypoint_admin_username" {
  description = "Anypoint Platform Admin Username (for scope assignment)"
  type        = string
  default     = "<admin_username_here>"
}

variable "anypoint_admin_password" {
  description = "Anypoint Platform Admin Password (for scope assignment)"
  type        = string
  sensitive   = true
  default     = "<admin_password_here>"
}

###############################################################################
# Provider Configuration - Normal User Credentials
###############################################################################

variable "anypoint_normal_client_id" {
  description = "Anypoint Platform Normal User Connected App Client ID"
  type        = string
  default     = "<anypoint_connected_app_client_id>"
  # Example: "b77ea48ca94e5f3a9f72ba9561914644"
}

variable "anypoint_normal_client_secret" {
  description = "Anypoint Platform Normal User Connected App Client Secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
  # Example: "1ef5FB0F6cbf5762C600b3182cGEE5F2"
}

###############################################################################
# Common Provider Configuration
###############################################################################

variable "anypoint_base_url" {
  description = "Anypoint Platform Base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

###############################################################################
# Organization Configuration
###############################################################################

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
  default     = "terraform-suborg-new"
}

###############################################################################
# Connected App Configuration
###############################################################################

variable "connected_app_client_id" {
  description = "Client ID of the existing connected app to assign scopes to"
  type        = string
  default     = "<anypoint_connected_app_client_id>"
}

###############################################################################
# Private Space Configuration
###############################################################################

variable "private_space_region" {
  description = "AWS region for the private space"
  type        = string
  default     = "us-east-2"
}

###############################################################################
# Private Network Configuration
###############################################################################

variable "network_cidr_block" {
  description = "CIDR block for the private network (must not overlap with reserved ranges)"
  type        = string
  default     = "10.111.0.0/16"

  validation {
    condition     = can(cidrhost(var.network_cidr_block, 0))
    error_message = "Network CIDR block must be a valid CIDR notation (e.g., 10.111.0.0/16)."
  }
}

variable "network_reserved_cidrs" {
  description = "Reserved CIDR blocks for the private network (for VPN, etc.)"
  type        = list(string)
  default     = []
}
