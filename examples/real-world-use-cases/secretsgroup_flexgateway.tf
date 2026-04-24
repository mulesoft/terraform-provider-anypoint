terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}


resource "anypoint_secret_group" "main" {
  environment_id = var.environment_id
  name           = "real-world-example-secrets"
  downloadable   = false
}

resource "anypoint_secret_group_keystore" "tls" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-keystore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/../certs/key.pem"))
}

resource "anypoint_secret_group_truststore" "ca" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "ca-truststore"
  type            = "PEM"

  truststore_base64 = base64encode(file("${path.module}/../certs/truststore.pem"))
}

resource "anypoint_flex_tls_context" "flex" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "flex-tls-context"

  keystore_id   = anypoint_secret_group_keystore.tls.id
  truststore_id = anypoint_secret_group_truststore.ca.id

  alpn_protocols = ["h2", "http/1.1"]
}

# --------------------------------------------------------------------------
# Fully configured Managed Flex Gateway with explicit version
# --------------------------------------------------------------------------
resource "anypoint_managed_flexgateway" "complete" {
  name            = "real-world-example-gateway"  
  environment_id  = var.environment_id
  target_id       = var.target_id
  ingress = {
    public_url = "https://example.mulesoft.com/"
    forward_ssl_session = true
    last_mile_security  = true
  }
}

# --------------------------------------------------------------------------
# Variables
# --------------------------------------------------------------------------
variable "anypoint_client_id" {
  description = "Anypoint Platform Connected App client ID"
  type        = string
  default     = "e5a776d9862a4f2d8f61ba8450803908"
}

variable "anypoint_client_secret" {
  description = "Anypoint Platform Connected App client secret"
  type        = string
  sensitive   = true
  default     = "0a5E1fbfc1154D9885c32842171F7490"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "environment_id" {
  description = "The environment ID where the gateway will be deployed"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
}

variable "target_id" {
  description = "The private space / target ID for the gateway deployment"
  type        = string
  default     = "8469150e-4af2-425b-af48-8df0e5c9e285"
}

# --------------------------------------------------------------------------
# Outputs
# --------------------------------------------------------------------------

output "complete_gateway_id" {
  description = "The ID of the fully configured managed Flex Gateway"
  value       = anypoint_managed_flexgateway.complete.id
}

output "complete_gateway_public_url" {
  description = "The public URL of the fully configured gateway"
  value       = anypoint_managed_flexgateway.complete.ingress.public_url
}

output "complete_gateway_internal_url" {
  description = "The internal URL of the fully configured gateway"
  value       = anypoint_managed_flexgateway.complete.ingress.internal_url
}

output "complete_gateway_status" {
  description = "The current status of the fully configured gateway"
  value       = anypoint_managed_flexgateway.complete.status
}
