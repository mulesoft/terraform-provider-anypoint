# Anypoint Terraform Provider - Quick Start Guide

## 🎯 What You Received

- Pre-built Terraform provider (no Go installation needed!)
- Complete examples directory with all use cases
- Comprehensive documentation (SOP, guides, READMEs)
- Automatic installation scripts

## ⚡ 5-Minute Setup

## 1. Prerequisites

### 1.1 Anypoint Platform Requirements
Before starting, ensure you have:

- ✅ **Active Anypoint Platform Account**
  - Access to your target organization
  - Appropriate permissions for resource management
  - Admin or Organization Administrator role (for advanced features)

- ✅ **Connected App - App acts on its own behalf (grant type = client credentials)**
  - Client ID
  - Client Secret
  - Proper scopes configured

  - **Minimum Required Scopes:**
    ```
    - admin:cloudhub (for CloudHub 2.0 resources)
    - admin:access_controls
    - manage:runtimefabric (if using Runtime Fabric)
    - read:organization
    - edit:organization (for modifying)
    - read:orgenvironments
    - create:environment
    - edit:environment
    - view:environment
    - read:orgusers
    - edit:orgusers
    - profile
    ```

- ✅ **Connected App - App acts on user behalf (grant type = Password)**

    - For some complex operartions like **Creating an organization, Updating the scope of connected app created in the previous step with new organization/environments**, we need to create a connected app on behalf of admin user. Since this involves admin uername and password, we can look at finding a better alternate in future.
    - Admin Username and password
    - Client ID
    - Client Secret

    - **Minimum Required Scopes:**
    ```
    - offline_access
    - full
    ```

### 1.2 Technical Knowledge
- Basic understanding of Terraform concepts
- Familiarity with command-line interfaces
- Understanding of your organization's Anypoint Platform structure

---

## 2. System Requirements

### 2.1 Operating System Support
| OS | Version | Status |
|---|---|---|
| macOS | 10.15+ | ✅ Supported |
| Linux | Ubuntu 18.04+, RHEL 7+ | ✅ Supported |
| Windows | 10+ | ✅ Supported |

### 2.2 Software Dependencies

#### 2.2.1 Terraform    
    - Minimum Version: 1.0
    - Recommended Version: 1.5+
    - Installation: [Download from HashiCorp](https://www.terraform.io/downloads)


### 3: Unpack the tar and Install the Provider

**macOS/Linux:**
```bash
# Extract the package (creates a directory automatically)
tar -xzf anypoint-terraform-provider-[your-platform].tar.gz
cd anypoint-terraform-provider-[your-platform]/

# Install (this will place the provider in the correct location)
chmod +x install.sh
./install.sh
```

**Windows:**
```bash
# Extract using Windows built-in tools or 7-Zip (creates a directory)
# Navigate to the extracted directory, then double-click or run:
install.bat
```

### 4: Verify Installation
```bash
cd examples/accessmanagement/team
terraform providers
# Should show: sf.com/mulesoft/anypoint
```

### 5: Configure Credentials

**Option A: Environment Variables (Recommended - No file editing needed!)**
```bash
# Set your Anypoint Platform credentials once for
# (Connected App acts on its own behalf (client credentials))
export TF_VAR_anypoint_client_id="<client-id>"
export TF_VAR_anypoint_client_secret="<client-secret>"
export TF_VAR_anypoint_base_url="https://stgx.anypoint.mulesoft.com"
```

```bash
# Set your Anypoint Platform credentials once for
# (Connected acts on behalf of admin user)
export TF_VAR_anypoint_admin_client_id="<admin-connected-app-client-id>"
export TF_VAR_anypoint_admin_client_secret="<admin-connected-app-client-secret>"
export ANYPOINT_ADMIN_USERNAME="<admin-username>"
export ANYPOINT_ADMIN_PASSWORD="<admin-password>"
```

**Option B: Configuration File**
```bash
# Copy and edit the variables file
cp terraform.tfvars.example terraform.tfvars
# Edit terraform.tfvars with your actual credentials
```

### Step 6: Test with an Example
```bash
# Navigate to a simple example
cd examples/accessmanagement/environment/

# Test the setup (works with either credential method above)
terraform plan
terraform apply
```

## ✅ Success!

If `terraform plan` runs without errors, you're ready to go!

## 📚 What's Next?

- **Complete Documentation**: See `CLIENT_SOP.md` for detailed instructions
- **Authentication Setup**: Review Section 4 of the SOP for credential management
- **More Examples**: Explore the `examples/` directory for different use cases
- **Troubleshooting**: Check Section 7 of the SOP if you encounter issues

## 🆘 Need Help?

1. **Quick Issues**: Check the troubleshooting section in `CLIENT_SOP.md`
2. **Platform Detection**: Run these commands to identify your platform:
   - **macOS**: `uname -m` (arm64 = Apple Silicon, x86_64 = Intel)
   - **Linux**: `uname -m` (x86_64 = 64-bit, aarch64 = ARM64)
3. **Verification**: Run `terraform providers` to confirm installation

## 📦 Package Contents

```
anypoint-terraform-provider-[platform]/
├── install.sh (or install.bat for Windows)
├── README.md
├── CLIENT_SOP.md                          # Complete documentation
├── CLIENT_QUICK_START.md                  # This guide
├── examples/                              # All example configurations
│   ├── accessmanagement/                  # User, team, environment examples
│   ├── cloudhub2/                         # CloudHub 2.0 examples
│   └── auth_types/                        # Authentication examples
└── sf.com/mulesoft/anypoint/0.1.0/[platform]/
    └── terraform-provider-anypoint_v0.1.0
```