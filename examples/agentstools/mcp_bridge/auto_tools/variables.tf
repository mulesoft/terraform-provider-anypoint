###############################################################################
# Variables
###############################################################################

variable "anypoint_client_id" {
  description = "Connected App client ID"
  type        = string
  sensitive   = true
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Organization ID"
  type        = string
}

variable "environment_id" {
  description = "Environment ID"
  type        = string
}

variable "gateway_id" {
  description = "Flex / Self-Managed Gateway UUID"
  type        = string
}

variable "petstore_asset_id" {
  description = "Exchange asset ID of the petstore REST API"
  type        = string
}

variable "petstore_asset_version" {
  description = "Exchange asset version of the petstore REST API"
  type        = string
  default     = "1.0.0"
}

variable "petstore_upstream_uri" {
  description = "Backend the petstore bridge forwards tool calls to"
  type        = string
  default     = "https://sandbox.example.com/petstore/v1"
}
