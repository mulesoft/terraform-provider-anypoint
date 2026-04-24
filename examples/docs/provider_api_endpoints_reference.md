# Anypoint Terraform Provider - REST API Endpoints Reference

**Generated:** March 30, 2026

## Overview
This document provides a comprehensive reference of all REST API endpoints used by the Anypoint Terraform Provider, organized by module and resource.

**Total Resources:** 37
**Base URL:** All endpoints are prefixed with the Anypoint Platform base URL (typically `https://anypoint.mulesoft.com`)

---

## API Module Prefixes

| Module | API Prefix | Description |
|--------|-----------|-------------|
| Access Management | `/accounts/api/` | User, team, role, and organization management |
| API Management | `/apimanager/api/v1/` | API instance and policy management |
| Monitoring | `/monitoring/api/alerts/api/v2/` | Alert management |
| Gateway Manager | `/gatewaymanager/api/v1/` | Flex Gateway management |
| CloudHub 2.0 | `/runtimefabric/api/` | Private space and networking |
| Secrets Management | `/secrets-manager/api/v1/` | Secrets, certificates, and TLS contexts |

---

## Access Management APIs

### Connected App
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/accounts/api/connectedApplications` |
| Read | GET | `/accounts/api/connectedApplications/{clientID}` |
| Delete | DELETE | `/accounts/api/connectedApplications/{clientID}` |

**Notes:** Update not supported (immutable resource)

### Connected App Scopes
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Read | GET | `/accounts/api/connectedApplications/{connectedAppID}/scopes` |
| Update | PATCH | `/accounts/api/connectedApplications/{connectedAppID}/scopes` |
| Delete | DELETE | `/accounts/api/connectedApplications/{connectedAppID}/scopes` |

**Notes:** Create uses Update endpoint (PATCH)

### Environment
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/accounts/api/organizations/{orgId}/environments` |
| Read | GET | `/accounts/api/organizations/{orgId}/environments/{environmentId}` |
| Update | PUT | `/accounts/api/organizations/{orgId}/environments/{environmentId}` |
| Delete | DELETE | `/accounts/api/organizations/{orgId}/environments/{environmentId}` |

### Organization
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/accounts/api/organizations` |
| Read | GET | `/accounts/api/organizations/{organizationId}` |
| Delete | DELETE | `/accounts/api/organizations/{organizationId}` |

**Notes:** Update not supported; Delete includes polling for async completion


### Team
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/accounts/api/organizations/{orgId}/teams` |
| Read | GET | `/accounts/api/organizations/{orgId}/teams/{teamId}` |
| Update | PATCH | `/accounts/api/organizations/{orgId}/teams/{teamId}` |
| Update Parent | PUT | `/accounts/api/organizations/{orgId}/teams/{teamId}/parent` |
| Delete | DELETE | `/accounts/api/organizations/{orgId}/teams/{teamId}` |

**Notes:** Parent changes use separate PUT endpoint

### Team Members
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Add | PATCH | `/accounts/api/organizations/{orgId}/teams/{teamId}/members` |
| Read | GET | `/accounts/api/organizations/{orgId}/teams/{teamId}/members` |
| Remove | DELETE | `/accounts/api/organizations/{orgId}/teams/{teamId}/members` |

**Notes:** Update = Remove (DELETE) + Add (PATCH)

### Team Roles
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Assign | POST | `/accounts/api/organizations/{orgId}/teams/{teamId}/roles` |
| Read | GET | `/accounts/api/organizations/{orgId}/teams/{teamId}/roles` |
| Remove | DELETE | `/accounts/api/organizations/{orgId}/teams/{teamId}/roles` |

**Notes:** Update = Remove (DELETE) + Assign (POST)

### User
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/accounts/api/organizations/{orgId}/users` |
| Read | GET | `/accounts/api/organizations/{orgId}/users/{userId}` |
| Update | PUT | `/accounts/api/organizations/{orgId}/users/{userId}` |
| Delete | DELETE | `/accounts/api/organizations/{orgId}/users/{userId}` |

---

## API Management APIs

### Alert
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/monitoring/api/alerts/api/v2/organizations/{orgId}/environments/{envId}/alerts` |
| Read | GET | `/monitoring/api/alerts/api/v2/organizations/{orgId}/environments/{envId}/alerts/{alertId}` |
| Update | PUT | `/monitoring/api/alerts/api/v2/organizations/{orgId}/environments/{envId}/alerts/{alertId}` |
| Delete | DELETE | `/monitoring/api/alerts/api/v2/organizations/{orgId}/environments/{envId}/alerts/{alertId}` |

### API Instance
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis` |
| Promote | POST | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis` |
| Read | GET | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}` |
| Update | PATCH | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}` |
| Delete | DELETE | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}` |
| Get Gateway | GET | `/gatewaymanager/xapi/v1/organizations/{orgId}/environments/{envId}/gateways/{gatewayId}` |

**Notes:** Promote uses same POST endpoint with different payload; Gateway info resolved during creation

### API Policy
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/policies?allowDuplicated=true` |
| Read | GET | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/policies/{policyId}` |
| Update | PATCH | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/policies/{policyId}` |
| Delete | DELETE | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/policies/{policyId}` |

**Notes:** Includes policy lookup and validation before applying

### SLA Tier
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/tiers` |
| Read/List | GET | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/tiers` |
| Update | PUT | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/tiers/{tierId}` |
| Delete | DELETE | `/apimanager/api/v1/organizations/{orgId}/environments/{envId}/apis/{apiId}/tiers/{tierId}` |

**Notes:** Read operation uses list endpoint and filters by tier ID

### Managed Flex Gateway
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/gatewaymanager/api/v1/organizations/{orgId}/environments/{envId}/gateways` |
| Read | GET | `/gatewaymanager/api/v1/organizations/{orgId}/environments/{envId}/gateways/{gatewayId}` |
| Update | PATCH | `/gatewaymanager/api/v1/organizations/{orgId}/environments/{envId}/gateways/{gatewayId}` |
| Delete | DELETE | `/gatewaymanager/api/v1/organizations/{orgId}/environments/{envId}/gateways/{gatewayId}` |
| Get Versions | GET | `/gatewaymanager/xapi/v1/gateway/versions` |
| Get Domains | GET | `/runtimefabric/api/organizations/{orgId}/targets/{targetId}/environments/{envId}/domains?sendAppUniqueId=true` |

**Notes:** Auto-selects runtime version; Builds ingress URLs from target domains

---

## CloudHub 2.0 APIs

### Private Space
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/runtimefabric/api/organizations/{orgId}/privatespaces` |
| Read | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |
| Update | PUT | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |
| Delete | DELETE | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |

### Firewall Rules
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create/Update/Delete | PATCH | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |
| Read | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |

**Notes:** All operations use PATCH on private space; Delete sends empty rules array

### Private Network
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create/Update | PATCH | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |
| Read | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |

**Notes:** Delete is no-op (deleted with private space)

### Private Space Advanced Config
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create/Update/Delete | PATCH | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |
| Read | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}` |

**Notes:** Delete resets configuration to defaults

### Private Space Association
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/associations` |
| Update | POST | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/associations` |
| Delete | DELETE | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/associations/{associationId}` |

**Notes:** No Read API - state maintained client-side

### Private Space Connection
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections` |
| Read | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections/{connectionId}` |
| Update | PUT | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections/{connectionId}` |
| Delete | DELETE | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections/{connectionId}` |

### Private Space Upgrade
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Schedule | POST | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/upgrade` |
| Cancel | DELETE | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/upgrade` |

**Notes:** One-time scheduled operation; No Read or Update

### TLS Context (CloudHub 2.0)
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/tlsContexts` |
| List | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/tlsContexts` |
| Read | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/tlsContexts/{tlsContextId}` |
| Update | PUT | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/tlsContexts/{tlsContextId}` |
| Delete | DELETE | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/tlsContexts/{tlsContextId}` |

**Notes:** Create returns 201 with no body; Uses List to find created resource

### VPN Connection
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections` |
| Read | GET | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections/{connectionId}` |
| Delete Connection | DELETE | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections/{connectionId}` |
| Delete VPN | DELETE | `/runtimefabric/api/organizations/{orgId}/privatespaces/{privateSpaceId}/connections/{connectionId}/vpns/{vpnId}` |

**Notes:** Update = Delete VPN + implicit re-add through Read

---

## Secrets Management APIs

### Secret Group
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups` |
| Read | GET | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}` |
| Update | PATCH | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}` |
| Delete | DELETE | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}` |

### Certificate
| Operation | Method | Endpoint | Content-Type |
|-----------|--------|----------|--------------|
| Create | POST | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificates` | multipart/form-data |
| Read | GET | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificates/{certificateId}` | - |
| Update | PUT | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificates/{certificateId}` | multipart/form-data |
| Delete | DELETE | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificates/{certificateId}` | - |

### Certificate Pin Set
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificate-pinsets` |
| Read | GET | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificate-pinsets/{pinsetId}` |
| Update | PUT | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificate-pinsets/{pinsetId}` |
| Delete | DELETE | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/certificate-pinsets/{pinsetId}` |

### Keystore
| Operation | Method | Endpoint | Content-Type |
|-----------|--------|----------|--------------|
| Create | POST | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/keystores` | multipart/form-data |
| Read | GET | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/keystores/{keystoreId}` | - |
| Update | PUT | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/keystores/{keystoreId}` | multipart/form-data |
| Delete | DELETE | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/keystores/{keystoreId}` | - |

**Notes:** Supports PEM, JKS, PKCS12, and JCEKS formats

### Shared Secret
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/shared-secrets` |
| Read | GET | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/shared-secrets/{sharedSecretId}` |
| Update | PUT | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/shared-secrets/{sharedSecretId}` |
| Delete | DELETE | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/shared-secrets/{sharedSecretId}` |

### TLS Context (Flex Gateway)
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/tlscontexts` |
| Read | GET | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/tlscontexts/{tlsContextId}` |
| Update | PUT | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/tlscontexts/{tlsContextId}` |
| Delete | DELETE | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/tlscontexts/{tlsContextId}` |

**Notes:** Target automatically set to FlexGateway

### Truststore
| Operation | Method | Endpoint |
|-----------|--------|----------|
| Create | POST | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/truststores` |
| Read | GET | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/truststores/{truststoreId}` |
| Update | PUT | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/truststores/{truststoreId}` |
| Delete | DELETE | `/secrets-manager/api/v1/organizations/{orgId}/environments/{envId}/secretGroups/{secretGroupId}/truststores/{truststoreId}` |

**Notes:** Supports PEM, JKS, PKCS12, and JCEKS formats

---

## Common Path Parameters

| Parameter | Description | Example |
|-----------|-------------|---------|
| `{orgId}` | Organization ID (UUID) | `a1b2c3d4-e5f6-7890-abcd-ef1234567890` |
| `{envId}` | Environment ID (UUID) | `b2c3d4e5-f6a7-8901-bcde-f12345678901` |
| `{privateSpaceId}` | Private Space ID | `my-private-space` |
| `{apiId}` | API Instance ID (numeric) | `12345678` |
| `{policyId}` | Policy ID (numeric) | `987654` |
| `{tierId}` | SLA Tier ID (numeric) | `123456` |
| `{alertId}` | Alert ID (numeric) | `456789` |
| `{gatewayId}` | Gateway ID | `flex-gateway-01` |
| `{clientID}` | Connected App Client ID | `abc123def456` |
| `{userId}` | User ID (UUID) | `c3d4e5f6-a7b8-9012-cdef-123456789012` |
| `{teamId}` | Team ID (UUID) | `d4e5f6a7-b8c9-0123-def0-123456789abc` |
| `{secretGroupId}` | Secret Group ID | `sg-12345678-abcd-ef01-2345-6789abcdef01` |
| `{certificateId}` | Certificate ID | `cert-123abc` |
| `{keystoreId}` | Keystore ID | `ks-456def` |
| `{truststoreId}` | Truststore ID | `ts-789ghi` |
| `{tlsContextId}` | TLS Context ID | `tls-012jkl` |

---

## HTTP Methods Summary

| Method | Usage | Count |
|--------|-------|-------|
| GET | Read operations | 37 |
| POST | Create operations, assignments | 28 |
| PUT | Full updates | 16 |
| PATCH | Partial updates | 10 |
| DELETE | Delete operations, removals | 32 |

---

## Special Patterns

### Multipart Form Data
The following resources use `multipart/form-data` for file uploads:
- Certificate (Create, Update)
- Keystore (Create, Update)

### No Dedicated Read API
- **Private Space Association**: State maintained client-side only

### Multiple Operations on Same Endpoint
- **Firewall Rules**: Create/Update/Delete all use PATCH
- **Private Network**: Create/Update both use PATCH
- **Private Space Advanced Config**: Create/Update/Delete all use PATCH

### Async Operations
- **Organization Delete**: Includes polling/wait for async completion

### Query Parameters
- **API Policy Create**: `?allowDuplicated=true`
- **Managed Flex Gateway Domains**: `?sendAppUniqueId=true`

---

## Base URL Configuration

The provider uses a configurable base URL, typically:
- **Production**: `https://anypoint.mulesoft.com`
- **EU**: `https://eu1.anypoint.mulesoft.com`
- **Gov Cloud**: `https://gov.anypoint.mulesoft.com`

All endpoints in this document are relative to the configured base URL.

---

## Authentication

All API requests require authentication via:
- **Bearer Token**: Standard OAuth 2.0 bearer token in `Authorization` header
- **Connected App**: Client credentials grant flow

Some APIs also require organization context via headers:
- `X-ANYPNT-ORG-ID`: Organization ID
- `X-ANYPNT-ENV-ID`: Environment ID (for environment-scoped operations)

---

For detailed request/response schemas and examples, refer to the [Anypoint Platform API Documentation](https://anypoint.mulesoft.com/exchange/portals/anypoint-platform/).
