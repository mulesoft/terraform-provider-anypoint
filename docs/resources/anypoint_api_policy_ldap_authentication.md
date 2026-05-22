---
page_title: "anypoint_api_policy_ldap_authentication Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a LDAP Authentication policy on an Anypoint API instance.
---

# anypoint_api_policy_ldap_authentication (Resource)

Manages a LDAP Authentication policy on an Anypoint API instance.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_api_policy_ldap_authentication" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    ldap_server_url           = "ldap://ldap.example.com:389"
    ldap_server_user_dn       = "cn=admin,dc=example,dc=com"
    ldap_server_user_password = "admin-password"
    ldap_search_base          = "ou=users,dc=example,dc=com"
    ldap_search_filter        = "(uid={0})"
    ldap_search_in_subtree    = true
  }

  order = 1
}
```

## Schema

### Required

- `environment_id` (String) The environment ID.
- `api_instance_id` (String) The API instance ID.
- `configuration` (Block) The policy configuration. See [Configuration](#nestedschema--configuration) below.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `label` (String) A human-readable label for this policy instance.
- `order` (Number) The order of policy execution.
- `asset_version` (String) The policy asset version. Defaults to `1.4.1`.
- `disabled` (Boolean) Whether the policy is disabled. Defaults to `false`.
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.

### Read-Only

- `id` (String) The policy ID.
- `policy_template_id` (String) The policy template ID assigned by the server.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `ldap_search_base` (String) Base DN for LDAP searches.
- `ldap_search_filter` (String) LDAP search filter expression.
- `ldap_server_url` (String) URL of the LDAP server.
- `ldap_server_user_dn` (String) Distinguished name of the LDAP bind user.
- `ldap_server_user_password` (String) Password for the LDAP bind user.

Optional:

- `ldap_search_in_subtree` (Boolean) Whether to search in subtrees.

## Import

An existing `anypoint_api_policy_ldap_authentication` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_ldap_authentication.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_ldap_authentication" "imported" {
  organization_id = "<organization_id>"
  environment_id  = "<environment_id>"
  api_instance_id = "<api_instance_id>"
}
```

After adding the import block, run:

```shell
# Let Terraform generate the full resource configuration automatically:
terraform plan -generate-config-out=generated.tf

# Or apply the import directly if you have an existing resource block:
terraform apply
```

### Using the CLI (deprecated, Terraform < 1.5)

```shell
terraform import anypoint_api_policy_ldap_authentication.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
