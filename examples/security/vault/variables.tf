variable "vault_mount" {
  description = "Vault KV v2 mount where the Anypoint admin secret is stored."
  type        = string
  default     = "secret"
}

variable "vault_path" {
  description = <<EOT
Path under the KV v2 mount that holds the Anypoint admin credentials. The
secret data must contain four fields: client_id, client_secret, username,
password.
EOT
  type        = string
  default     = "anypoint/terraform-admin"
}

variable "anypoint_base_url" {
  description = "Anypoint Platform base URL."
  type        = string
  default     = "https://anypoint.mulesoft.com"
}

variable "master_organization_id" {
  description = "UUID of the master organization to look up as a smoke test."
  type        = string
  default     = "<add-your-value-here>"
}
