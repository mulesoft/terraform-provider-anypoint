# Security Examples — Credential Injection Patterns

These examples demonstrate the recommended ways to feed Anypoint credentials
into the provider **without ever putting them in `.tf` files or committed
`.tfvars`**. Each subdirectory is a self-contained, runnable Terraform config
that picks one credential source and threads it into the `provider "anypoint"`
block.

| Example | When to use it |
|---|---|
| [`aws-secrets-manager/`](./aws-secrets-manager/) | Your team already runs AWS and stores shared secrets in AWS Secrets Manager. |
| [`vault/`](./vault/) | You run HashiCorp Vault (OSS or Enterprise) and use it as the source of truth for service credentials. |
| [`terraform-cloud/`](./terraform-cloud/) | You use Terraform Cloud or Enterprise. Credentials live as sensitive workspace **Environment Variables**. |
| [`github-actions/`](./github-actions/) | You run `terraform apply` from a GitHub Actions pipeline. Credentials live in repo/organization **Actions Secrets**. |

> For the full security guide (storage ranking, SSO behavior, state-file
> hygiene) see [`docs/SECURITY.md`](../../docs/SECURITY.md).

---

## Common rules across all four patterns

1. **Never commit real credentials.** Only `.tfvars.example` and workflow
   YAML that references secret names (not values) is safe to commit.
2. **Use a dedicated, non-SSO Anypoint user** for Terraform automation. The
   "admin" connected app password grant does not work with SSO-federated
   users. See `docs/SECURITY.md` §2 for the full explanation.
3. **Encrypt your remote state backend.** Secrets read through a data source
   land in `terraform.tfstate` in plaintext — identical to the AWS, GCP,
   Kong, and Kubernetes providers. Treat state as sensitive.
4. **Rotate the connected-app client secret and the admin password on a
   schedule** (90-day cadence is typical), regardless of storage mechanism.
