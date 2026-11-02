variable "anypoint_admin_client_id" {
  type        = string
  description = "Client ID of an on-behalf-of-user connected app."
}

variable "anypoint_admin_client_secret" {
  type        = string
  sensitive   = true
  description = "Client secret of the on-behalf-of-user connected app."
}

variable "anypoint_admin_username" {
  type        = string
  description = "Username of an Org Administrator of the parent organization."
}

variable "anypoint_admin_password" {
  type        = string
  sensitive   = true
  description = "Password of the Org Administrator user."
}

variable "anypoint_base_url" {
  type        = string
  description = "Anypoint control plane base URL (e.g. https://anypoint.mulesoft.com or https://eu1.anypoint.mulesoft.com)."
  default     = "https://anypoint.mulesoft.com"
}

variable "organization_name" {
  type        = string
  description = "Name of the new organization (business group) to create."
}

variable "parent_organization_id" {
  type        = string
  description = "ID of the parent organization under which the new business group is created."
}

variable "owner_user_id" {
  type        = string
  description = "User ID of the owner of the new organization."
}
