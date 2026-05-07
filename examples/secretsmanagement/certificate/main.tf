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
