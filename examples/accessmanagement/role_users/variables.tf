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

variable "anypoint_username" {
  description = "Anypoint Platform username"
  type        = string
  sensitive   = true
  default     = "<anypoint_username>"
}

variable "anypoint_password" {
  description = "Anypoint Platform password"
  type        = string
  sensitive   = true
  default     = "<anypoint_password>"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "org_id" {
  description = "The organization ID"
  type        = string
  default     = "<org_id>"
}

variable "role_group_id" {
  description = "The role group ID to assign users to"
  type        = string
  default     = "<role_group_id>"
}

variable "user_id" {
  description = "The user ID to assign to the role group"
  type        = string
  default     = "<user_id>"
}

variable "user_id_2" {
  description = "Second user ID for multiple assignments example"
  type        = string
  default     = "<user_id_2>"
}

variable "search_username" {
  description = "Username to search for in the users datasource example"
  type        = string
  default     = "john.doe"
}
