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

variable "region_id" {
    description = "The region id."
    type = string
    default = "us-east-2"
} 

variable "cidr_block" {
  description = "The CIDR block for the private network."
  type = string
  default = "10.0.0.0/20"
}

variable "reserved_cidrs" {
  description = "The reserved CIDRs for the private network."
  type = list(string)
  default = ["10.1.0.0/26"]
}

variable "private_space_name" {
  description = "Name of the private space"
  type        = string
  default     = "tf-psc-v2"
}

variable "organization_id" {
  description = "Organization ID (defaults to provider organization if omitted)"
  type        = string
  default     = ""
}
