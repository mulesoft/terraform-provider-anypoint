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
  default = ["10.0.0.192/26"]
}

variable "private_space_id" {
  description = "The ID of the private space."
  type = string
}

variable "organization_id" {
  description = "The ID of the organization."
  type = string
}