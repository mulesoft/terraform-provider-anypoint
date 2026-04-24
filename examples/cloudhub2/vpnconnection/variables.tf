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

# variable "private_space_id" {
#   description = "The ID of the private space where the VPN connection will be created"
#   type        = string
# }

variable "organization_id" {
  description = "The ID of the organization where the VPN connection will be created"
  type        = string
}

variable "private_space_name" {
  description = "The name of the private space where the VPN connection will be created"
  type        = string
}

variable "region_id" {
    description = "The region id."
    type = string
    default = "us-east-2"
} 

variable "cidr_block" {
  description = "The CIDR block for the private network."
  type = string
  default = "10.0.0.0/18"
}

# variable "reserved_cidrs" {
#   description = "The reserved CIDRs for the private network."
#   type = list(string)
#   default = ["10.0.0.128/25"]
# } 

# variable "vpn_connection_id" {
#   description = "The ID of the VPN connection to fetch (for data source)"
#   type        = string
# }

variable "connection_name" {
  description = "The name of the VPN connection"
  type        = string
  default     = "example-vpn-connection"
}

variable "local_asn" {
  description = "Local ASN for the VPN connection"
  type        = string
  default     = "64512"
}

variable "remote_asn" {
  description = "Remote ASN for the VPN connection"
  type        = string
  default     = "65001"
}

variable "remote_ip_address" {
  description = "Remote IP address for the VPN connection"
  type        = string
  default     = "204.12.238.216"
}

variable "psk_1" {
  description = "Pre-shared key for the first VPN tunnel"
  type        = string
  default     = ""  
}

variable "psk_2" {
  description = "Pre-shared key for the second VPN tunnel"
  type        = string
  default     = ""
}

variable "ptp_cidr_1" {
  description = "Point-to-point CIDR for the first VPN tunnel"
  type        = string
  default     = ""
}

variable "ptp_cidr_2" {
  description = "Point-to-point CIDR for the second VPN tunnel"
  type        = string
  default     = ""
}

variable "startup_action" {
  description = "Startup action for the VPN tunnels"
  type        = string
  default     = "start"
}