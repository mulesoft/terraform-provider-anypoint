# GitHub Actions → Anypoint Provider

Run `terraform plan` / `terraform apply` from a GitHub Actions workflow.
Anypoint credentials live in **GitHub Actions Secrets**; backend credentials
(AWS S3 + KMS in this example) come from **GitHub OIDC federation** so there
are no long-lived cloud keys anywhere.

## When to use this pattern

- Your Terraform config lives in a GitHub-hosted repo.
- You want gated applies on `main` with plan-on-PR for review.
- You want short-lived backend credentials via OIDC federation — no static
  AWS access keys, no `AWS_SECRET_ACCESS_KEY` secret on the repo.

## Prerequisites

### 1. GitHub Actions Secrets

Configure these at the **repository** level (or **organization** level if
you want to share across repos). Settings → Secrets and variables → Actions.

| Secret name | Value |
|---|---|
| `ANYPOINT_CLIENT_ID` | Connected-app client ID |
| `ANYPOINT_CLIENT_SECRET` | Connected-app client secret |
| `ANYPOINT_ADMIN_USERNAME` | Dedicated non-SSO admin user |
| `ANYPOINT_ADMIN_PASSWORD` | Admin user password |
| `AWS_TERRAFORM_ROLE_ARN` | ARN of the IAM role the runner will assume via OIDC |

And one **Variable** (not secret; visible in logs):

| Variable name | Value |
|---|---|
| `ANYPOINT_BASE_URL` | e.g. `https://anypoint.mulesoft.com` |

### 2. AWS OIDC federation for the backend

Trust GitHub's OIDC provider in AWS IAM and create a role scoped to the
state bucket / DynamoDB lock table. HashiCorp has a walk-through:
<https://developer.hashicorp.com/terraform/tutorials/automation/github-actions-oidc>

Summary:

1. Add GitHub's OIDC provider to AWS IAM (one-time per account).
2. Create an IAM role with a trust policy conditional on `sub` and `repo`:

   ```json
   {
     "Version": "2012-10-17",
     "Statement": [{
       "Effect": "Allow",
       "Principal": { "Federated": "arn:aws:iam::<account-id>:oidc-provider/token.actions.githubusercontent.com" },
       "Action": "sts:AssumeRoleWithWebIdentity",
       "Condition": {
         "StringEquals": { "token.actions.githubusercontent.com:aud": "sts.amazonaws.com" },
         "StringLike":   { "token.actions.githubusercontent.com:sub": "repo:<org>/<repo>:ref:refs/heads/main" }
       }
     }]
   }
   ```

3. Grant the role `s3:GetObject`, `s3:PutObject`, `s3:DeleteObject` on the
   state bucket, `kms:Decrypt`, `kms:Encrypt`, `kms:GenerateDataKey` on the
   state key, and `dynamodb:*Item` on the lock table.
4. Put the role ARN in the `AWS_TERRAFORM_ROLE_ARN` secret.

### 3. The Anypoint admin user

Must be a **dedicated non-SSO local user** with narrowly-scoped admin roles.
See [`docs/SECURITY.md`](../../../docs/SECURITY.md) §2.

## Using the workflow

Copy `terraform-apply.yml` into `.github/workflows/` in your Terraform repo
and push. The workflow:

- Runs `terraform plan` on every PR (no apply).
- Runs `terraform apply` only when a PR merges to `main`.
- Auto-redacts every Actions Secret from the log stream.
- Uses short-lived STS credentials issued per-run via OIDC — no static AWS
  keys live in the repo or the GitHub secret store.

## State protection reminder

The decoded Anypoint password is in `terraform.tfstate` in plaintext the
moment it's used in a resource that stores it in state. Use an S3 backend
with SSE-KMS + restricted bucket policy, and enable Terraform 1.7+ native
state encryption for defense-in-depth. See
[`docs/SECURITY.md`](../../../docs/SECURITY.md) §3.

## Sanity check: are secrets actually masked?

Run a quick `echo` test in a scratch workflow:

```yaml
- run: echo "client id tail is ${ANYPOINT_CLIENT_ID: -4}"
  env:
    ANYPOINT_CLIENT_ID: ${{ secrets.ANYPOINT_CLIENT_ID }}
```

GitHub Actions replaces the value with `***` in logs. If you ever see the
real value, the secret was not declared as a Secret — move it immediately.
