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
  name           = "terraform-keystores"
  downloadable   = false
}

# ─── PEM Keystore ────────────────────────────────────────────────
# For PEM files (text), wrap file() with base64encode() to produce the required base64 input.

resource "anypoint_secret_group_keystore" "pem" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-pem-keystore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../../certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/../../certs/key.pem"))
}

# ─── PEM Keystore with CA Chain ──────────────────────────────────

resource "anypoint_secret_group_keystore" "pem_with_ca" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "tls-pem-with-truststore"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../../certs/cert.pem"))
  key_base64         = base64encode(file("${path.module}/../../certs/key.pem"))
  ca_path_base64     = base64encode(file("${path.module}/../../certs/truststore.pem"))
}

resource "anypoint_secret_group_keystore" "ks_jks" {
  organization_id      = var.organization_id
  environment_id       = var.environment_id
  secret_group_id      = anypoint_secret_group.main.id
  name                 = "sparq-ks-jks"
  type                 = "JKS"
  keystore_file_base64 = filebase64("${path.module}/../../certs/keystore.jks")
  alias                = "sparq"
  key_passphrase           = "Ankit123"
  store_passphrase           = "Ankit123"
}


# ─── JKS Keystore ───────────────────────────────────────────────
# For binary JKS files, use filebase64() which reads and base64-encodes in one step.

# resource "anypoint_secret_group_keystore" "jks" {
#   environment_id  = var.environment_id
#   secret_group_id = anypoint_secret_group.main.id
#   name            = "tls-jks-keystore"
#   type            = "JKS"
#
#   keystore_file_base64 = filebase64("${path.module}/certs/keystore.jks")
#   passphrase           = var.jks_passphrase
#   alias                = "myalias"
# }

# ─── PKCS12 Keystore ────────────────────────────────────────────
# Same pattern as JKS: filebase64() for binary .p12 files.

# resource "anypoint_secret_group_keystore" "pkcs12" {
#   environment_id  = var.environment_id
#   secret_group_id = anypoint_secret_group.main.id
#   name            = "tls-pkcs12-keystore"
#   type            = "PKCS12"
#
#   keystore_file_base64 = filebase64("${path.module}/certs/keystore.p12")
#   passphrase           = var.p12_passphrase
#   alias                = "myalias"
# }

# ─── Outputs ─────────────────────────────────────────────────────

output "pem_keystore_id" {
  value = anypoint_secret_group_keystore.pem.id
}

output "pem_keystore_expiration" {
  value = anypoint_secret_group_keystore.pem.expiration_date
}
