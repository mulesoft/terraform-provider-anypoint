variable "anypoint_client_id" {
  description = "Connected App client ID"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_id>"
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
  default     = "<anypoint_connected_app_client_secret>"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Environment ID where the API instance is deployed"
  type        = string
  default     = "<env_id>"
}

variable "api_instance_id" {
  description = "Numeric API instance ID to look up"
  type        = string
  default     = "12345"
}
