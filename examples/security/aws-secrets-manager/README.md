# AWS Secrets Manager → Anypoint Provider

Pull the Anypoint admin connected-app credentials from AWS Secrets Manager and
feed them into `provider "anypoint"` via a data source. Nothing sensitive ever
lands in `.tf` or `.tfvars`.

## When to use this pattern

- Your team already runs on AWS and has Secrets Manager as the source of
  truth for shared service credentials.
- You want a single shared credential that all team members / pipelines
  consume the same way.
- You are OK with the credential value landing in `terraform.tfstate`
  (plaintext, Terraform-wide design constraint) because you store state in
  S3+KMS with restricted access.

## Prerequisites

1. An AWS Secrets Manager secret containing the Anypoint admin creds as a
   JSON blob. Create it once, out of band:

   ```bash
   aws secretsmanager create-secret \
     --name anypoint/terraform-admin \
     --description "Anypoint admin connected-app creds for Terraform" \
     --secret-string "$(cat <<'EOF'
   {
     "client_id":     "xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx",
     "client_secret": "yyyyyyyyyyyyyyyyyyyyyyyyyyyyyyyy",
     "username":      "terraform-admin@example.com",
     "password":      "<40+-char-random-password>"
   }
   EOF
   )"
   ```

2. An AWS principal (developer user, or CI role) with
   `secretsmanager:GetSecretValue` on that secret, and IAM for the backend
   bucket / DynamoDB lock table.

3. The Anypoint "admin" user referenced in the secret must be a
   **dedicated non-SSO local user** with narrowly-scoped admin roles. See
   [`docs/SECURITY.md`](../../../docs/SECURITY.md) §2 for why.

## Running

```bash
cd examples/security/aws-secrets-manager
terraform init
terraform plan \
  -var 'aws_region=us-east-1' \
  -var 'anypoint_secret_id=anypoint/terraform-admin' \
  -var 'master_organization_id=<your-master-org-uuid>'
```

A clean plan output indicates the wiring is correct.

## State protection reminder

The decoded secret value is written into `terraform.tfstate` as plaintext the
moment the `aws_secretsmanager_secret_version` data source resolves it — this
is the Terraform-wide behavior of every secrets-manager data source (AWS,
GCP, Azure, Vault). Protect state by:

- Using an S3 backend with `encrypt = true` + `kms_key_id`.
- Restricting bucket access to the CI apply role only.
- Enabling Terraform 1.7+ native state encryption for defense-in-depth.
- Never committing `terraform.tfstate` to Git.

See [`docs/SECURITY.md`](../../../docs/SECURITY.md) §3 for the full state
hygiene checklist.
