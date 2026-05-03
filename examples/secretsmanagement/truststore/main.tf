terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
    }
  }
}

provider "anypoint" {
  client_id     = var.client_id
  client_secret = var.client_secret
  org_id        = var.org_id
}

variable "client_id" {
  type      = string
  sensitive = true
}

variable "client_secret" {
  type      = string
  sensitive = true
}

variable "org_id" {
  type = string
}

variable "environment_id" {
  type = string
}

# ─── Secret Group ────────────────────────────────────────────────

resource "anypoint_secret_group" "main" {
  environment_id = var.environment_id
  name           = "terraform-truststores"
  downloadable   = false
}

# ─── PEM Truststore ─────────────────────────────────────────────
# For PEM files (text), wrap file() with base64encode().

resource "anypoint_secret_group_truststore" "pem" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "ca-truststore"
  type            = "PEM"

  truststore_base64 = base64encode(file("${path.module}/../../certs/truststore.pem"))
}

# ─── JKS Truststore ─────────────────────────────────────────────
# For binary JKS files, use filebase64() which reads and base64-encodes in one step.

# resource "anypoint_secret_group_truststore" "jks" {
#   environment_id  = var.environment_id
#   secret_group_id = anypoint_secret_group.main.id
#   name            = "ca-truststore-jks"
#   type            = "JKS"
#
#   truststore_base64 = filebase64("${path.module}/certs/truststore.jks")
#   passphrase        = var.jks_passphrase
# }

# ─── PKCS12 Truststore ──────────────────────────────────────────

# resource "anypoint_secret_group_truststore" "pkcs12" {
#   environment_id  = var.environment_id
#   secret_group_id = anypoint_secret_group.main.id
#   name            = "ca-truststore-p12"
#   type            = "PKCS12"
#
#   truststore_base64 = filebase64("${path.module}/certs/truststore.p12")
#   passphrase        = var.p12_passphrase
# }

# ─── Outputs ─────────────────────────────────────────────────────

output "pem_truststore_id" {
  value = anypoint_secret_group_truststore.pem.id
}

output "pem_truststore_expiration" {
  value = anypoint_secret_group_truststore.pem.expiration_date
}
