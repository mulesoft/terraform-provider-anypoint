#!/bin/bash
set -e

# Anypoint Terraform Provider - Build Script for Client Distribution
# This script builds the provider for all supported platforms and creates distribution packages

PROVIDER_NAME="terraform-provider-anypoint"
VERSION="0.1.0"
NAMESPACE="sfprod.com/mulesoft/anypoint"
GO=/Users/ankit.sarda/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.5.darwin-arm64/bin/go

echo "🚀 Building Anypoint Terraform Provider for Distribution"
echo "=================================================="
echo "Version: $VERSION"
echo "Namespace: $NAMESPACE"
echo ""

# Clean previous builds
echo "🧹 Cleaning previous builds..."
rm -rf dist/
mkdir -p dist/packages

# Define platforms
PLATFORMS=(
    "windows/amd64"
    "linux/amd64"
    "linux/arm64"
    "darwin/amd64"
    "darwin/arm64"
)

echo "🔨 Building for all platforms..."
for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a platform_parts <<< "$platform"
    GOOS=${platform_parts[0]}
    GOARCH=${platform_parts[1]}
    
    platform_name="${GOOS}_${GOARCH}"
    echo "  Building for $platform_name..."
    
    if [ "$GOOS" = "windows" ]; then
        output_name="${PROVIDER_NAME}_${platform_name}.exe"
    else
        output_name="${PROVIDER_NAME}_${platform_name}"
    fi
    
    CGO_ENABLED=0 GOOS=$GOOS GOARCH=$GOARCH $GO build -o "dist/$output_name" -ldflags="-s -w"
done

echo ""
echo "📦 Creating distribution packages..."

for platform in "${PLATFORMS[@]}"; do
    IFS='/' read -r -a platform_parts <<< "$platform"
    GOOS=${platform_parts[0]}
    GOARCH=${platform_parts[1]}
    platform_name="${GOOS}_${GOARCH}"
    
    echo "  Packaging $platform_name..."
    
    # Create the proper directory structure
    package_dir="dist/packages/$platform_name"
    provider_dir="$package_dir/$NAMESPACE/$VERSION/$platform_name"
    mkdir -p "$provider_dir"
    
    # Copy the binary with the correct name
    if [ "$GOOS" = "windows" ]; then
        cp "dist/${PROVIDER_NAME}_${platform_name}.exe" "$provider_dir/${PROVIDER_NAME}_v${VERSION}.exe"
    else
        cp "dist/${PROVIDER_NAME}_${platform_name}" "$provider_dir/${PROVIDER_NAME}_v${VERSION}"
        chmod +x "$provider_dir/${PROVIDER_NAME}_v${VERSION}"
    fi
    
    # Copy examples and documentation, excluding local Terraform state,
    # downloaded plugins, lock files, and any concrete tfvars (which may
    # contain credentials). Falls back to cp + post-prune if rsync is
    # unavailable.
    echo "    Including examples and documentation..."
    if command -v rsync >/dev/null 2>&1; then
        rsync -a \
            --exclude='terraform.tfstate' \
            --exclude='terraform.tfstate.*' \
            --exclude='.terraform' \
            --exclude='.terraform.lock.hcl' \
            --exclude='terraform.tfvars' \
            --exclude='.DS_Store' \
            --exclude='.claude' \
            --exclude='.cursor' \
            --exclude='.idea' \
            --exclude='.vscode' \
            examples/ "$package_dir/examples/"
    else
        cp -r examples "$package_dir/"
        find "$package_dir/examples" \
            \( -name 'terraform.tfstate' -o -name 'terraform.tfstate.*' \
               -o -name '.terraform.lock.hcl' -o -name '.DS_Store' \) \
            -type f -delete 2>/dev/null || true
        find "$package_dir/examples" \
            \( -name '.terraform' -o -name '.claude' -o -name '.cursor' \
               -o -name '.idea' -o -name '.vscode' \) \
            -type d -exec rm -rf {} + 2>/dev/null || true
        find "$package_dir/examples" -name 'terraform.tfvars' \
            -not -name '*.example' -type f -delete 2>/dev/null || true
    fi

    # Scrub hard-coded credentials / test-account IDs from the copied examples
    # tree. The source examples/ is never modified. Must run BEFORE tar.
    if [ -x "$(dirname "$0")/sanitize_dist_examples.sh" ]; then
        "$(dirname "$0")/sanitize_dist_examples.sh" "$package_dir/examples"
    else
        echo "    ⚠️  sanitize_dist_examples.sh not found; skipping credential scrub" >&2
    fi

    # Copy key documentation files
    cp CLIENT_SOP.md "$package_dir/" 2>/dev/null || true
    cp CLIENT_QUICK_START.md "$package_dir/" 2>/dev/null || true
    cp README.md "$package_dir/" 2>/dev/null || true

    # Ship repo-root docs/ (SECURITY.md, MIGRATION.md, ROADMAP_SECRETS.md,
    # resources/) so customers can read them without cloning the source repo.
    if [ -d docs ]; then
        if command -v rsync >/dev/null 2>&1; then
            rsync -a --exclude='.DS_Store' docs/ "$package_dir/docs/"
        else
            cp -r docs "$package_dir/"
            find "$package_dir/docs" -name '.DS_Store' -type f -delete 2>/dev/null || true
        fi

        # Same credential / UUID scrub applied to examples, against docs/.
        # Belt-and-suspenders — a stale real UUID slipping into a
        # docs/resources/*.md file would otherwise ship unscrubbed.
        if [ -x "$(dirname "$0")/sanitize_dist_examples.sh" ]; then
            "$(dirname "$0")/sanitize_dist_examples.sh" "$package_dir/docs"
        fi
    fi
    
    # Create installation script
    cat > "$package_dir/install.sh" << 'EOF'
#!/bin/bash
# Anypoint Terraform Provider Installation Script

set -e

PROVIDER_DIR="$HOME/.terraform.d/plugins"
TARGET_DIR="$PROVIDER_DIR/sfprod.com/mulesoft/anypoint/0.1.0"

echo "Installing Anypoint Terraform Provider..."

# Detect platform
case "$(uname -s)" in
    Darwin*)    
        case "$(uname -m)" in
            arm64) PLATFORM="darwin_arm64" ;;
            x86_64) PLATFORM="darwin_amd64" ;;
            *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
        esac
        ;;
    Linux*)     
        case "$(uname -m)" in
            x86_64) PLATFORM="linux_amd64" ;;
            aarch64|arm64) PLATFORM="linux_arm64" ;;
            *) echo "Unsupported architecture: $(uname -m)"; exit 1 ;;
        esac
        ;;
    CYGWIN*|MINGW32*|MSYS*|MINGW*)
        PLATFORM="windows_amd64"
        PROVIDER_DIR="$APPDATA/terraform.d/plugins"
        ;;
    *)
        echo "Unsupported operating system: $(uname -s)"
        exit 1
        ;;
esac

# Create target directory
mkdir -p "$TARGET_DIR/$PLATFORM"

# Copy provider binary
if [ -f "sfprod.com/mulesoft/anypoint/0.1.0/$PLATFORM/terraform-provider-anypoint_v0.1.0.exe" ]; then
    cp "sfprod.com/mulesoft/anypoint/0.1.0/$PLATFORM/terraform-provider-anypoint_v0.1.0.exe" "$TARGET_DIR/$PLATFORM/"
    echo "✅ Provider installed successfully at: $TARGET_DIR/$PLATFORM/"
elif [ -f "sfprod.com/mulesoft/anypoint/0.1.0/$PLATFORM/terraform-provider-anypoint_v0.1.0" ]; then
    cp "sfprod.com/mulesoft/anypoint/0.1.0/$PLATFORM/terraform-provider-anypoint_v0.1.0" "$TARGET_DIR/$PLATFORM/"
    chmod +x "$TARGET_DIR/$PLATFORM/terraform-provider-anypoint_v0.1.0"
    echo "✅ Provider installed successfully at: $TARGET_DIR/$PLATFORM/"
else
    echo "❌ Provider binary not found for platform: $PLATFORM"
    exit 1
fi

echo ""
echo "🎉 Installation complete!"
echo "You can now use the provider in your Terraform configurations."
echo ""
echo "Next steps:"
echo "1. Navigate to an example directory (e.g., examples/accessmanagement/environment/)"
echo "2. Copy terraform.tfvars.example to terraform.tfvars"
echo "3. Configure your Anypoint Platform credentials"
echo "4. Run: terraform plan"
EOF
    
    chmod +x "$package_dir/install.sh"
    
    # Create Windows installation script
    cat > "$package_dir/install.bat" << 'EOF'
@echo off
setlocal enabledelayedexpansion

echo Installing Anypoint Terraform Provider for Windows...

set "PROVIDER_DIR=%APPDATA%\terraform.d\plugins"
set "TARGET_DIR=%PROVIDER_DIR%\sfprod.com\mulesoft\anypoint\0.1.0\windows_amd64"

if not exist "%TARGET_DIR%" mkdir "%TARGET_DIR%"

if exist "sfprod.com\mulesoft\anypoint\0.1.0\windows_amd64\terraform-provider-anypoint_v0.1.0.exe" (
    copy "sfprod.com\mulesoft\anypoint\0.1.0\windows_amd64\terraform-provider-anypoint_v0.1.0.exe" "%TARGET_DIR%\"
    echo.
    echo ✅ Provider installed successfully at: %TARGET_DIR%\
    echo.
    echo 🎉 Installation complete!
    echo You can now use the provider in your Terraform configurations.
    echo.
    echo Next steps:
    echo 1. Navigate to an example directory (e.g., examples\accessmanagement\environment\)
    echo 2. Copy terraform.tfvars.example to terraform.tfvars
    echo 3. Configure your Anypoint Platform credentials
    echo 4. Run: terraform plan
) else (
    echo ❌ Provider binary not found for Windows platform
    exit /b 1
)

pause
EOF
    
    # Create README for the package
    cat > "$package_dir/README.md" << EOF
# Anypoint Terraform Provider - Pre-built Binary

This package contains a pre-built Anypoint Terraform Provider binary for $platform_name.

## Quick Installation

### Linux/macOS:
\`\`\`bash
chmod +x install.sh
./install.sh
\`\`\`

### Windows:
Double-click \`install.bat\` or run from Command Prompt:
\`\`\`cmd
install.bat
\`\`\`

## Manual Installation

1. Copy the provider binary to your Terraform plugins directory:
   - **Linux/macOS**: \`~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/$platform_name/\`
   - **Windows**: \`%APPDATA%\\terraform.d\\plugins\\sfprod.com\\mulesoft\\anypoint\\0.1.0\\$platform_name\\\`

2. Make sure the binary is executable (Linux/macOS only):
   \`\`\`bash
   chmod +x ~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/$platform_name/terraform-provider-anypoint_v0.1.0
   \`\`\`

## Verification

After installation, verify the provider is available:
\`\`\`bash
terraform providers
\`\`\`

## What's Included

- Pre-built Terraform provider binary
- Complete examples directory with all use cases
- Installation scripts for easy setup
- Complete documentation (CLIENT_SOP.md, CLIENT_QUICK_START.md)

## Next Steps

1. Navigate to the examples directory: \`cd examples/\`
2. Choose an example that fits your needs:
   - \`examples/accessmanagement/environment/\` - Simple environment management
   - \`examples/accessmanagement/team/\` - Team management
   - \`examples/cloudhub2/privatespace/\` - CloudHub 2.0 private spaces
3. Configure your Anypoint Platform credentials
4. Run \`terraform plan\`

For complete documentation, see CLIENT_SOP.md
EOF
    
    # Create compressed archive with proper directory structure
    cd "dist/packages"
    if command -v tar >/dev/null 2>&1; then
        # Rename directory to have a clean name for extraction
        mv "$platform_name" "anypoint-terraform-provider-${platform_name}"
        tar -czf "anypoint-terraform-provider-${platform_name}.tar.gz" "anypoint-terraform-provider-${platform_name}"
        # Rename back for consistency
        mv "anypoint-terraform-provider-${platform_name}" "$platform_name"
    fi
    cd ../..
done

echo ""
echo "🔐 Generating SHA256SUMS..."
(
    cd dist/packages
    # stable ordering: platform packages first, alphabetical
    shasum -a 256 anypoint-terraform-provider-*.tar.gz > SHA256SUMS
    echo "    dist/packages/SHA256SUMS"
)

echo ""
echo "📝 Writing DISTRIBUTION_README.md..."
BUILD_DATE=$(date -u +%Y-%m-%d)
{
    cat <<'HEADER'
# Anypoint Terraform Provider — Distribution Packages

HEADER
    echo "**Build date:** ${BUILD_DATE}"
    echo "**Version:** ${VERSION}"
    echo "**Provider address:** \`${NAMESPACE}\`"
    echo ""
    echo "## Packages"
    echo ""
    echo "| Platform | Archive | SHA-256 |"
    echo "|----------|---------|---------|"
    while read -r sum file; do
        case "$file" in
            *darwin_arm64*) plat="macOS (Apple Silicon)" ;;
            *darwin_amd64*) plat="macOS (Intel)" ;;
            *linux_amd64*)  plat="Linux (x86_64)" ;;
            *linux_arm64*)  plat="Linux (ARM64)" ;;
            *windows_amd64*) plat="Windows (x86_64)" ;;
            *) plat="$file" ;;
        esac
        echo "| ${plat} | \`${file}\` | \`${sum}\` |"
    done < dist/packages/SHA256SUMS
    cat <<'BODY'

Checksums are also available in `SHA256SUMS`. Verify an archive with:

```bash
shasum -a 256 -c SHA256SUMS --ignore-missing
```

## What's in this build

### Fixes

- **`anypoint_organization` supports `terraform import`.** The resource now
  implements `ResourceWithImportState` and derives the scalar
  `parent_organization_id` from the server's ancestor chain on the first
  post-import `Read`, so `terraform import anypoint_organization.foo <uuid>`
  followed by `terraform plan` produces a clean diff.
- **`anypoint_organization` destroy no longer emits a spurious "Deletion
  Timeout" warning.** Anypoint soft-deletes organizations by stamping
  `deletedAt` on the record instead of returning 404. The deletion poller
  now treats a non-nil `deletedAt` as "deleted" and stops polling immediately.
- **`entitlements` block is now fully optional with defaults.** You can omit
  `entitlements = { … }` entirely and every nested quota/flag defaults to
  `0` / `false`. Individual sub-blocks (`managed_gateway_small`, `vcores_*`,
  `design_center`, etc.) are also optional and hydrate from state on refresh.
- **`anypoint_organization` no longer plans a spurious in-place update on
  re-run.** Inner Optional+Computed attributes (`reassigned`, `enabled`,
  nested `assigned`, Design Center `api`/`mozart`) now carry
  `UseStateForUnknown()` plan modifiers so Terraform keeps the prior value
  when the user's config doesn't specify them.
- **`anypoint_organization` entitlement flatten writes concrete zeros.** When
  the server omits an Optional+Computed entitlement from its response (for
  example after a PUT reset), the flatten helpers now write a concrete
  zero-valued object (`{assigned = 0}`, `{enabled = false}`, …) into state
  rather than a null. A HCL declaration like
  `managed_gateway_large = { assigned = 0 }` therefore matches the refreshed
  state and no longer triggers a spurious in-place update on every plan.
- **`anypoint_environment` drift detection on `name`, `type`, and
  `is_production`.** The `Read` method now surfaces UI-initiated renames and
  classification changes as Terraform drift (previously masked by provider
  state short-circuiting). Debug logging via `tflog` for troubleshooting.
- **`anypoint_organization` data source honors provider-level
  username/password.** The data source now propagates credentials from the
  `provider "anypoint"` block instead of forcing `ANYPOINT_USERNAME` /
  `ANYPOINT_PASSWORD` environment variables. Error messaging covers both
  `ANYPOINT_USERNAME` and `ANYPOINT_ADMIN_USERNAME`.
- **Graceful 404 handling on refresh.** Missing backend resources no longer
  blow up `terraform refresh`; they are silently removed from state.
- **`static_ips` and `vpns` entitlements removed from the schema.** The
  Anypoint Access Management API treats these two entitlements as
  server-managed and does not accept them in the organization create/update
  payload. Configure them through the Anypoint UI or API.

### Features

- **`anypoint_organization` supports in-place updates.** Rename and
  entitlement changes flow through a PUT to the Access Management
  organizations endpoint instead of requiring destroy/recreate.
- **Organization import example.** `examples/accessmanagement/organization/`
  now ships an `import_example.tf` stub and an "Importing an Existing
  Organization" section in its README explaining the end-to-end workflow.

### Packaging

- All example credentials (`client_id`, `client_secret`, `username`,
  `password`, known test-account UUIDs) are replaced with
  `<add-your-value-here>` placeholders in `.tf`, `.tfvars.example`, `.sh`,
  `.bat`, and `.md` files. Scrubbing is driven by
  `scripts/sanitize_dist_examples.sh` and runs automatically during
  `scripts/build_for_distribution.sh`.
- `terraform.tfstate*`, `.terraform/`, `.terraform.lock.hcl`, and
  non-`.example` `terraform.tfvars` files are excluded from the packaged
  examples tree.

## Installation

### Option A — installer script (recommended)

```bash
tar -xzf anypoint-terraform-provider-darwin_arm64.tar.gz
cd anypoint-terraform-provider-darwin_arm64
./install.sh
```

The installer places the provider binary under
`~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/<os_arch>/`, which is
where Terraform looks up the provider via its `required_providers` block.

On Windows, run `install.bat` from the extracted folder.

### Option B — manual install

```bash
tar -xzf anypoint-terraform-provider-darwin_arm64.tar.gz
mkdir -p ~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/darwin_arm64
cp sfprod.com/mulesoft/anypoint/0.1.0/darwin_arm64/terraform-provider-anypoint_v0.1.0 \
   ~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/darwin_arm64/
```

Substitute `darwin_arm64` with your actual platform folder.

## Upgrading from a previous build

If you previously applied an organization with an older build of this
provider, the first plan after upgrading may show a large diff on every
entitlement because state was written as `null` for fields the server omits.
To reconcile:

```bash
terraform init -upgrade
terraform apply -refresh-only
```

After that, `terraform plan` on an unchanged org should report
`No changes. Your infrastructure matches the configuration.`

## Adopting an existing organization

See `examples/accessmanagement/organization/import_example.tf` and the
"Importing an Existing Organization" section of the same directory's
`README.md` for the end-to-end workflow. Summary:

```bash
cd examples/accessmanagement/organization
# Edit import_example.tf with the actual name / parent_organization_id / owner_id
terraform init
terraform import anypoint_organization.imported_org <organization-uuid>
terraform plan   # should be clean
```

## Quick smoke test

```bash
cd examples/accessmanagement/organization
# Edit variables.tf or export TF_VAR_* / ANYPOINT_* env vars with real creds
terraform init
terraform plan
```

Apply once, then `terraform plan` again — you should see
`No changes. Your infrastructure matches the configuration.`

## Recommended Terraform version

- **Terraform 1.6 or later** (1.7.x / 1.8.x tested). The provider is built
  against plugin-framework v1.x which requires Terraform ≥ 1.0, but 1.6+ is
  recommended for the improved plan output and `moved {}` block semantics.

## Provider source configuration

Consumers must reference the provider via its source address:

```hcl
terraform {
  required_providers {
    anypoint = {
      source  = "sfprod.com/mulesoft/anypoint"
      version = "0.1.0"
    }
  }
}
```

## Authentication

All credentials can be supplied via the provider block **or** environment
variables. Env vars take precedence when both are set.

| Provider attribute | Environment variable |
|---|---|
| `client_id` | `ANYPOINT_CLIENT_ID` |
| `client_secret` | `ANYPOINT_CLIENT_SECRET` |
| `username` | `ANYPOINT_USERNAME` or `ANYPOINT_ADMIN_USERNAME` |
| `password` | `ANYPOINT_PASSWORD` or `ANYPOINT_ADMIN_PASSWORD` |
| `base_url` | `ANYPOINT_BASE_URL` |

The `anypoint_organization` resource and data source require username and
password (password grant) in addition to the connected-app client
credentials, because the Access Management API uses user tokens for those
endpoints.

## Migrating from the community provider

See [`docs/MIGRATION.md`](./docs/MIGRATION.md) inside the extracted package
for the compatibility matrix and migration runbook. Pair with
`scripts/migrate_from_community.sh` from the source repo for automated
inventory and state rewriting.

## Security

Before rolling this out to a production config, read:

- [`docs/SECURITY.md`](./docs/SECURITY.md) — credential storage ranking,
  SSO-vs-admin-connected-app guidance, and Terraform state hygiene for PEM
  passphrases / private keys.
- [`examples/security/`](./examples/security/) — runnable credential
  injection patterns for AWS Secrets Manager, HashiCorp Vault, Terraform
  Cloud, and GitHub Actions.
- [`docs/ROADMAP_SECRETS.md`](./docs/ROADMAP_SECRETS.md) — migration plan
  for moving secret-valued attributes (`passphrase`, `key_base64`, …) to
  `WriteOnly` so they stop appearing in state at all.

## Support

File issues or feature requests through your internal MuleSoft
platform-engineering channel. Please include:

1. Provider version (`terraform version`)
2. Terraform CLI version
3. The minimal HCL reproducing the issue
4. `TF_LOG=DEBUG` output (with credentials scrubbed)
BODY
} > dist/packages/DISTRIBUTION_README.md
echo "    dist/packages/DISTRIBUTION_README.md"

echo ""
echo "✅ Build complete!"
echo ""
echo "📋 Distribution Summary:"
echo "======================="
for platform in "${PLATFORMS[@]}"; do
    platform_name=$(echo "$platform" | tr '/' '_')
    echo "  📦 $platform_name: dist/packages/anypoint-terraform-provider-${platform_name}.tar.gz"
done

echo ""
echo "📁 Files created:"
echo "  dist/packages/                     - Individual platform packages"
echo "  dist/packages/*.tar.gz             - Compressed archives for distribution"
echo ""
echo "📦 Each package includes:"
echo "  ✅ Pre-built provider binary"
echo "  ✅ Complete examples directory"
echo "  ✅ Installation scripts"
echo "  ✅ Documentation (SOP, Quick Start, etc.)"
echo ""
echo "🚀 Ready for client distribution!"
echo ""
echo "To test installation:"
echo "  1. Extract a package: tar -xzf dist/packages/anypoint-terraform-provider-darwin_arm64.tar.gz"
echo "  2. Run the installer: ./install.sh"
echo "  3. Navigate to examples and test: cd examples/accessmanagement/environment && terraform plan"