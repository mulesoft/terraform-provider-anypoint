# Secrets Management Examples

This directory contains examples for managing Anypoint Platform Secrets Manager resources using Terraform.

## Available Examples

### [Secret Group](./secretgroup/)
- **Resource**: `anypoint_secret_group`
- **Description**: Create and manage secret groups to organize secrets by environment or purpose
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups`
- **Use Cases**:
  - Organize secrets by application or team
  - Create environment-specific secret groups

### [Certificate](./certificate/)
- **Resource**: `anypoint_secret_group_certificate`
- **Description**: Upload and manage X.509 certificates for TLS/SSL configurations
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificates`
- **Use Cases**:
  - Store TLS certificates for secure communications
  - Configure mutual TLS authentication

### [Keystore](./keystore/)
- **Resource**: `anypoint_secret_group_keystore`
- **Description**: Manage keystores containing private keys and certificates
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/keystores`
- **Use Cases**:
  - Store private keys securely
  - Configure client certificates
  - Enable mutual TLS (mTLS)

### [Truststore](./truststore/)
- **Resource**: `anypoint_secret_group_truststore`
- **Description**: Manage truststores containing trusted CA certificates
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/truststores`
- **Use Cases**:
  - Store trusted certificate authorities
  - Validate server certificates

### [TLS Context](./tlscontext/)
- **Resource**: `anypoint_secret_group_tls_context`
- **Description**: Configure TLS contexts combining keystores and truststores for comprehensive TLS setup
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/tlsContexts`
- **Use Cases**:
  - Complete TLS configuration with client and server certificates
  - Configure cipher suites and TLS versions
  - Implement mutual TLS authentication for Omni Gateway APIs

### [Shared Secret](./sharedsecret/)
- **Resource**: `anypoint_secret_group_shared_secret`
- **Description**: Store and manage shared secrets like API keys, passwords, and tokens
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/sharedSecrets`
- **Use Cases**:
  - Store API keys and tokens
  - Manage database passwords
  - Secure application credentials

## Common Setup

All examples in this category require:

1. **Provider Configuration**:
   ```hcl
   terraform {
     required_providers {
       anypoint = {
         source = "example.com/ankitsarda/anypoint"
       }
     }
   }

   provider "anypoint" {
     client_id     = var.anypoint_client_id
     client_secret = var.anypoint_client_secret
     base_url      = var.anypoint_base_url
   }
   ```

2. **Authentication**: Anypoint Platform credentials (Connected App recommended)
3. **Base URL**: `https://anypoint.mulesoft.com` (production) or `https://stgx.anypoint.mulesoft.com` (staging)
4. **Secret Group**: Most resources require an existing secret group
5. **Environment**: All resources are scoped to a specific environment

## Quick Start

1. Navigate to any example directory
2. Copy `terraform.tfvars.example` to `terraform.tfvars` (if available)
3. Fill in your Anypoint Platform credentials and required IDs
4. Run:
   ```bash
   terraform init
   terraform plan
   terraform apply
   ```

## Resource Dependencies

```
Secret Group (Foundation)  ← anypoint_secret_group
├── Certificates           ← anypoint_secret_group_certificate
│   └── X.509 certificates for TLS
├── Keystores              ← anypoint_secret_group_keystore
│   └── Private keys + certificates
├── Truststores            ← anypoint_secret_group_truststore
│   └── Trusted CA certificates
├── TLS Context            ← anypoint_secret_group_tls_context
│   ├── References keystore (optional)
│   └── References truststore (optional)
└── Shared Secrets         ← anypoint_secret_group_shared_secret
    └── API keys, passwords, tokens
```

> **Delete behaviour:** The Secrets Manager API does not support individual sub-resource DELETE (HTTP 405). Deleting a keystore, truststore, certificate, shared secret, or TLS context resource in Terraform only removes it from state — the secret group must be deleted to remove all sub-resources from the Platform. Use `anypoint_secret_group` as the parent resource and declare sub-resources as dependents.

## Common Use Cases

### Complete TLS Setup with Mutual Authentication

```hcl
# 1. Create secret group
resource "anypoint_secret_group" "api_secrets" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  name            = "API Security Secrets"
  downloadable    = false
}

# 2. Upload server certificate + key as keystore (JKS)
resource "anypoint_secret_group_keystore" "server_keystore" {
  organization_id  = var.organization_id
  environment_id   = var.environment_id
  secret_group_id  = anypoint_secret_group.api_secrets.id

  name                 = "Server Keystore"
  type                 = "JKS"
  keystore_file_base64 = filebase64("${path.module}/server-keystore.jks")
  store_passphrase     = var.keystore_store_passphrase
  key_passphrase       = var.keystore_key_passphrase
  alias                = "server"
}

# 3. Create truststore with CA certificates
resource "anypoint_secret_group_truststore" "ca_trust" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.api_secrets.id

  name              = "CA Truststore"
  type              = "JKS"
  truststore_base64 = filebase64("${path.module}/ca-trust.jks")
  passphrase        = var.truststore_passphrase
}

# 4. Configure TLS context (Omni Gateway)
resource "anypoint_secret_group_tls_context" "mtls" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.api_secrets.id

  name                        = "Mutual TLS Context"
  target                      = "outbound"
  keystore_id                 = anypoint_secret_group_keystore.server_keystore.id
  truststore_id               = anypoint_secret_group_truststore.ca_trust.id
  enable_client_cert_validation = true

  alpn_protocols = ["h2", "http/1.1"]
}
```

### API Keys and Credentials Management

```hcl
# Create secret group for application secrets
resource "anypoint_secret_group" "app_secrets" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  name            = "Application Secrets"
}

# Store symmetric key / API key
resource "anypoint_secret_group_shared_secret" "api_key" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.app_secrets.id

  name = "External API Key"
  type = "SymmetricKey"
  key  = var.external_api_key
}

# Store username + password credentials
resource "anypoint_secret_group_shared_secret" "db_creds" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.app_secrets.id

  name     = "Database Credentials"
  type     = "UsernamePassword"
  username = var.db_username
  password = var.db_password
}

# Store AWS S3 credentials
resource "anypoint_secret_group_shared_secret" "s3_creds" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.app_secrets.id

  name              = "S3 Credentials"
  type              = "S3Credential"
  access_key_id     = var.aws_access_key_id
  secret_access_key = var.aws_secret_access_key
}

# Store opaque blob (e.g. OAuth client secret)
resource "anypoint_secret_group_shared_secret" "oauth_secret" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.app_secrets.id

  name    = "OAuth Client Secret"
  type    = "Blob"
  content = var.oauth_client_secret
}
```

### PEM Keystore for Omni Gateway

```hcl
resource "anypoint_secret_group_keystore" "pem_ks" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  secret_group_id = anypoint_secret_group.api_secrets.id

  name             = "PEM Keystore"
  type             = "PEM"
  # For PEM: provide base64-encoded certificate and key contents
  certificate_base64 = base64encode(file("${path.module}/server-cert.pem"))
  key_base64         = base64encode(file("${path.module}/server-key.pem"))
  # ca_path_base64 is optional (certificate chain)
  # key_passphrase is optional (only for encrypted private keys)
}
```

## TLS Context Configuration

### Target Types
- **outbound** - For outgoing connections from Omni Gateway to upstream
- **inbound** - For incoming connections (currently only outbound is used with `anypoint_secret_group_tls_context`)

### Supported TLS Versions
Configure via `min_tls_version` and `max_tls_version` (e.g. `"TLSv1.2"`, `"TLSv1.3"`).

### ALPN Protocols
Specify via `alpn_protocols` list:
- `"h2"` — HTTP/2
- `"http/1.1"` — HTTP/1.1

## Secret Types

### Shared Secret Types
- **SymmetricKey** — API keys, tokens, general-purpose secrets
- **UsernamePassword** — Database passwords, user credentials
- **S3Credential** — AWS access key + secret access key
- **Blob** — Opaque binary or string content

### Keystore/Truststore Types
- **JKS** — Java KeyStore format
- **PKCS12** — Standard format (.p12, .pfx)
- **PEM** — Privacy-Enhanced Mail format (certificate + key as separate base64 fields)
- **JCEKS** — Java Cryptography Extension KeyStore

## Best Practices

### Security
1. **Never Commit Secrets** — Use variables, environment variables, or secure vaults
2. **Rotate Regularly** — Update certificates, keys, and passwords periodically
3. **Least Privilege** — Grant minimum necessary access to secret groups
4. **Monitor Expiration** — Track certificate expiration via `expiration_date` field
5. **Use TLSv1.2+** — Disable older, insecure protocols

### Terraform Management
1. **Use filebase64()** — For binary keystores/truststores; use `base64encode(file(...))` for PEM text
2. **Sensitive Variables** — Mark passphrase/key variables as sensitive
3. **Separate Secrets** — Keep secrets in separate `.tfvars` files (gitignored)
4. **Declare Dependencies** — Sub-resources must declare `depends_on` or reference the secret group

## Troubleshooting

### Certificate Import Errors
- Verify certificate format (PEM, JKS, PKCS12)
- Check certificate chain order
- Ensure private key matches certificate

### Keystore/Truststore Issues
- For JKS/PKCS12/JCEKS: both `store_passphrase` and `key_passphrase` are required
- For PEM: `key_passphrase` is optional (only for encrypted private keys)
- Verify alias exists in keystore
- Check file permissions and path

### Sub-Resource Delete Behaviour
If `terraform destroy` fails with HTTP 405 on a keystore/truststore/certificate/shared secret/TLS context, this is expected — the SM API does not support individual sub-resource DELETE. Remove the sub-resource from Terraform state manually and delete the parent `anypoint_secret_group` to clean up the Platform.

## API Documentation

For detailed API documentation, visit:
- [Anypoint Secrets Manager Documentation](https://docs.mulesoft.com/secrets-manager/)
- [TLS Context Configuration](https://docs.mulesoft.com/secrets-manager/tls-context-create)
- [Certificate Management](https://docs.mulesoft.com/secrets-manager/asm-secret-type-support-reference)
