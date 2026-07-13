terraform {
  required_providers {
    anypoint = {
      source = "mulesoft/anypoint"
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
  name           = "terraform-shared-secrets"
  downloadable   = false
}

# ─── UsernamePassword ────────────────────────────────────────────

resource "anypoint_secret_group_shared_secret" "db_creds" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "db-credentials"
  type            = "UsernamePassword"

  username = "admin"
  password = var.db_password
}

variable "db_password" {
  type      = string
  sensitive = true
}

# ─── S3Credential ───────────────────────────────────────────────

resource "anypoint_secret_group_shared_secret" "s3" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "s3-backup-creds"
  type            = "S3Credential"

  access_key_id      = var.aws_access_key
  secret_access_key  = var.aws_secret_key
  expiration_date    = "2026-12-31"
}

variable "aws_access_key" {
  type      = string
  sensitive = true
}

variable "aws_secret_key" {
  type      = string
  sensitive = true
}

# ─── SymmetricKey ───────────────────────────────────────────────

resource "anypoint_secret_group_shared_secret" "symmetric" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "encryption-key"
  type            = "SymmetricKey"

  key = base64encode("my-256-bit-secret-key-value-here")
}

# ─── Blob ───────────────────────────────────────────────────────

resource "anypoint_secret_group_shared_secret" "blob" {
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.main.id
  name            = "api-token"
  type            = "Blob"

  content = var.api_token
}

variable "api_token" {
  type      = string
  sensitive = true
}

# ─── Outputs ─────────────────────────────────────────────────────

output "db_creds_id" {
  value = anypoint_secret_group_shared_secret.db_creds.id
}

output "s3_creds_id" {
  value = anypoint_secret_group_shared_secret.s3.id
}

output "symmetric_key_id" {
  value = anypoint_secret_group_shared_secret.symmetric.id
}

output "blob_id" {
  value = anypoint_secret_group_shared_secret.blob.id
}
