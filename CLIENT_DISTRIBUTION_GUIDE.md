# Anypoint Terraform Provider - Client Distribution Guide

## 📦 Pre-built Provider Distribution

This document explains how to distribute pre-built Terraform provider binaries to clients, eliminating the need for Go installation and compilation.

---

## 🎯 Distribution strategy

### Pre-built Binary Packages
- **What**: Platform-specific compiled binaries with installation scripts
- **Client Requirements**: Only Terraform (no Go needed)
- **Distribution**: Compressed archives via email, cloud storage, or file sharing

---

## 🚀 Pre-built Binary Distribution (Quick Start)

### Step 1: Build All Platform Binaries

```bash
# Build for all supported platforms
make package-all

# Or use the distribution script
./scripts/build_for_distribution.sh
```

This creates:
```
dist/packages/
├── anypoint-terraform-provider-darwin_amd64.tar.gz    # macOS Intel
├── anypoint-terraform-provider-darwin_arm64.tar.gz    # macOS Apple Silicon
├── anypoint-terraform-provider-linux_amd64.tar.gz     # Linux 64-bit
├── anypoint-terraform-provider-linux_arm64.tar.gz     # Linux ARM64
└── anypoint-terraform-provider-windows_amd64.tar.gz   # Windows 64-bit
```

### Step 2: Distribute to Clients

#### Method A: Email/File Sharing
1. Send the appropriate platform package to each client
2. Include the simplified installation instructions below

#### Method B: Cloud Storage
1. Upload packages to cloud storage (S3, Google Drive, Dropbox, etc.)
2. Share download links with clients
3. Include platform-specific download instructions

#### Method C: Internal File Server
1. Host packages on your internal file server
2. Provide download URLs to clients

---

## 📋 Client Installation Instructions

### For macOS Users

1. **Download the package** for your Mac type:
   - **Apple Silicon (M1/M2)**: `anypoint-terraform-provider-darwin_arm64.tar.gz`
   - **Intel**: `anypoint-terraform-provider-darwin_amd64.tar.gz`

2. **Extract and install**:
   ```bash
   # Extract the package (creates anypoint-terraform-provider-darwin_arm64/ directory)
   tar -xzf anypoint-terraform-provider-darwin_arm64.tar.gz
   cd anypoint-terraform-provider-darwin_arm64/
   
   # Run the installer
   chmod +x install.sh
   ./install.sh
   ```

3. **Verify installation**:
   ```bash
   # Check if Terraform can find the provider
   terraform providers
   ```

### For Linux Users

1. **Download the package** for your architecture:
   - **64-bit**: `anypoint-terraform-provider-linux_amd64.tar.gz`
   - **ARM64**: `anypoint-terraform-provider-linux_arm64.tar.gz`

2. **Extract and install**:
   ```bash
   # Extract the package (creates anypoint-terraform-provider-linux_amd64/ directory)
   tar -xzf anypoint-terraform-provider-linux_amd64.tar.gz
   cd anypoint-terraform-provider-linux_amd64/
   
   # Run the installer
   chmod +x install.sh
   ./install.sh
   ```

### For Windows Users

1. **Download**: `anypoint-terraform-provider-windows_amd64.tar.gz`

2. **Extract**: Use Windows built-in extraction or 7-Zip (creates `anypoint-terraform-provider-windows_amd64/` directory)

3. **Install**: 
   - Navigate to the extracted directory
   - Double-click `install.bat`
   - Or run from Command Prompt: `install.bat`

---

## 🛠️ Internal Build Process

### For Your Team (Provider Maintainers)

1. **Automated Build Script**:
   ```bash
   # Build all platforms
   ./scripts/build_for_distribution.sh
   
   # Packages are created in dist/packages/
   ls -la dist/packages/
   ```

2. **Quality Assurance**:
   ```bash
   # Test on different platforms
   # Extract and test installation script
   tar -xzf dist/packages/anypoint-terraform-provider-darwin_arm64.tar.gz
   cd anypoint-terraform-provider-darwin_arm64/
   ./install.sh
   
   # Test with examples
   cd examples/accessmanagement/environment/
   terraform plan
   terraform providers  # Should show the anypoint provider
   ```

3. **Distribution Checklist**:
   - [ ] All platforms built successfully
   - [ ] Installation scripts tested on target platforms
   - [ ] Examples work with pre-built provider
   - [ ] Documentation updated
   - [ ] Client communication prepared

---

## 🔍 Troubleshooting

### Common Issues

**Problem**: "Provider not found" error
**Solution**: 
- Verify installation script ran successfully
- Check provider is in correct directory
- Run `terraform providers` to list available providers

**Problem**: Permission denied (Linux/macOS)
**Solution**:
- Make sure binary is executable: `chmod +x ~/.terraform.d/plugins/sf.com/mulesoft/anypoint/0.1.0/*/terraform-provider-anypoint_v0.1.0`

**Problem**: Windows installation issues
**Solution**:
- Run install.bat as Administrator
- Check Windows Defender/antivirus isn't blocking the binary

### Verification Commands

```bash
# Check provider installation
terraform providers

# Verify specific provider
ls -la ~/.terraform.d/plugins/sf.com/mulesoft/anypoint/0.1.0/

# Test with example
cd examples/accessmanagement/environment/
terraform plan  # Should work without Go installation
```

---

## 📞 Support

When clients have issues:
1. Ask for their operating system and architecture
2. Verify they downloaded the correct package
3. Check installation script output
4. Validate with `terraform providers` command

This approach eliminates Go installation requirements while maintaining a professional, easy-to-use experience for your clients.