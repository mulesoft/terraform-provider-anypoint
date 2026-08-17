###############################################################################
# Variables
###############################################################################

variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App client ID"
  type        = string
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform Connected App client secret"
  type        = string
  sensitive   = true
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "The organization ID that owns the Private Space"
  type        = string
}

variable "private_space_id" {
  description = "The ID of the Private Space to attach the Transit Gateway to"
  type        = string
}

variable "name" {
  description = "Name of the transit gateway attachment"
  type        = string
  default     = "tf-test-tgw"
}

variable "resource_share_id" {
  description = "AWS RAM resource share ID in UUID format (e.g. e8e330a8-4f8c-452b-afd0-7810c41287f1)"
  type        = string
}

variable "resource_share_account" {
  description = "AWS account ID that owns the Transit Gateway"
  type        = string
}

variable "routes" {
  description = "CIDR routes for the connection. Required attribute, but may be empty ([]). Must not overlap the Private Space CIDR."
  type        = list(string)
  default     = ["192.168.1.0/24", "172.16.0.0/12"]
}
