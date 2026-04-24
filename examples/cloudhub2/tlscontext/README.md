# CloudHub 2.0 TLS Context Resource

This example demonstrates how to create and manage TLS contexts in CloudHub 2.0 using the Anypoint Terraform Provider. TLS contexts are used to configure SSL/TLS certificates for securing connections in your applications.

## Overview

The TLS context resource supports two keystore types:
- **PEM**: Certificate and private key in PEM format
- **JKS**: Java KeyStore format

## Prerequisites

- A CloudHub 2.0 private space
- Valid SSL/TLS certificates
- Terraform installed
- Anypoint Terraform Provider configured

## API Reference

### Create TLS Context
- **Endpoint**: `POST /runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/tlsContexts`
- **Method**: POST

#### Request Body (PEM Format)
```json
{
  "name": "example-tls-context",
  "tlsConfig": {
    "keyStore": {
      "source": "PEM",
      "certificate": "-----BEGIN CERTIFICATE-----\n...\n-----END CERTIFICATE-----",
      "key": "-----BEGIN PRIVATE KEY-----\n...\n-----END PRIVATE KEY-----",
      "keyPassphrase": "passphrase",
      "keyFileName": "server.key",
      "certificateFileName": "server.crt"
    }
  },
  "ciphers": {
    "aes128GcmSha256": true,
    "ecdheEcdsaAes128GcmSha256": true,
    // ... other cipher configurations
  }
}
```

#### Request Body (JKS Format)
```json
{
  "name": "example-tls-context",
  "tlsConfig": {
    "keyStore": {
      "source": "JKS",
      "keystoreBase64": "base64-encoded-jks-data",
      "keyPassphrase": "key-passphrase",
      "storePassphrase": "store-passphrase",
      "alias": "certificate-alias",
      "keystoreFileName": "keystore.jks"
    }
  },
  "ciphers": {
    "aes128GcmSha256": true,
    "ecdheEcdsaAes128GcmSha256": true,
    // ... other cipher configurations
  }
}
```

## Resource Configuration

### Basic Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `private_space_id` | string | Yes | The ID of the private space |
| `organization_id` | string | No | The ID of the target organization (defaults to provider's organization) |
| `name` | string | Yes | The name of the TLS context |
| `keystore_type` | string | Yes | Either "PEM" or "JKS" |

### PEM Keystore Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `certificate` | string | Yes | PEM certificate content |
| `key` | string | Yes | PEM private key content |
| `key_passphrase` | string | No | Private key passphrase |
| `key_filename` | string | No | Key file name |
| `certificate_filename` | string | No | Certificate file name |

### JKS Keystore Attributes

| Attribute | Type | Required | Description |
|-----------|------|----------|-------------|
| `keystore_base64` | string | Yes | Base64 encoded JKS keystore |
| `store_passphrase` | string | Yes | Keystore passphrase |
| `alias` | string | Yes | Certificate alias |
| `key_passphrase` | string | No | Private key passphrase |
| `keystore_filename` | string | No | Keystore file name |

### Cipher Configuration

The `ciphers` block supports the following cipher suites:

| Cipher Suite | Description |
|--------------|-------------|
| `aes128_gcm_sha256` | AES 128-bit GCM with SHA256 |
| `aes256_gcm_sha384` | AES 256-bit GCM with SHA384 |
| `ecdhe_ecdsa_aes128_gcm_sha256` | ECDHE-ECDSA AES 128-bit GCM |
| `ecdhe_ecdsa_aes256_gcm_sha384` | ECDHE-ECDSA AES 256-bit GCM |
| `ecdhe_rsa_aes128_gcm_sha256` | ECDHE-RSA AES 128-bit GCM |
| `ecdhe_rsa_aes256_gcm_sha384` | ECDHE-RSA AES 256-bit GCM |
| `tls_aes256_gcm_sha384` | TLS 1.3 AES 256-bit GCM |
| `tls_aes128_gcm_sha256` | TLS 1.3 AES 128-bit GCM |
| `tls_chacha20_poly1305_sha256` | TLS 1.3 ChaCha20-Poly1305 |

## Usage Examples

### 1. PEM Keystore Example

```hcl
resource "anypoint_tls_context" "pem_example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id  # Optional: specify target organization
  name             = "example-pem-tls-context"
  keystore_type    = "PEM"
  
  certificate              = var.pem_certificate
  key                      = var.pem_private_key
  key_passphrase           = var.pem_key_passphrase
  key_filename             = "server.key"
  certificate_filename     = "server.crt"
  
  ciphers = {
    aes128_gcm_sha256                = true
    ecdhe_ecdsa_aes128_gcm_sha256    = true
    ecdhe_rsa_aes128_gcm_sha256      = true
    tls_aes256_gcm_sha384            = true
    tls_aes128_gcm_sha256            = true
    # All other ciphers default to false
  }
}
```

### 2. JKS Keystore Example

```hcl
resource "anypoint_tls_context" "jks_example" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id  # Optional: specify target organization
  name             = "example-jks-tls-context"
  keystore_type    = "JKS"
  
  keystore_base64     = var.jks_keystore_base64
  store_passphrase    = var.jks_store_passphrase
  alias               = var.jks_alias
  keystore_filename   = "keystore.jks"
  
  ciphers = {
    aes256_gcm_sha384                = true
    ecdhe_ecdsa_aes256_gcm_sha384    = true
    ecdhe_rsa_aes256_gcm_sha384      = true
    tls_aes256_gcm_sha384            = true
    # All other ciphers default to false
  }
}
```

### 3. Data Source Example

```hcl
data "anypoint_tls_context" "existing" {
  private_space_id = var.private_space_id
  organization_id  = var.organization_id  # Optional: specify target organization
  id               = "existing-tls-context-id"
}

output "tls_context_info" {
  value = {
    name            = data.anypoint_tls_context.existing.name
    type            = data.anypoint_tls_context.existing.type
    key_store_cn    = data.anypoint_tls_context.existing.key_store.cn
    expiration_date = data.anypoint_tls_context.existing.key_store.expiration_date
  }
}
```

## Common Use Cases

### 1. Certificate Expiration Monitoring

```hcl
locals {
  expiring_certificates = [
    for ctx in [
      anypoint_tls_context.pem_example,
      anypoint_tls_context.jks_example
    ] : ctx if ctx.key_store != null && ctx.key_store.expiration_date != null
  ]
}

output "certificate_expiration_summary" {
  value = {
    for ctx in local.expiring_certificates : ctx.name => {
      id              = ctx.id
      expiration_date = ctx.key_store.expiration_date
      common_name     = ctx.key_store.cn
      san             = ctx.key_store.san
    }
  }
}
```

### 2. Security Hardening with Restricted Ciphers

```hcl
resource "anypoint_tls_context" "secure" {
  private_space_id = var.private_space_id
  name             = "secure-tls-context"
  keystore_type    = "PEM"
  
  certificate = var.certificate
  key         = var.private_key
  
  # Only allow modern, secure ciphers
  ciphers = {
    # TLS 1.3 ciphers (most secure)
    tls_aes256_gcm_sha384            = true
    tls_chacha20_poly1305_sha256     = true
    tls_aes128_gcm_sha256            = true
    
    # ECDHE with AEAD ciphers (good forward secrecy)
    ecdhe_ecdsa_aes256_gcm_sha384    = true
    ecdhe_rsa_aes256_gcm_sha384      = true
    ecdhe_ecdsa_aes128_gcm_sha256    = true
    ecdhe_rsa_aes128_gcm_sha256      = true
    
    # Disable all other ciphers
    aes128_sha256                    = false
    aes256_sha256                    = false
    dhe_rsa_aes128_sha256            = false
    dhe_rsa_aes256_sha256            = false
    # ... (all others default to false)
  }
}
```

## Best Practices

1. **Use Strong Ciphers**: Enable only modern, secure cipher suites
2. **Monitor Expiration**: Set up alerts for certificate expiration
3. **Rotate Certificates**: Regularly rotate SSL/TLS certificates
4. **Secure Storage**: Store private keys and passphrases securely
5. **Use Variables**: Use Terraform variables for sensitive data

## Testing

1. Copy `terraform.tfvars.example` to `terraform.tfvars`
2. Fill in your specific values
3. Run `terraform plan` to preview changes
4. Run `terraform apply` to create resources

## Troubleshooting

### Common Issues

1. **Invalid Certificate Format**: Ensure PEM certificates include proper headers/footers
2. **Wrong Passphrase**: Verify key passphrases are correct
3. **JKS Encoding**: Ensure JKS files are properly base64 encoded
4. **Cipher Mismatch**: Check that at least one cipher is enabled

### Debugging

Enable Terraform debug logging:
```bash
export TF_LOG=DEBUG
terraform apply
```

## Security Considerations

- Store certificates and keys securely
- Use strong passphrases
- Regularly rotate certificates
- Monitor certificate expiration
- Use only secure cipher suites
- Implement proper access controls 