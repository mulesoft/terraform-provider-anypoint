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

variable "org_id" {
  description = "The organization id."
  type        = string
  default     = "<org_id>"
}

variable "parent_team_id" {
  description = "The id of the parent team. Use the organization id to create a root team."
  type        = string
  default     = "c63f78eb-39c8-4fb2-80df-09f885c480e0"
} 