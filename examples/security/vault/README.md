# HashiCorp Vault → Anypoint Provider

Pull the Anypoint admin connected-app credentials from a Vault KV v2 secret
and feed them into `provider "anypoint"` via a data source.

## When to use this pattern

- You already run HashiCorp Vault (OSS or Enterprise) as the source of truth
  for service credentials.
- You want short-lived Vault tokens issued via AppRole, Kubernetes auth, or
  OIDC / GitHub Actions federation for the CI runner — so the Terraform run
  never has a long-lived static token on disk.

## Prerequisites

1. Vault is reachable and your client is authenticated. The Vault provider
   reads `VAULT_ADDR` and `VAULT_TOKEN` from the environment by default, or
   you can configure an explicit auth method (AppRole / Kubernetes / OIDC)
   in the `provider "vault" {}` block.

2. A KV v2 secret containing the Anypoint admin creds. Create it once, out
   of band:

   ```bash
   vault kv put secret/anypoint/terraform-admin \
     client_id="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx" \
     client_secret="yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy" \
     username="terraform-admin@example.com" \
     password="<40+-char-random-password>"
   ```

3. A Vault policy granting `read` on that path to whoever / whatever is
   running Terraform. Example HCL policy:

   ```hcl
   path "secret/data/anypoint/terraform-admin" {
     capabilities = ["read"]
   }
   ```

4. The Anypoint "admin" user referenced in the secret must be a
   **dedicated non-SSO local user**. See
   [`docs/SECURITY.md`](../../../docs/SECURITY.md) §2.

## Running

```bash
cd examples/security/vault
export VAULT_ADDR="https://vault.example.com"
export VAULT_TOKEN="$(vault login -method=oidc -format=json | jq -r '.auth.client_token')"

terraform init
terraform plan \
  -var 'vault_mount=secret' \
  -var 'vault_path=anypoint/terraform-admin' \
  -var 'master_organization_id=<your-master-org-uuid>'
```

## State protection reminder

The decoded secret value is written into `terraform.tfstate` as plaintext —
same as the AWS Secrets Manager pattern, same as every Terraform secrets
data source. Use an encrypted remote backend and restrict access. See
[`docs/SECURITY.md`](../../../docs/SECURITY.md) §3.
