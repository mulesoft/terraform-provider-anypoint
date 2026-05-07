terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

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
  description = "Anypoint control-plane URL"
  type        = string
  default     = "https://stgx.anypoint.mulesoft.com"
}

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "<org_id>"
}

variable "environment_id" {
  description = "Target environment ID (e.g. Sandbox or Production)"
  type        = string
  default     = "<private_space_id>"
}

# ─── Secret Group ────────────────────────────────────────────────

resource "anypoint_secret_group" "main" {
  environment_id = var.environment_id
  name           = "terraform-tls-123"
  downloadable   = true
}

# ─── Keystore (PEM cert + key) ──────────────────────────────────

resource "anypoint_secret_group_keystore" "tls" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-keystore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../../certs/cert2.pem"))
  key_base64         = base64encode(file("${path.module}/../../certs/key2.pem"))
}

resource "anypoint_secret_group_keystore" "tls_1" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-keystore-1"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../../certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/../../certs/key.pem"))
}

# ─── Truststore (CA certificate) ────────────────────────────────

resource "anypoint_secret_group_truststore" "ca" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "ca-truststore-renamed-123"
  type            = "PEM"

  truststore_base64 = base64encode(file("${path.module}/../../certs/truststore2.pem"))
}

# ─── TLS Context ────────────────────────────────────────────────
# References keystore and truststore by their IDs — the provider
# automatically builds "keystores/{id}" and "truststores/{id}" paths.

resource "anypoint_secret_group_tls_context" "omni" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "omni-tls-context-123"

  keystore_id   = anypoint_secret_group_keystore.tls.id
  truststore_id = anypoint_secret_group_truststore.ca.id

  min_tls_version = "TLSv1.3"
  max_tls_version = "TLSv1.3"
  alpn_protocols  = ["h2", "http/1.1"]

  enable_client_cert_validation = false
  skip_server_cert_validation   = false
}

# ─── TLS Context (mTLS enabled) ─────────────────────────────────

resource "anypoint_secret_group_tls_context" "mtls" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "mtls-context-123"

  keystore_id   = anypoint_secret_group_keystore.tls_1.id
  truststore_id = anypoint_secret_group_truststore.ca.id

  min_tls_version = "TLSv1.3"
  max_tls_version = "TLSv1.3"
  alpn_protocols  = ["h2", "http/1.1"]

  enable_client_cert_validation = true
  skip_server_cert_validation   = false
}

# ─── Outputs ─────────────────────────────────────────────────────

output "tls_context_id" {
  value = anypoint_secret_group_tls_context.omni.id
}

output "mtls_context_id" {
  value = anypoint_secret_group_tls_context.mtls.id
}
