###############################################################################
# Variables
###############################################################################

# ── Provider credentials ────────────────────────────────────────────────────

variable "anypoint_client_id" {
  description = "Connected App client ID"
  type        = string
  sensitive   = true
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Connected App client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint control-plane URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

# ── Organisation & Environment ──────────────────────────────────────────────

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Source environment ID (e.g. Sandbox or Production)"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

variable "target_environment_id" {
  description = "Environment ID to promote the API instance into (e.g. Production)"
  type        = string
  default     = "448ec638-4283-40e3-ba3a-d1db2b63e02d"
}

# ── Private Space ───────────────────────────────────────────────────────────

variable "private_space_name" {
  description = "Name for the private space (used as prefix for child resources)"
  type        = string
  default     = "ps-1"
}

variable "region" {
  description = "AWS region for the private space (e.g. us-east-1, us-east-2)"
  type        = string
  default     = "us-east-2"
}

variable "network_cidr_block" {
  description = "CIDR block for the private network"
  type        = string
  default     = "10.0.0.0/20"
}

# ── VPN ─────────────────────────────────────────────────────────────────────

variable "vpn_local_asn" {
  description = "Local (Anypoint side) BGP ASN"
  type        = string
  default     = "64512"
}

variable "vpn_remote_asn" {
  description = "Remote (on-prem) BGP ASN"
  type        = string
  default     = "65001"
}

variable "vpn_remote_ip" {
  description = "Public IP of the on-prem VPN device"
  type        = string
  default     = "203.0.113.42"
}

variable "vpn_tunnel_1_psk" {
  description = "Pre-shared key for VPN tunnel 1"
  type        = string
  sensitive   = true
  default     = ""
}

variable "vpn_tunnel_1_ptp_cidr" {
  description = "Point-to-point CIDR for VPN tunnel 1 (e.g. 169.254.10.0/30)"
  type        = string
  default     = ""
}

variable "vpn_tunnel_2_psk" {
  description = "Pre-shared key for VPN tunnel 2"
  type        = string
  sensitive   = true
  default     = ""
}

variable "vpn_tunnel_2_ptp_cidr" {
  description = "Point-to-point CIDR for VPN tunnel 2 (e.g. 169.254.11.0/30)"
  type        = string
  default     = ""
}

# ── Flex Gateway ────────────────────────────────────────────────────────────

variable "gateway_size" {
  description = "Flex Gateway replica size (small, large)"
  type        = string
  default     = "small"
}

variable "gateway_runtime_version" {
  description = "Gateway version for API instance deployment"
  type        = string
  default     = "1.9.9"
}

# ── API Instance ────────────────────────────────────────────────────────────

variable "api_asset_id" {
  description = "Exchange asset ID for the API specification"
  type        = string
  default     = "api-test"
}

variable "api_asset_version" {
  description = "Exchange asset version"
  type        = string
  default     = "1.0.0"
}

variable "api_base_path" {
  description = "Base path for the API instance (appended to http://0.0.0.0:8081/)"
  type        = string
  default     = "basePath"
}

variable "upstream_primary_uri" {
  description = "Primary backend URI"
  type        = string
  default     = "http://backend-primary.internal:8080"
}

variable "upstream_secondary_uri" {
  description = "Secondary backend URI (for canary / blue-green)"
  type        = string
  default     = "http://backend-secondary.internal:8080"
}

variable "upstream_primary_weight" {
  description = "Traffic weight (0-100) sent to the primary upstream"
  type        = number
  default     = 90
}

# ── API Policies ────────────────────────────────────────────────────────────

variable "mulesoft_policy_group_id" {
  description = "Exchange group ID for MuleSoft-provided policies"
  type        = string
  default     = "68ef9520-24e9-4cf2-b2f5-620025690913"
}

# ── Outbound Policies ────────────────────────────────────────────────────────

variable "upstream_id" {
  description = "Routing-upstream UUID for outbound policies. Find this in API Manager under the API instance routing configuration, or via GET /xapi/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/upstreams"
  type        = string
  default     = ""
}

variable "oauth_client_secret" {
  description = "OAuth 2.0 client secret used by credential-injection and in-task authorization outbound policies"
  type        = string
  sensitive   = true
  default     = "eyshsjsshhshsss.shjsyyehe"
}

variable "upstream_basic_auth_password" {
  description = "Password for the upstream Basic Auth credential-injection outbound policy"
  type        = string
  sensitive   = true
  default     = ""
}

variable "aws_access_key_id" {
  description = "AWS access key ID for the native-aws-lambda outbound policy"
  type        = string
  sensitive   = true
  default     = "sadadadsa"
}

variable "aws_secret_access_key" {
  description = "AWS secret access key for the native-aws-lambda outbound policy"
  type        = string
  sensitive   = true
  default     = "asdasdadas"
}

# ── Alerts ───────────────────────────────────────────────────────────────────

variable "alert_email" {
  description = "Email address to receive API alerts"
  type        = string
  default     = "admin@example.com"
}
