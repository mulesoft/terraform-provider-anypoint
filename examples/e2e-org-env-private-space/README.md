# End-to-End: Sub-Organization, Environments & Private Space

This directory demonstrates the complete Anypoint Platform tenant-onboarding flow using Terraform — from creating a new sub-organization through to a production-ready private network. It uses **two provider aliases** (admin and normal-user) to model real-world credential separation between privileged and standard operations.

## Files

| File | Description |
|------|-------------|
| `suborg_with_privatespace_complete.tf` | Main end-to-end configuration: sub-org → environments → connected app scopes → private space → private network → space association |
| `suborg_with_privatespace_variables.tf` | All input variables (admin credentials, normal-user credentials, org/network config) |
| `suborg_with_privatespace.tfvars.example` | Template — copy to `terraform.tfvars` and fill in your values |
| `scope_validation_example.tf` | Annotated reference for valid `anypoint_connected_app_scopes` values and validation error examples |
| `setup_suborg.sh` | Shell helper script for bootstrapping the setup |
| `DUAL_CREDENTIALS_GUIDE.md` | How to configure and use two provider aliases (admin vs normal-user) |
| `SUBORG_WITH_PRIVATESPACE_GUIDE.md` | Detailed step-by-step guide for the complete flow |
| `SUBORG_EXAMPLE_README.md` | Additional notes on the sub-organization example |
| `SCOPE_VALIDATION.md` | Implementation notes on the built-in scope name validation in `anypoint_connected_app_scopes` |

---

## What Gets Provisioned

```
Parent Organization (existing Salesforce / Anypoint root org)
└── anypoint_organization  "sub_org"               Step 1
    ├── anypoint_environment  "sandbox_suborg"      Step 2
    ├── anypoint_environment  "production_suborg"   Step 2
    ├── anypoint_connected_app_scopes  "app_scopes" Step 3
    ├── anypoint_private_space  "sandbox_space"     Step 4
    │   ├── anypoint_private_network  "sandbox_network"           Step 5
    │   └── anypoint_private_space_association  "sandbox_space_association"  Step 6
```

### Step-by-Step

| Step | Resource | Provider | Description |
|------|----------|----------|-------------|
| 1 | `anypoint_organization` | `anypoint.admin` | Create sub-org with entitlements (vCores, VPCs, network connections) |
| 2 | `anypoint_environment` × 2 | `anypoint.admin` | Create `sandbox` and `production` environments in the sub-org |
| 3 | `anypoint_connected_app_scopes` | `anypoint.admin` | Grant a Connected App the scopes it needs to operate within the new sub-org |
| 4 | `anypoint_private_space` | `anypoint.normal_user` | Create a private space in the sandbox environment |
| 5 | `anypoint_private_network` | `anypoint.normal_user` | Create a private network (VPC) inside the private space |
| 6 | `anypoint_private_space_association` | `anypoint.normal_user` | Associate the private space with all orgs and environments |

---

## Dual-Provider Pattern

This example uses two named provider instances to enforce credential separation:

```hcl
# Admin — user-auth (username + password). Required for org creation and scope assignment.
provider "anypoint" {
  alias         = "admin"
  client_id     = var.anypoint_admin_client_id
  client_secret = var.anypoint_admin_client_secret
  username      = var.anypoint_admin_username
  password      = var.anypoint_admin_password
  auth_type     = "user"
}

# Normal user — connected-app auth. Used for private space and network creation.
provider "anypoint" {
  alias         = "normal_user"
  client_id     = var.anypoint_normal_client_id
  client_secret = var.anypoint_normal_client_secret
  auth_type     = "connected_app"
}
```

> **Single-credential setup**: If you only have admin credentials, replace all `provider = anypoint.normal_user` references with `provider = anypoint.admin` and remove the `normal_user` provider block and its variables. See [DUAL_CREDENTIALS_GUIDE.md](./DUAL_CREDENTIALS_GUIDE.md) for details.

---

## Connected App Scopes Granted

The `anypoint_connected_app_scopes` resource grants the following scopes to the normal-user Connected App so it can operate within the newly created sub-org:

| Scope | Context | Purpose |
|-------|---------|---------|
| `admin:cloudhub` | sub-org | CloudHub 2.0 access |
| `manage:runtime_fabrics` | sub-org | Runtime Fabrics management |
| `manage:cloudhub_networking` | sub-org | Private space / networking management |
| `create:environment` | sub-org | Ability to create new environments |
| `manage:api_configuration` | sub-org + sandbox env | API Manager configuration |
| `manage:apis` | sub-org + sandbox env | API instance management |
| `manage:api_groups` | sub-org | API Group management |
| `manage:api_policies` | sub-org + sandbox env | Policy management |
| `manage:secret_groups` | sub-org + sandbox env | Secrets group management |
| `manage:secrets` | sub-org + sandbox env | Secrets management |

> The provider validates scope names at plan time. Invalid names (typos, wrong separators, non-existent scopes) are rejected before reaching the API. See [SCOPE_VALIDATION.md](./SCOPE_VALIDATION.md) and `scope_validation_example.tf` for the full list of 119 valid scopes and error examples.

---

## Sub-Organization Entitlements

The created sub-org is provisioned with the following entitlement allocation:

| Entitlement | Assigned |
|-------------|----------|
| `create_environments` | `true` |
| `create_sub_orgs` | `false` |
| `global_deployment` | `false` |
| `vcores_production` | 1.0 |
| `vcores_sandbox` | 1.0 |
| `vcores_design` | 0.5 |
| `vpcs` | 1 |
| `network_connections` | 1 |
| `vpns` | 0 |
| `static_ips` | 0 |

Adjust these values in `suborg_with_privatespace_complete.tf` to match your org's available quota. Note: `hybrid`, `runtime_fabric`, and `flex_gateway` flags are master-org-only and are inherited — they cannot be set on sub-orgs.

The sub-org resource uses `lifecycle { ignore_changes = all; prevent_destroy = true }` to protect against accidental drift corrections or deletions.

---

## Quick Start

```bash
# 1. Copy and populate credentials
cp suborg_with_privatespace.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your real values

# 2. Run
terraform init
terraform plan
terraform apply
```

---

## Required Variables

| Variable | Description | Required |
|----------|-------------|----------|
| `anypoint_admin_client_id` | Admin Connected App client ID | Yes |
| `anypoint_admin_client_secret` | Admin Connected App client secret | Yes |
| `anypoint_admin_username` | Admin username (user-auth) | Yes |
| `anypoint_admin_password` | Admin password (user-auth) | Yes |
| `anypoint_normal_client_id` | Normal-user Connected App client ID | Yes* |
| `anypoint_normal_client_secret` | Normal-user Connected App client secret | Yes* |
| `anypoint_base_url` | Anypoint control-plane URL | No (default: stgx) |
| `organization_id` | Parent (root) organization ID | Yes |
| `owner_user_id` | User ID to assign as sub-org owner | Yes |
| `sub_org_name` | Name for the new sub-organization | Yes |
| `connected_app_client_id` | Client ID of the app to assign scopes to | Yes |
| `private_space_region` | AWS region for the private space | Yes |
| `network_cidr_block` | CIDR block for the private network | No (default: `10.111.0.0/16`) |
| `network_reserved_cidrs` | Reserved CIDRs (for VPN/TGW peering) | No (default: `[]`) |

\* If using a single credential set, replace `normal_user` provider references with `admin` and omit these variables.

---

## Outputs

| Output | Description |
|--------|-------------|
| `sub_organization` | Sub-org `id`, `name`, `client_id`, `created_at` |
| `environments` | Sandbox environment `id`, `name`, `client_id`, `arc_namespace` |
| `connected_app_scopes` | Scope assignment resource ID |
| `private_space` | Private space `id`, `name`, `region`, `status`, `deployment_count` |
| `private_network` | Network `id`, `name`, `cidr_block`, `inbound_static_ips`, `outbound_static_ips`, `dns_target` |

---

## Next Steps After Apply

1. **Deploy workloads** — Use the output `environments.sandbox.id` and `sub_organization.id` when creating CloudHub 2.0 applications or API instances in the new sub-org.

2. **Add VPN** — Connect the private network to on-premises infrastructure:
   ```hcl
   resource "anypoint_vpn_connection" "site_to_site" {
     private_space_id = anypoint_private_space.sandbox_space.id
     # ...
   }
   ```

3. **Create additional environments** — Add `anypoint_environment` resources referencing `anypoint_organization.sub_org.id`.

4. **Invite users** — Assign team members to the sub-org's environments using role groups.

5. **Deploy the API stack** — Proceed to the [e2e-apimanagement](../e2e-apimanagement/README.md) example using the sub-org and environment IDs from this run's outputs.

---

## See Also

- [DUAL_CREDENTIALS_GUIDE.md](./DUAL_CREDENTIALS_GUIDE.md) — detailed guide on admin vs normal-user credential separation
- [SUBORG_WITH_PRIVATESPACE_GUIDE.md](./SUBORG_WITH_PRIVATESPACE_GUIDE.md) — extended step-by-step walkthrough
- [SCOPE_VALIDATION.md](./SCOPE_VALIDATION.md) — implementation notes on the scope name validator
- [scope_validation_example.tf](./scope_validation_example.tf) — reference for valid scopes and error examples
- [E2E API Management](../e2e-apimanagement/README.md) — deploy APIs on top of this foundation
- [CloudHub 2.0 Examples](../cloudhub2/README.md) — deploy applications into the private space
