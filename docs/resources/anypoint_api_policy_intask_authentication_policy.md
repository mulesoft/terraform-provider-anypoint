---
page_title: "anypoint_api_policy_intask_authentication_policy Resource - terraform-provider-anypoint"
subcategory: "API Policies"
description: |-
  Manages a InTask Authentication policy on an Anypoint API instance.
---

# anypoint_api_policy_intask_authentication_policy (Resource)

Manages a InTask Authentication policy on an Anypoint API instance.

-> **Connected App:** This resource requires a **standard connected app** (client credentials). An admin connected app is not needed. The connected app must have relevant scopes.

## Example Usage

```terraform
resource "anypoint_api_policy_intask_authentication_policy" "example" {
  organization_id = var.organization_id
  environment_id  = var.environment_id
  api_instance_id = anypoint_api_instance.example.id

  configuration = {
    secondary_auth_provider = "example-provider"
    authorization_endpoint  = "https://auth.example.com/authorize"
    token_endpoint          = "https://auth.example.com/token"
    redirect_uri            = "https://app.example.com/callback"
    scopes                  = "openid profile"
    response_type           = "code"
    token_timeout           = 3600
  }

  upstream_ids = [anypoint_api_upstream.example.id]
}
```

## Schema

### Required

- `environment_id` (String) The environment ID.
- `api_instance_id` (String) The API instance ID.
- `configuration` (Block) The policy configuration. See [Configuration](#nestedschema--configuration) below.
- `upstream_ids` (List of String) List of upstream IDs this policy applies to.

### Optional

- `organization_id` (String) The organization ID. If not provided, the organization ID will be inferred from the connected app credentials.
- `label` (String) A human-readable label for this policy instance.
- `asset_version` (String) The policy asset version. Defaults to `1.0.0-20260113204639`.

### Read-Only

- `id` (String) The policy ID.
- `policy_template_id` (String) The policy template ID assigned by the server.

<a id="nestedschema--configuration"></a>
### Nested Schema for `configuration`

Required:

- `authorization_endpoint` (String) URL of the OAuth 2.0 authorization endpoint.
- `redirect_uri` (String) The redirect URI for the OAuth 2.0 flow.
- `secondary_auth_provider` (String) Name of the secondary authentication provider.
- `token_endpoint` (String) URL of the OAuth 2.0 token endpoint.

Optional:

- `body_encoding` (String) Encoding for the token request body.
- `challenge_response_status_code` (Number) HTTP status code for the challenge response.
- `code_challenge_method` (String) The PKCE code challenge method (e.g. `S256`).
- `response_type` (String) The OAuth 2.0 response type (e.g. `code`).
- `scopes` (String) Space-separated list of required OAuth scopes.
- `token_audience` (String) Expected audience value for the token.
- `token_timeout` (Number) Token validity timeout in seconds.
- `user_email_header` (String) Header name to extract the user email from.
- `user_id_header` (String) Header name to extract the user ID from.

## Import

An existing `anypoint_api_policy_intask_authentication_policy` policy can be imported using its composite ID: `organization_id/environment_id/api_instance_id/policy_id`.

The `policy_id` is the numeric ID of the policy (visible in Anypoint API Manager or from the API response).

### Using an import block (Terraform ≥ 1.5 — recommended)

```terraform
import {
  to = anypoint_api_policy_intask_authentication_policy.imported
  id = "<organization_id>/<environment_id>/<api_instance_id>/<policy_id>"
}

resource "anypoint_api_policy_intask_authentication_policy" "imported" {
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
terraform import anypoint_api_policy_intask_authentication_policy.imported <organization_id>/<environment_id>/<api_instance_id>/<policy_id>
```
