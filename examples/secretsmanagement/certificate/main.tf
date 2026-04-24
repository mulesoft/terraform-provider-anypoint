terraform {
  required_providers {
    anypoint = {
      source = "sf.com/mulesoft/anypoint"
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
  name           = "terraform-certificates-1"
  downloadable   = false
}

# ─── Certificate (PEM) ──────────────────────────────────────────

resource "anypoint_secret_group_certificate" "pem" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "cert1"
  type            = "PEM"

  certificate_base64 = base64encode(file("${path.module}/../../certs/cert.pem"))
}

# ─── Certificate Pinset ─────────────────────────────────────────

resource "anypoint_secret_group_certificate_pinset" "pin" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "certpinset-1"

  certificate_pinset_base64 = base64encode(file("${path.module}/../../certs/cert.pem"))
}

# ─── Outputs ─────────────────────────────────────────────────────

output "certificate_id" {
  value = anypoint_secret_group_certificate.pem.id
}

output "certificate_expiration" {
  value = anypoint_secret_group_certificate.pem.expiration_date
}

output "pinset_id" {
  value = anypoint_secret_group_certificate_pinset.pin.id
}
