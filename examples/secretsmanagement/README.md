# Secrets Management Examples

This directory contains examples for managing Anypoint Platform Secrets Manager resources using Terraform.

## Available Examples

### [Secret Group](./secretgroup/)
- **Resource**: `anypoint_secretgroup`
- **Data Source**: `anypoint_secretgroup`
- **Description**: Create and manage secret groups to organize secrets by environment or purpose
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups`
- **Use Cases**:
  - Organize secrets by application or team
  - Create environment-specific secret groups
  - Manage secret group access and permissions

### [Certificate](./certificate/)
- **Resource**: `anypoint_certificate`
- **Data Source**: `anypoint_certificate`
- **Description**: Upload and manage X.509 certificates for TLS/SSL configurations
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificates`
- **Use Cases**:
  - Store TLS certificates for secure communications
  - Manage certificate lifecycle and expiration
  - Configure mutual TLS authentication

### [Keystore](./keystore/)
- **Resource**: `anypoint_keystore`
- **Data Source**: `anypoint_keystore`
- **Description**: Manage keystores containing private keys and certificates
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/keystores`
- **Use Cases**:
  - Store private keys securely
  - Configure client certificates
  - Enable mutual TLS (mTLS)

### [Truststore](./truststore/)
- **Resource**: `anypoint_truststore`
- **Data Source**: `anypoint_truststore`
- **Description**: Manage truststores containing trusted CA certificates
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/truststores`
- **Use Cases**:
  - Store trusted certificate authorities
  - Validate server certificates
  - Implement certificate chain validation

### [TLS Context](./tlscontext/)
- **Resource**: `anypoint_secretsmanager_tls_context`
- **Data Source**: `anypoint_secretsmanager_tlscontext`
- **Description**: Configure TLS contexts combining keystores and truststores for comprehensive TLS setup
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/tlsContexts`
- **Use Cases**:
  - Complete TLS configuration with client and server certificates
  - Configure cipher suites and TLS versions
  - Implement mutual TLS authentication
  - Enable TLS for API proxies and Mule applications

### [Shared Secret](./sharedsecret/)
- **Resource**: `anypoint_shared_secret`
- **Data Source**: `anypoint_sharedsecret`
- **Description**: Store and manage shared secrets like API keys, passwords, and tokens
- **API**: `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/sharedSecrets`
- **Use Cases**:
  - Store API keys and tokens
  - Manage database passwords
  - Secure application credentials
  - Share secrets across applications

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
Secret Group (Foundation)
├── Certificates
│   └── X.509 certificates for TLS
├── Keystores
│   └── Private keys + certificates
├── Truststores
│   └── Trusted CA certificates
├── TLS Context
│   ├── References keystore (optional)
│   └── References truststore (optional)
└── Shared Secrets
    └── API keys, passwords, tokens
```

## Common Use Cases

### Complete TLS Setup with Mutual Authentication

```hcl
# 1. Create secret group
resource "anypoint_secretgroup" "api_secrets" {
  organization_id = var.organization_id
  environment_id  = var.environment_id

  name = "API Security Secrets"
}

# 2. Upload server certificate
resource "anypoint_certificate" "server_cert" {
  secret_group_id = anypoint_secretgroup.api_secrets.id

  name             = "Server Certificate"
  certificate_file = file("${path.module}/server-cert.pem")
  expiration_date  = "2025-12-31"
}

# 3. Create keystore with private key
resource "anypoint_keystore" "server_keystore" {
  secret_group_id = anypoint_secretgroup.api_secrets.id

  name            = "Server Keystore"
  keystore_file   = file("${path.module}/server-keystore.jks")
  keystore_type   = "JKS"
  password        = var.keystore_password
  key_password    = var.key_password
  alias           = "server"
}

# 4. Create truststore with CA certificates
resource "anypoint_truststore" "ca_trust" {
  secret_group_id = anypoint_secretgroup.api_secrets.id

  name           = "CA Truststore"
  truststore_file = file("${path.module}/ca-trust.jks")
  truststore_type = "JKS"
  password       = var.truststore_password
}

# 5. Configure TLS context
resource "anypoint_secretsmanager_tls_context" "mtls" {
  secret_group_id = anypoint_secretgroup.api_secrets.id

  name        = "Mutual TLS Context"
  target      = "inbound"
  tls_version = "TLSv1.2"

  keystore_id   = anypoint_keystore.server_keystore.id
  truststore_id = anypoint_truststore.ca_trust.id

  enable_mutual_authentication = true
}
```

### API Keys and Credentials Management

```hcl
# Create secret group for application secrets
resource "anypoint_secretgroup" "app_secrets" {
  organization_id = var.organization_id
  environment_id  = var.environment_id

  name = "Application Secrets"
}

# Store API key
resource "anypoint_shared_secret" "api_key" {
  secret_group_id = anypoint_secretgroup.app_secrets.id

  name  = "External API Key"
  value = var.external_api_key
  type  = "Symmetric"
}

# Store database credentials
resource "anypoint_shared_secret" "db_password" {
  secret_group_id = anypoint_secretgroup.app_secrets.id

  name  = "Database Password"
  value = var.database_password
  type  = "Password"
}

# Store OAuth client secret
resource "anypoint_shared_secret" "oauth_secret" {
  secret_group_id = anypoint_secretgroup.app_secrets.id

  name  = "OAuth Client Secret"
  value = var.oauth_client_secret
  type  = "ClientSecret"
}
```

### Certificate Lifecycle Management

```hcl
# Upload certificate with expiration tracking
resource "anypoint_certificate" "ssl_cert" {
  secret_group_id = anypoint_secretgroup.api_secrets.id

  name             = "SSL Certificate"
  certificate_file = file("${path.module}/ssl-cert.pem")
  expiration_date  = "2025-06-30"

  lifecycle {
    # Warn before certificate expires
    precondition {
      condition     = timecmp(timestamp(), "2025-05-30T00:00:00Z") < 0
      error_message = "Certificate will expire soon! Renew certificate before 2025-06-30"
    }
  }
}
```

## TLS Context Configuration

### Supported TLS Versions
- **TLSv1.2** - Recommended minimum
- **TLSv1.3** - Latest standard (when available)

### Target Types
- **inbound** - For incoming connections (server)
- **outbound** - For outgoing connections (client)

### Cipher Suites
Configure cipher suites based on security requirements:
- **High Security**: TLS_ECDHE_RSA_WITH_AES_256_GCM_SHA384
- **Standard**: TLS_RSA_WITH_AES_128_GCM_SHA256
- **Legacy Support**: Additional ciphers for compatibility

## Secret Types

### Shared Secret Types
- **Symmetric** - General purpose secrets, API keys
- **Password** - Database passwords, user credentials
- **ClientSecret** - OAuth client secrets
- **Token** - Authentication tokens, bearer tokens

### Keystore/Truststore Types
- **JKS** - Java KeyStore format
- **PKCS12** - Standard format (.p12, .pfx)
- **PEM** - Privacy-Enhanced Mail format

## Best Practices

### Security
1. **Never Commit Secrets** - Use variables, environment variables, or secure vaults
2. **Rotate Regularly** - Update certificates, keys, and passwords periodically
3. **Least Privilege** - Grant minimum necessary access to secret groups
4. **Monitor Expiration** - Track certificate expiration dates
5. **Use TLSv1.2+** - Disable older, insecure protocols

### Organization
1. **Group by Environment** - Separate dev/test/prod secrets
2. **Descriptive Names** - Use clear, consistent naming conventions
3. **Document Purpose** - Add descriptions explaining secret usage
4. **Version Management** - Track certificate and key versions

### Terraform Management
1. **Use file()** - Read certificates from files, don't embed in code
2. **Sensitive Variables** - Mark password variables as sensitive
3. **Lifecycle Rules** - Use preconditions for expiration warnings
4. **Separate Secrets** - Keep secrets in separate .tfvars files (gitignored)

## Certificate Formats

### PEM Format
```
-----BEGIN CERTIFICATE-----
MIIDXTCCAkWgAwIBAgIJAKx...
-----END CERTIFICATE-----
```

### Certificate Chain
Include full chain in order:
1. Server certificate
2. Intermediate CA(s)
3. Root CA

### Private Key Formats
- **PKCS#1**: `-----BEGIN RSA PRIVATE KEY-----`
- **PKCS#8**: `-----BEGIN PRIVATE KEY-----`

## Troubleshooting

### Certificate Import Errors
- Verify certificate format (PEM, DER, PKCS12)
- Check certificate chain order
- Ensure private key matches certificate
- Validate expiration date format

### Keystore/Truststore Issues
- Confirm password is correct
- Verify alias exists in keystore
- Check file permissions and path
- Validate keystore type (JKS, PKCS12)

### TLS Context Configuration
- Ensure keystore and truststore exist before creating context
- Verify target type matches use case (inbound/outbound)
- Check TLS version compatibility
- Validate cipher suite configuration

## API Documentation

For detailed API documentation, visit:
- [Anypoint Secrets Manager Documentation](https://docs.mulesoft.com/secrets-manager/)
- [TLS Context Configuration](https://docs.mulesoft.com/secrets-manager/tls-context-create)
- [Certificate Management](https://docs.mulesoft.com/secrets-manager/asm-secret-type-support-reference)
- [Keystore and Truststore](https://docs.mulesoft.com/secrets-manager/asm-permission-concept)
