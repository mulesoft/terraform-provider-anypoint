variable "anypoint_client_id" {
  description = "Anypoint Platform client ID"
  type        = string
  sensitive   = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "private_space_id" {
  description = "The ID of the private space where the Transit Gateway will be created"
  type        = string
}

variable "transit_gateway_name" {
  description = "The name of the Transit Gateway"
  type        = string
  default     = "example-transit-gateway"
}

variable "resource_share_id" {
  description = "The resource share ID for the Transit Gateway"
  type        = string
}

variable "resource_share_account" {
  description = "The resource share account for the Transit Gateway"
  type        = string
}

variable "routes" {
  description = "List of route CIDR blocks for the Transit Gateway"
  type        = list(string)
  default     = ["10.0.0.0/16"]
} 