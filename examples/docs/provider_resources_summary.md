# Anypoint Terraform Provider - Resources Summary

**Generated:** March 30, 2026

## Overview
This document provides a comprehensive summary of all resources supported by the Anypoint Terraform Provider, including their CRUD operations and corresponding API methods.

**Total Resources:** 37

## Resource Breakdown by Category

### Access Management (11 Resources)
- Connected App
- Connected App Scopes
- Environment
- Organization
- Team
- Team Members
- Team Roles
- User

### API Management (6 Resources)
- Alert
- API Instance
- API Instance Promotion
- API Policy
- SLA Tier
- Managed Flex Gateway

### CloudHub 2.0 (11 Resources)
- Firewall Rules
- Private Network
- Private Space
- Private Space Advanced Config
- Private Space Association
- Private Space Connection
- Private Space Upgrade
- TLS Context (CloudHub2)
- VPN Connection

### Secrets Management (8 Resources)
- Certificate
- Certificate Pin Set
- Keystore
- Secret Group
- Shared Secret
- TLS Context (Flex Gateway)
- Truststore

## CRUD Operation Statistics

### Full CRUD Support (Create, Read, Update, Delete)
**31 Resources** - 84% of all resources

### Partial CRUD Support
**6 Resources** - 16% of all resources

#### No Update Support (2 Resources):
1. **Connected App** - Update returns error (immutable resource)
2. **Organization** - Update returns warning (immutable resource)

#### Special Delete Behavior (2 Resources):
1. **Private Network** - Delete is no-op (deleted with private space)
2. **Private Space Advanced Config** - Delete resets to defaults

#### Limited Read Support (1 Resource):
1. **Private Space Association** - No Read API available (maintains state only)

#### One-Time Operations (1 Resource):
1. **Private Space Upgrade** - Update not supported (one-time scheduled operation)

## Notable Implementation Patterns

### Composite Create Operations
Resources that call multiple APIs during creation:
- **API Instance**: `CreateAPIInstance() + GetGatewayInfo()`
- **Managed Flex Gateway**: `CreateManagedFlexGateway() + GetGatewayVersions() + GetDomains()`
- **TLS Context (CloudHub2)**: `CreateTLSContext() + ListTLSContexts()` (Create returns 201 with no body)

### Composite Update Operations
Resources that perform complex update logic:
- **Team Members/Roles**: Remove existing + Add new
- **VPN Connection**: `DeleteVPN() + implicit add through Read`

### Composite Delete Operations
Resources with special delete handling:
- **Organization**: `DeleteOrganization() + WaitForOrganizationDeletion()` (async operation)
- **Private Space Association**: Deletes each association individually in loop

### Multi-Step Operations
Resources requiring additional API calls:
- **API Policy**: Uses `LookupPolicy()` and `ValidatePolicyConfiguration()`
- **Team**: Uses `UpdateTeam()` and `UpdateTeamParent()` for parent changes

## API Client Methods Reference

### Access Management Client Methods
- CreateConnectedApp(), GetConnectedApp(), DeleteConnectedApp()
- UpdateConnectedAppScopes(), GetConnectedAppScopes(), DeleteConnectedAppScopes()
- CreateEnvironment(), GetEnvironment(), UpdateEnvironment(), DeleteEnvironment()
- CreateOrganization(), GetOrganization(), DeleteOrganization(), WaitForOrganizationDeletion()
- CreateTeam(), GetTeam(), UpdateTeam(), UpdateTeamParent(), DeleteTeam()
- AddMembersToTeam(), GetTeamMembers(), RemoveMembersFromTeam()
- AssignRolesToTeam(), GetTeamRoles(), RemoveRolesFromTeam()
- CreateUser(), GetUser(), UpdateUser(), DeleteUser()

### API Management Client Methods
- CreateAlert(), GetAlert(), UpdateAlert(), DeleteAlert()
- CreateAPIInstance(), GetAPIInstance(), UpdateAPIInstance(), DeleteAPIInstance(), GetGatewayInfo()
- PromoteAPIInstance()
- CreateAPIPolicy(), GetAPIPolicy(), UpdateAPIPolicy(), DeleteAPIPolicy(), LookupPolicy(), ValidatePolicyConfiguration()
- CreateSLATier(), GetSLATier(), UpdateSLATier(), DeleteSLATier()
- CreateManagedFlexGateway(), GetManagedFlexGateway(), UpdateManagedFlexGateway(), DeleteManagedFlexGateway(), GetGatewayVersions(), GetDomains()

### CloudHub 2.0 Client Methods
- UpdateFirewallRules(), GetFirewallRules()
- CreatePrivateNetwork(), GetPrivateNetwork(), UpdatePrivateNetwork()
- CreatePrivateSpace(), GetPrivateSpace(), UpdatePrivateSpace(), DeletePrivateSpace()
- UpdatePrivateSpaceAdvancedConfig()
- CreatePrivateSpaceAssociations(), DeletePrivateSpaceAssociation()
- CreatePrivateSpaceConnection(), GetPrivateSpaceConnection(), UpdatePrivateSpaceConnection(), DeletePrivateSpaceConnection()
- UpgradePrivateSpace(), DeletePrivateSpaceUpgrade()
- CreateTLSContext(), GetTLSContext(), UpdateTLSContext(), DeleteTLSContext(), ListTLSContexts()
- CreateVPNConnection(), GetVPNConnection(), DeleteVPN(), DeleteVPNConnection()

### Secrets Management Client Methods
- CreateCertificate(), GetCertificate(), UpdateCertificate(), DeleteCertificate()
- CreateCertificatePinset(), GetCertificatePinset(), UpdateCertificatePinset(), DeleteCertificatePinset()
- CreateKeystore(), GetKeystore(), UpdateKeystore(), DeleteKeystore()
- CreateSecretGroup(), GetSecretGroup(), UpdateSecretGroup(), DeleteSecretGroup()
- CreateSharedSecret(), GetSharedSecret(), UpdateSharedSecret(), DeleteSharedSecret()
- CreateTLSContext(), GetTLSContext(), UpdateTLSContext(), DeleteTLSContext()
- CreateTruststore(), GetTruststore(), UpdateTruststore(), DeleteTruststore()

## Resource-Specific Notes

### Access Management
- **Connected App**: Update not supported - resource is immutable after creation
- **Organization**: Update returns warning - most fields are immutable; Delete includes async wait
- **Team**: Supports parent team changes via UpdateTeamParent()

### API Management
- **API Instance**: Auto-resolves gateway information during creation
- **API Instance Promotion**: Only instance_label field can be updated
- **API Policy**: Validates policy configuration before applying
- **Managed Flex Gateway**: Auto-selects runtime version and builds ingress URLs from target domain

### CloudHub 2.0
- **Firewall Rules**: Create and Update both use UpdateFirewallRules(); Delete sends empty rules list
- **Private Network**: Delete is no-op - network deleted when private space is deleted
- **Private Space Advanced Config**: Delete resets configuration to defaults
- **Private Space Association**: No Read API - state maintained client-side only
- **Private Space Upgrade**: One-time scheduled operation - update not supported
- **TLS Context**: Create returns 201 with no body, requires ListTLSContexts() to find created resource
- **VPN Connection**: Update implemented as delete + re-add

### Secrets Management
- **Keystore**: Supports PEM, JKS, PKCS12, and JCEKS formats
- **Truststore**: Supports PEM, JKS, PKCS12, and JCEKS formats
- **TLS Context (Flex)**: Target automatically set to FlexGateway
- **Certificate Pin Set**: Used for certificate pinning validation

## File Locations

### Resource Implementations
- **Access Management**: `/internal/resource/accessmanagement/`
- **API Management**: `/internal/resource/apimanagement/`
- **CloudHub 2.0**: `/internal/resource/cloudhub2/`
- **Secrets Management**: `/internal/resource/secretsmanagement/`

### Data Source Implementations
- **Access Management**: `/internal/datasource/accessmanagement/`
- **CloudHub 2.0**: `/internal/datasource/cloudhub2/`

## Key Observations

1. **Consistent Patterns**: Most resources follow standard CRUD patterns with client methods named consistently
2. **Relationship Management**: Several resources manage relationships (roles, users, members) with assign/remove operations
3. **Async Operations**: Organization deletion includes async wait functionality
4. **Immutable Resources**: Connected App and Organization updates are not supported
5. **Complex Dependencies**: Some resources require multiple API calls to complete operations
6. **Format Support**: Keystore and Truststore resources support multiple certificate formats
7. **Auto-Resolution**: Some resources auto-resolve dependencies (gateway info, versions, domains)
8. **State Management**: Private Space Association maintains state client-side due to no Read API

---

## Related Documentation

This summary is part of a comprehensive provider documentation set:

1. **provider_resources_crud_apis.csv** - Original CSV with client method names for each CRUD operation
2. **provider_resources_rest_apis.csv** - Complete CSV with actual REST API endpoints (HTTP methods and paths)
3. **provider_api_endpoints_reference.md** - Detailed reference guide for all REST API endpoints
4. **provider_resources_summary.md** - This file - high-level overview and statistics

For detailed API endpoint information including HTTP methods, paths, and parameters, refer to:
- **provider_resources_rest_apis.csv** for a quick reference table
- **provider_api_endpoints_reference.md** for comprehensive endpoint documentation with notes and examples
