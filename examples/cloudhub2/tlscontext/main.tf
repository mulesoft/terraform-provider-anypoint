terraform {
  required_providers {
    anypoint = {
      source = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}

provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  base_url      = var.anypoint_base_url
}

locals {
  certificate_file = var.certificate_file != null ? var.certificate_file : "${path.module}/../../certs/cert_anypoint_tls_context.pem"
  key_file         = var.key_file != null ? var.key_file : "${path.module}/../../certs/key_anypoint_tls_context.pem"
}

# Example 1: TLS Context with PEM keystore
resource "anypoint_tls_context" "pem_example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id
  name             = "example-pem-tls-context"
  keystore_type    = "PEM"

  # Read certificate and private key from files so sensitive material
  # is never embedded in the Terraform configuration.
  certificate = file(local.certificate_file)
  key         = file(local.key_file)

  key_filename         = "key.pem"
  certificate_filename = "cert.pem"
  
  # Cipher configuration
  ciphers = {
    aes128_gcm_sha256                = true
    aes128_sha256                    = false
    aes256_gcm_sha384                = false
    aes256_sha256                    = false
    dhe_rsa_aes128_sha256            = false
    dhe_rsa_aes256_gcm_sha384        = false
    dhe_rsa_aes256_sha256            = false
    ecdhe_ecdsa_aes128_gcm_sha256    = true
    ecdhe_ecdsa_aes256_gcm_sha384    = true
    ecdhe_rsa_aes128_gcm_sha256      = true
    ecdhe_rsa_aes256_gcm_sha384      = true
    ecdhe_ecdsa_chacha20_poly1305    = false
    ecdhe_rsa_chacha20_poly1305      = false
    dhe_rsa_chacha20_poly1305        = false
    tls_aes256_gcm_sha384            = true
    tls_chacha20_poly1305_sha256     = true
    tls_aes128_gcm_sha256            = true
  }
}

# Example 2: TLS Context with JKS keystore
# resource "anypoint_tls_context" "jks_example" {
#   private_space_id = "beaa792a-7fff-4865-9d36-d7e28ebbc04d"
#   name             = "example-jks-tls-context"
#   keystore_type    = "JKS"
  
#   # JKS-specific configuration
#   keystore_base64     = "var.jks_keystore_base64"
#   store_passphrase    = "var.jks_store_passphrase"
#   key_passphrase      = "var.jks_key_passphrase"
#   alias               = "var.jks_alias"
#   keystore_filename   = "keystore.jks"
  
#   # Cipher configuration (more restrictive)
#   ciphers = {
#     aes128_gcm_sha256                = false
#     aes128_sha256                    = false
#     aes256_gcm_sha384                = true
#     aes256_sha256                    = false
#     dhe_rsa_aes128_sha256            = false
#     dhe_rsa_aes256_gcm_sha384        = false
#     dhe_rsa_aes256_sha256            = false
#     ecdhe_ecdsa_aes128_gcm_sha256    = false
#     ecdhe_ecdsa_aes256_gcm_sha384    = true
#     ecdhe_rsa_aes128_gcm_sha256      = false
#     ecdhe_rsa_aes256_gcm_sha384      = true
#     ecdhe_ecdsa_chacha20_poly1305    = false
#     ecdhe_rsa_chacha20_poly1305      = false
#     dhe_rsa_chacha20_poly1305        = false
#     tls_aes256_gcm_sha384            = true
#     tls_chacha20_poly1305_sha256     = false
#     tls_aes128_gcm_sha256            = false
#   }
# }

# Output TLS Context information
output "pem_tls_context" {
  value = {
    id           = anypoint_tls_context.pem_example.id
    name         = anypoint_tls_context.pem_example.name
    type         = anypoint_tls_context.pem_example.type
    trust_store  = anypoint_tls_context.pem_example.trust_store
    key_store    = anypoint_tls_context.pem_example.key_store
  }
}

# output "jks_tls_context" {
#   value = {
#     id           = anypoint_tls_context.jks_example.id
#     name         = anypoint_tls_context.jks_example.name
#     type         = anypoint_tls_context.jks_example.type
#     trust_store  = anypoint_tls_context.jks_example.trust_store
#     key_store    = anypoint_tls_context.jks_example.key_store
#   }
# }

# # Example showing how to use the TLS context data source
# data "anypoint_tls_context" "existing" {
#   private_space_id = var.private_space_id
#   id               = var.existing_tls_context_id
# }

# output "existing_tls_context_details" {
#   value = data.anypoint_tls_context.existing
# }

# Example showing certificate expiration monitoring
# locals {
#   expiring_certificates = [
#     for ctx in [
#       anypoint_tls_context.pem_example,
#       anypoint_tls_context.jks_example
#     ] : ctx if ctx.key_store != null && ctx.key_store.expiration_date != null
#   ]
# }

# output "certificate_expiration_summary" {
#   value = {
#     for ctx in local.expiring_certificates : ctx.name => {
#       id              = ctx.id
#       expiration_date = ctx.key_store.expiration_date
#       common_name     = ctx.key_store.cn
#       san             = ctx.key_store.san
#     }
#   }
# }

# # Example showing cipher configuration summary
# output "cipher_configuration_summary" {
#   value = {
#     pem_context = {
#       name           = anypoint_tls_context.pem_example.name
#       enabled_ciphers = [
#         for cipher, enabled in anypoint_tls_context.pem_example.ciphers : cipher
#         if enabled
#       ]
#     }
#     jks_context = {
#       name           = anypoint_tls_context.jks_example.name
#       enabled_ciphers = [
#         for cipher, enabled in anypoint_tls_context.jks_example.ciphers : cipher
#         if enabled
#       ]
#     }
#   }
# } 