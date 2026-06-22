# API Policy Data Sources

This document describes the two new data sources for API policies.

## Data Sources

### 1. `anypoint_api_policy` - Single Policy Lookup

Retrieves a single API policy by its ID.

#### Example Usage

```hcl
data "anypoint_api_policy" "rate_limit" {
  organization_id  = "my-org-id"  # Optional, defaults to provider org
  environment_id   = "dev"
  api_instance_id  = "123"
  policy_id        = "456"
}

output "policy_config" {
  value = jsondecode(data.anypoint_api_policy.rate_limit.configuration_json)
}
```

#### Schema

**Required:**
- `environment_id` (String) - The environment ID
- `api_instance_id` (String) - The API instance ID
- `policy_id` (String) - The policy ID to retrieve

**Optional:**
- `organization_id` (String) - The organization ID (defaults to provider org)

**Computed:**
- `id` (String) - Composite ID: `<org_id>/<env_id>/<api_id>/<policy_id>`
- `policy_template_id` (String) - Policy template identifier
- `group_id` (String) - Exchange group ID
- `asset_id` (String) - Exchange asset ID
- `asset_version` (String) - Asset version
- `configuration_json` (String) - JSON-encoded policy configuration
- `order` (Int64) - Execution order
- `disabled` (Bool) - Whether the policy is disabled
- `pointcut_json` (String) - JSON-encoded pointcut data

---

### 2. `anypoint_api_policies` - List All Policies

Lists all policies for an API instance.

#### Example Usage

```hcl
data "anypoint_api_policies" "all" {
  environment_id   = "dev"
  api_instance_id  = "123"
}

output "policy_count" {
  value = length(data.anypoint_api_policies.all.policies)
}

output "enabled_policies" {
  value = [
    for policy in data.anypoint_api_policies.all.policies :
    policy.policy_template_id if !policy.disabled
  ]
}
```

#### Schema

**Required:**
- `environment_id` (String) - The environment ID
- `api_instance_id` (String) - The API instance ID

**Optional:**
- `organization_id` (String) - The organization ID (defaults to provider org)

**Computed:**
- `id` (String) - Composite ID: `<org_id>/<env_id>/<api_id>`
- `policies` (List of Object) - List of policies, each with:
  - `id` (String) - Policy ID
  - `policy_template_id` (String) - Policy template identifier
  - `group_id` (String) - Exchange group ID
  - `asset_id` (String) - Exchange asset ID
  - `asset_version` (String) - Asset version
  - `configuration_json` (String) - JSON-encoded policy configuration
  - `order` (Int64) - Execution order
  - `disabled` (Bool) - Whether the policy is disabled
  - `pointcut_json` (String) - JSON-encoded pointcut data

## Implementation Details

- Both data sources use `strconv.Atoi` to convert string IDs to integers for the API client
- JSON fields (`configuration_json`, `pointcut_json`) are null if the underlying byte arrays are empty
- The client methods used are:
  - `GetAPIPolicy(ctx, orgID, envID, apiID int, policyID int)` for single lookup
  - `ListAPIPolicies(ctx, orgID, envID, apiID int)` for list

## Test Coverage

Both data sources include comprehensive tests:
- Constructor validation
- Metadata/schema validation  
- Configuration validation
- Successful reads
- Error handling
- Invalid ID handling
- Benchmarks
