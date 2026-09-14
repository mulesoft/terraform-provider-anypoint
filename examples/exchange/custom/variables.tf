variable "anypoint_client_id" {
  description = "Anypoint Platform connected-app client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform connected-app client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "org_id" {
  description = "The organization id. Also used as the asset group id."
  type        = string
  default     = "<org_id>"
}
