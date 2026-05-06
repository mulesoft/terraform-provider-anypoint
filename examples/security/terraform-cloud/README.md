# Terraform Cloud / Enterprise → Anypoint Provider

Run Terraform plans and applies in Terraform Cloud or Enterprise. The Anypoint
credentials live as **Sensitive Environment Variables** on the workspace and
are never present in HCL, `.tfvars`, or any committed file.

## When to use this pattern

- You already use Terraform Cloud / Enterprise for state storage and remote
  execution.
- You want HashiCorp-managed state encryption at rest, RBAC on workspace
  access, per-run audit logs, and cost-estimation / policy-as-code integration.
- You want Sentinel or OPA policies to gate every apply.

## Prerequisites

1. A Terraform Cloud organization and workspace. Update the `cloud {}` block
   in `main.tf` with your org + workspace names.

2. Workspace execution mode set to **Remote** (default) so plans and applies
   run on HashiCorp's runners.

3. These **Environment Variables** defined on the workspace, each marked
   **Sensitive**:

   | Variable | Value |
   |---|---|
   | `ANYPOINT_CLIENT_ID` | Connected-app client ID |
   | `ANYPOINT_CLIENT_SECRET` | Connected-app client secret |
   | `ANYPOINT_USERNAME` (or `ANYPOINT_ADMIN_USERNAME`) | Dedicated non-SSO admin user |
   | `ANYPOINT_PASSWORD` (or `ANYPOINT_ADMIN_PASSWORD`) | Admin user password |
   | `ANYPOINT_BASE_URL` | e.g. `https://anypoint.mulesoft.com` |

   Set them via the UI (Workspace → Variables → "+ Add variable" → category
   **Environment variable** → tick **Sensitive**) or via the `tfe` provider
   for IaC-managed variables.

4. A **user token** or **team token** for the CLI to authenticate to
   Terraform Cloud. Configure once with `terraform login`.

5. The Anypoint user behind `ANYPOINT_USERNAME` must be a **dedicated non-SSO
   local user** with narrowly-scoped admin roles. See
   [`docs/SECURITY.md`](../../../docs/SECURITY.md) §2.

## Running

```bash
cd examples/security/terraform-cloud

terraform login         # once per workstation
terraform init
terraform plan \
  -var 'master_organization_id=<your-master-org-uuid>'
```

Plans and applies execute on Terraform Cloud runners. The run output redacts
all sensitive variables automatically.

## Why this is the strongest pattern

- **No credentials on disk anywhere.** Sensitive Environment Variables are
  never rendered in the UI after save and are scrubbed from run logs.
- **State encrypted at rest by HashiCorp.**
- **Every read of a sensitive variable is audit-logged.**
- **Policy-as-code gates** (Sentinel / OPA) can enforce rules before apply,
  e.g. "no apply can target the production Anypoint org outside business
  hours" or "no sub-org can be destroyed without two approvers".

This is the same posture the AWS and GCP providers recommend for production
workloads.
