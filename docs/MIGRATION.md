# Migrating from the Community Provider (`mulesoft-anypoint/anypoint`)

This guide helps existing users of the **community** Anypoint Terraform provider
([mulesoft-anypoint/anypoint](https://registry.terraform.io/providers/mulesoft-anypoint/anypoint/latest)
— currently v1.8.2) move their Terraform-managed Anypoint resources onto the
**official** provider shipped in this repository (installed locally as
`sf.com/mulesoft/anypoint`).

> **Disclaimer on terminology.** Despite its `mulesoft-anypoint` namespace, the
> community provider is explicitly *not* part of the official MuleSoft product
> stack and is not covered by MuleSoft support SLAs (per its own
> [README](https://github.com/mulesoft-anypoint/terraform-provider-anypoint)).
> This document is aimed at customers who want to move off of it and onto the
> officially-supported provider.

## Before you start

Read the whole document before running anything. In particular, the
**compatibility matrix** below tells you whether your current Terraform
modules are:

- Mechanically migratable (a scripted `state replace-provider` + a `moved`
  block is enough),
- Migratable with some hand-editing (schema diffs need to be resolved), or
- Not migratable today because the official provider does not yet ship the
  equivalent resource.

If your modules use resources flagged **Not supported**, you have two choices:

1. Leave those resources under the community provider, remove them from the
   module you're migrating, and manage them out-of-band for now.
2. Use the official provider for everything it covers and keep a small
   community-provider module for the gaps. Both providers can coexist in the
   same repository (they just can't manage the same resource instance).

## Prerequisites

1. Terraform CLI 1.5 or newer (this repo's examples are validated against 1.7+).
2. The official provider installed locally via the distribution tarball — see
   `dist/packages/DISTRIBUTION_README.md`. The install drops the binary under
   `~/.terraform.d/plugins/sf.com/mulesoft/anypoint/0.1.0/<platform>/`.
3. **A clean `terraform plan` on the community provider.** Migrate from a known
   stable state, never from one that already has drift.
4. Both sets of credentials:
   - Connected app `client_id` / `client_secret` (both providers need these).
   - An **admin user login** (username / password) for the official provider —
     it requires it for admin-scoped operations (org creation, user invites,
     team management). The community provider's `username`/`password` fields
     are deprecated and unused, so most customers will need to provision a
     service user.

## Authentication — what changes

| Aspect | Community | Official |
|---|---|---|
| Auth env vars | `ANYPOINT_CLIENT_ID`, `ANYPOINT_CLIENT_SECRET`, `ANYPOINT_ACCESS_TOKEN` | `ANYPOINT_CLIENT_ID`, `ANYPOINT_CLIENT_SECRET`, `ANYPOINT_USERNAME`, `ANYPOINT_PASSWORD` (also `ANYPOINT_ADMIN_USERNAME` / `ANYPOINT_ADMIN_PASSWORD` fallbacks) |
| Provider block `username` / `password` | **Deprecated**, unused | **Required for admin ops** (org, user, team) |
| Control plane selection | `cplane = "us" \| "eu" \| "gov"` | `base_url = "https://<region>.anypoint.mulesoft.com"` |
| Pre-minted access token | `access_token` attribute supported | Not supported — always re-authenticates |

After migration, your `provider "anypoint" {}` block will look like:

```hcl
provider "anypoint" {
  client_id     = var.anypoint_client_id
  client_secret = var.anypoint_client_secret
  username      = var.anypoint_username
  password      = var.anypoint_password
  base_url      = "https://anypoint.mulesoft.com"   # or https://eu1.anypoint.mulesoft.com
}
```

## Compatibility matrix

**Legend:**

- **As-is** — same resource type name, same (or near-identical) schema.
  `state replace-provider` is enough.
- **Renamed** — different type name but compatible schema. Needs a `moved {}`
  block (and sometimes a `state mv`).
- **Schema change** — same or new type name, schema differs enough that you'll
  need to edit your HCL before `terraform plan` is clean.
- **Not supported** — no equivalent in the official provider yet. Keep using the
  community provider for this resource, or remove it from Terraform management.
- **New in official** — resource exists only on the official side; not relevant
  to the migration, but useful to know.

### Access Management

| Community resource | Official resource | Status | Notes |
|---|---|---|---|
| `anypoint_bg` | `anypoint_organization` | **Renamed + Schema change** | Business Group → Organization. `entitlements` is a nested object on the official side (one attribute per entitlement) rather than the flat map used by the community provider. Quota entitlements (`vcores_*`, `vpcs`, `network_connections`) are modeled as nested objects with `assigned` / `reassigned`. Note: `static_ips` and `vpns` are server-managed by Anypoint and not settable via Terraform, so they are intentionally excluded from the schema. |
| `anypoint_env` | `anypoint_environment` | **Renamed** | Schema similar: `name`, `org_id` → `organization_id`, `type`. |
| `anypoint_user` | — | **Not supported** | Removed from the official provider. |
| `anypoint_user_rolegroup` | — | **Not supported** | No equivalent. Role-group based access isn't modelled in the official provider; manage roles via `anypoint_team_roles` / `anypoint_team_members` instead. |
| `anypoint_rolegroup` | — | **Not supported** | Same as above. |
| `anypoint_rolegroup_roles` | — | **Not supported** | Same as above. |
| `anypoint_team` | `anypoint_team` | **As-is** | Type name matches; minor schema review recommended. |
| `anypoint_team_roles` | — | **Not supported** | Removed from the official provider. |
| `anypoint_team_member` | — | **Not supported** | Removed from the official provider. |
| `anypoint_team_group_mappings` | — | **Not supported** | External IdP group → team mappings aren't in the official provider yet. |
| `anypoint_idp_oidc` | — | **Not supported** | Identity-provider management is not in scope for the official provider today. |
| `anypoint_idp_saml` | — | **Not supported** | Same. |
| `anypoint_connected_app` | `anypoint_connected_app` | **As-is (schema change)** | Type name matches. Scope configuration on the community provider is inline; official provider splits it into a separate `anypoint_connected_app_scopes` resource. |
| — | `anypoint_connected_app_scopes` | **New in official** | Dedicated resource for scope assignment. |

### API Management

| Community resource | Official resource | Status | Notes |
|---|---|---|---|
| `anypoint_apim_mule4` | `anypoint_api_instance` | **Renamed + Schema change** | Official `anypoint_api_instance` is the unified resource for any API-instance type (Mule 3, Mule 4, Flex, proxy, etc.) — no separate `_mule4` or `_flexgateway` resources. Type is selected via attributes on the single resource. |
| `anypoint_apim_flexgateway` | `anypoint_managed_flexgateway` (plus `anypoint_api_instance` for the API side) | **Renamed + Schema change** | Managed Flex Gateway infrastructure is a first-class resource now; API-instance attachment goes through `anypoint_api_instance`. |
| `anypoint_apim_policy_client_id_enforcement` | `anypoint_api_policy_client_id_enforcement` | **Renamed (`apim_` → `api_`)** | |
| `anypoint_apim_policy_jwt_validation` | `anypoint_api_policy_jwt_validation` | **Renamed** | |
| `anypoint_apim_policy_rate_limiting` | `anypoint_api_policy_rate_limiting` | **Renamed** | |
| `anypoint_apim_policy_basic_auth` | `anypoint_api_policy_http_basic_authentication` | **Renamed + Schema change** | Note the longer underscore-separated name. |
| `anypoint_apim_policy_message_logging` | `anypoint_api_policy_message_logging` | **Renamed** | |
| `anypoint_apim_policy_custom` | `anypoint_api_policy` | **Renamed** | Generic custom-policy resource lost the `_custom` suffix. Covers custom/unknown policy types; for known types use the corresponding `anypoint_api_policy_<type>` resource. |
| — | `anypoint_api_policy_<type>` (60+ variants) | **New in official** | The official provider ships strongly-typed resources for every Exchange-listed policy (CORS, IP allow/block, HTTP caching, JWT, OAuth2 introspection, LLM gateway, MCP, A2A, etc.). Prefer these over the generic `anypoint_api_policy`. |
| — | `anypoint_api_instance_sla_tier` | **New in official** | |
| — | `anypoint_api_group` | **New in official** | |
| — | `anypoint_api_group_sla_tier` | **New in official** | |

### CloudHub 2.0 / Networking

| Community resource | Official resource | Status | Notes |
|---|---|---|---|
| `anypoint_private_space` | `anypoint_private_space` | **As-is (schema change)** | Same type name. Attribute set has grown; legacy attributes may have been renamed. Check `docs/resources/private_space.md`. |
| `anypoint_private_space_tlscontext_pem` | `anypoint_tls_context` | **Renamed + Schema change** | Official provider has a single `anypoint_tls_context` resource covering both PEM and JKS inputs. Community's split PEM/JKS resources collapse into one. |
| `anypoint_private_space_tlscontext_jks` | `anypoint_tls_context` | **Renamed + Schema change** | Same. |
| — | `anypoint_private_network` | **New in official** | |
| — | `anypoint_private_space_connection` | **New in official** | |
| — | `anypoint_private_space_association` | **New in official** | |
| — | `anypoint_private_space_upgrade` | **New in official** | |
| — | `anypoint_privatespace_advanced_config` | **New in official** | |
| — | `anypoint_firewall_rules` | **New in official** | |
| — | `anypoint_vpn_connection` | **New in official** | Private-space VPN connection (not the CH1 VPN). |

### CloudHub 1.0 / legacy (Not supported on the official provider)

| Community resource | Official equivalent | Status |
|---|---|---|
| `anypoint_vpc` | — | **Not supported** — CH1 VPCs are out of scope. Manage via the community provider or move these apps to CH2 private spaces. |
| `anypoint_vpn` | — | **Not supported** |
| `anypoint_dlb` | — | **Not supported** |
| `anypoint_amq` | — | **Not supported** (Anypoint MQ not modeled). |
| `anypoint_ame` | — | **Not supported** |
| `anypoint_ame_binding` | — | **Not supported** |
| `anypoint_fabrics` | — | **Not supported** (RTF). |
| `anypoint_fabrics_associations` | — | **Not supported** |
| `anypoint_cloudhub2_shared_space_deployment` | — | **Not supported** |
| `anypoint_rtf_deployment` | — | **Not supported** |

### Secrets Management

Every community `anypoint_secretgroup*` resource renames to `anypoint_secret_group*`
in the official provider (note the added underscore).

| Community resource | Official resource | Status |
|---|---|---|
| `anypoint_secretgroup` | `anypoint_secret_group` | **Renamed** |
| `anypoint_secretgroup_keystore` | `anypoint_secret_group_keystore` | **Renamed** |
| `anypoint_secretgroup_truststore` | `anypoint_secret_group_truststore` | **Renamed** |
| `anypoint_secretgroup_certificate` | `anypoint_secret_group_certificate` | **Renamed** |
| `anypoint_secretgroup_tlscontext_flexgateway` | `anypoint_flex_tls_context` | **Renamed + Schema change** |
| `anypoint_secretgroup_tlscontext_mule` | — | **Not supported** (Mule-runtime TLS context is not in the official provider; only the Flex variant is). |
| `anypoint_secretgroup_tlscontext_securityfabric` | — | **Not supported** |
| `anypoint_secretgroup_crldistrib_cfgs` | — | **Not supported** |
| — | `anypoint_secret_group_shared_secret` | **New in official** |
| — | `anypoint_secret_group_certificate_pinset` | **New in official** |

### Data sources — highlights

Naming discrepancies to be aware of (pattern: community often uses the singular
even for list queries; the official provider distinguishes singular vs plural):

| Community data source | Official equivalent | Notes |
|---|---|---|
| `anypoint_bg` | `anypoint_organization` | Renamed. |
| `anypoint_env` | `anypoint_environment` | Renamed. |
| `anypoint_teams` | `anypoint_team` | Official exposes only the singular-by-id data source today. |
| `anypoint_users` | `anypoint_user` | Same — singular-by-id only. |
| `anypoint_private_spaces` | `anypoint_private_space` | Same — singular-by-id only. |
| `anypoint_apim_instance` | `anypoint_api_instance` | Renamed. |
| `anypoint_apim_instance_policies` | — | Use the `anypoint_api_policy*` resources as the source of truth. |
| `anypoint_secretgroups` | — | Not supported as a data source. |
| `anypoint_flexgateway_registration_token` | — | Not supported. |
| `anypoint_exchange_policy_template*` | — | Not needed — the official provider ships strongly-typed policy resources for every supported policy. |

Data sources unique to the official provider:
`anypoint_firewallrules`, `anypoint_private_space_connection`,
`anypoint_private_space_associations`, `anypoint_private_space_upgrade`,
`anypoint_tls_context`, `anypoint_private_network`,
`anypoint_secret_group_*` (plural variants), `anypoint_managed_flexgateway[s]`,
`anypoint_api_instances`, `anypoint_agent_instances`, `anypoint_mcp_servers`.

## Migration runbook

Work through these steps **in a non-production workspace first**. For complex
estates, migrate one module at a time (e.g. team management first, then orgs,
then API Manager) rather than flipping everything at once.

### Step 1 — snapshot current state

```bash
cp terraform.tfstate terraform.tfstate.pre-migration.$(date +%Y%m%d-%H%M%S).backup
terraform state list > pre-migration.inventory.txt
terraform show -json > pre-migration.state.json
```

Keep these alongside your module for at least one release cycle.

### Step 2 — install the official provider

Extract the distribution tarball for your platform (see
`dist/packages/DISTRIBUTION_README.md`) and run its `install.sh` /
`install.bat`. This drops the binary under
`~/.terraform.d/plugins/sf.com/mulesoft/anypoint/0.1.0/<platform>/`.

Verify:

```bash
ls ~/.terraform.d/plugins/sf.com/mulesoft/anypoint/0.1.0/
```

### Step 3 — run the `scripts/migrate_from_community.sh` inventory

The helper script shipped with this repo reads your current state and
classifies every resource against the compatibility matrix above.

```bash
cd <your-module-dir>
/path/to/anypoint-terraform-provider-poc/scripts/migrate_from_community.sh inventory
```

You'll get a report like:

```
Ready to migrate  : 12 resources (same type name, state replace-provider is enough)
Needs renaming    :  4 resources (moved {} blocks will be emitted)
Needs HCL edits   :  6 resources (schema has changed)
Blocked           :  2 resources (not supported on the official provider)
```

Address the **Blocked** section before moving on: either remove those
resources from the Terraform module (with `terraform state rm` if you want
to keep the Anypoint-side object in place), or split them into a separate
sub-module that stays on the community provider.

### Step 4 — update `required_providers` and re-init

Edit your module:

```hcl
terraform {
  required_providers {
    anypoint = {
      source  = "sf.com/mulesoft/anypoint"
      version = "~> 0.1.0"
    }
  }
}
```

Remove the old `source = "mulesoft-anypoint/anypoint"` entry. Then:

```bash
terraform init -upgrade
```

This will **fail** at the lock step because your state still references
`registry.terraform.io/mulesoft-anypoint/anypoint`. That's expected — the next
step fixes it.

### Step 5 — `state replace-provider`

```bash
terraform state replace-provider \
  -auto-approve \
  registry.terraform.io/mulesoft-anypoint/anypoint \
  sf.com/mulesoft/anypoint
```

Or, equivalently, let the helper script do it with `migrate`:

```bash
/path/to/anypoint-terraform-provider-poc/scripts/migrate_from_community.sh migrate
```

This pass rewrites every resource entry in state to point at the official
provider. Resource *types* are unchanged at this point — only the provider
address.

### Step 6 — apply renames with `moved {}` blocks

For every entry in the **Renamed** column of the matrix, add a `moved {}`
block to your module (the helper script in `inventory` mode prints the exact
blocks you need):

```hcl
moved {
  from = anypoint_bg.acme_root
  to   = anypoint_organization.acme_root
}

moved {
  from = anypoint_env.prod
  to   = anypoint_environment.prod
}

moved {
  from = anypoint_secretgroup.shared
  to   = anypoint_secret_group.shared
}
```

Keep the `moved` blocks in the module for at least one release after the
migration so that any residual state files elsewhere (CI caches, other
workstations) can catch up. You can delete them after you're satisfied all
state files have been rewritten.

### Step 7 — resolve schema drift

Run:

```bash
terraform plan
```

For every diff Terraform reports, cross-reference the per-resource docs under
`docs/resources/<resource>.md`. Typical fixes:

- **Renamed attribute**: `org_id` → `organization_id`, `bg_id` → `organization_id`.
  Edit the HCL.
- **Flat map → nested object**: e.g. community `entitlements = { ... }` → official
  `entitlements { ... }` nested block. Restructure the HCL.
- **Added Required attribute**: usually surfaced as "configuration required but
  not supplied"; set it explicitly.
- **ID format change**: if the official provider reports `id` as different from
  what's in state (e.g. community used `org_id/env_id` compound, official uses
  the bare UUID), `terraform state rm <addr>` followed by
  `terraform import <addr> <new-id>` clears the drift.

When `terraform plan` says "No changes", the migration is complete.

### Step 8 — apply and verify

```bash
terraform apply
```

**Expect zero changes.** If Terraform wants to destroy/create anything after
your drift fixes, stop and investigate before approving. A migration that
results in recreation of live resources is almost always a bug in the HCL fix
(or a genuine `Not supported` you missed in Step 3), not a necessary action.

Post-apply, sanity-check one or two resources manually in the Anypoint
console to confirm they're still the same instances (same IDs, same timestamps).

## Rollback plan

If something goes wrong after Step 5 and you need to back out:

1. Stop any CI that runs `terraform apply`.
2. Restore the state file:
   ```bash
   cp terraform.tfstate.pre-migration.<timestamp>.backup terraform.tfstate
   ```
3. Revert `required_providers` and the HCL edits (or check out the pre-migration
   commit).
4. Run `terraform init -upgrade && terraform plan` — you should be back exactly
   where you started.

Backups are the single most important safety net. Do not start Step 5 without
Step 1.

## Known gotchas

1. **CI / Terraform Cloud / air-gapped runners.** The official provider is
   installed from a private filesystem path (`sf.com/mulesoft/anypoint`), not a
   public registry. Every environment that runs `terraform init` needs the
   tarball installed, or the binary hosted on an internal registry mirror, or
   `plugin_cache_dir` pointing at a shared location. Plan this before flipping
   production.

2. **Organization renames are no longer destructive.** In this provider,
   `anypoint_organization` supports in-place updates for `name` plus the three
   entitlement toggles (`create_sub_orgs`, `create_environments`,
   `global_deployment`) and the quota entitlements. The community provider
   forced recreate on many of these. If your CI approvals are gated on
   "destroy" counts, you'll see those numbers shrink after migration.

3. **Owner ID / parent org ID still force replacement.** Even in the official
   provider, changing `owner_id` or `parent_organization_id` on
   `anypoint_organization` destroys and recreates the org. The upstream API
   doesn't expose ownership transfer / reparenting.

4. **`anypoint_api_policy_custom` is gone.** If your community modules use
   custom-policy resources, map them to either the corresponding
   `anypoint_api_policy_<type>` resource (if the policy type is now
   first-class) or the generic `anypoint_api_policy` resource (for anything
   still custom).

5. **Deprecated `username` / `password` in the community provider.** Because
   they were deprecated and unused, most customers never set them. Do so now
   — the official provider *requires* them for admin operations.

6. **Graceful 404s on SLA tiers.** The official provider treats a 404 from the
   SLA-tier list endpoint as "the parent API instance was deleted out of band"
   and drops the tier from state silently on refresh. This is a change from the
   community provider's behavior of erroring on the refresh. It's the desired
   behaviour for cleanup flows but can surprise users expecting a hard fail.

## Reporting migration issues

If a resource is on the matrix as "mechanically migratable" but your module
ends up needing destroy-and-recreate after the runbook, that's a bug we want
to know about. Please include:

1. The community provider version you migrated from.
2. A minimal reproducer `.tf` file (with secrets redacted) showing the
   resource in question.
3. The output of `scripts/migrate_from_community.sh inventory`.
4. The output of `terraform plan` after Step 7, showing the unexpected diff.
