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

variable "organization_id" {
  description = "Organization ID"
  type        = string
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
  default     = "<env_id>"
}

variable "gateway_id" {
  description = "Flex / Self-Managed Gateway UUID. Used if you uncomment the resource stub; generate-config-out fills it from the live instance."
  type        = string
  default     = "<gateway_id>"
}

variable "mcp_bridge_id" {
  description = "Numeric MCP bridge instance ID from API Manager (the last segment of the import id)"
  type        = string
  default     = "<mcp_bridge_id>"
}
