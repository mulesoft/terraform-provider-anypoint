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

variable "organization_id" {
  description = "Anypoint organization ID"
  type        = string
  default     = "542cc7e3-2143-40ce-90e9-cf69da9b4da6"
}

variable "environment_id" {
  description = "Target environment ID (e.g. Sandbox or Production)"
  type        = string
  default     = "c0c9f7f5-57bb-4333-82d7-dbdcab912234"
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
