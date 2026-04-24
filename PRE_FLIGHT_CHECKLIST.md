# Pre-Flight Checklist for Client Sharing

Use this checklist before sharing the Anypoint Terraform Provider with your client.

## 🔐 Security & Credentials

- [ ] **No hardcoded credentials in non-example files**
- [ ] **All examples have terraform.tfvars.example files**
- [ ] **Real credentials are only in .gitignore'd files**
- [ ] **No organization IDs or sensitive data in documentation**
- [ ] **.gitignore file is present and comprehensive**

## 📚 Documentation

- [ ] **CLIENT_SETUP_GUIDE.md is complete and accurate**
- [ ] **Main README.md is up to date**
- [ ] **All examples have README.md files**
- [ ] **API endpoints and examples are documented**
- [ ] **Prerequisites and installation steps are clear**

## 🧪 Testing & Validation

- [ ] **Provider builds successfully** (`go build`)
- [ ] **No linting errors** (`golangci-lint run`)
- [ ] **At least one example works end-to-end**
- [ ] **Error messages are helpful and clear**
- [ ] **Import functionality works**

## 🛠️ Build & Release

- [ ] **Binary is buildable on client's target platform**
- [ ] **Version is tagged appropriately**
- [ ] **Release notes are prepared (if applicable)**
- [ ] **Dependencies are properly managed** (`go.mod` is clean)

## 🧹 Project Cleanup

- [ ] **Run cleanup script** (`./scripts/clean_for_client.sh`)
- [ ] **No .tfstate files present**
- [ ] **No build artifacts present**
- [ ] **No log files or temporary files**
- [ ] **No IDE-specific files**

## 📁 File Structure

- [ ] **Examples are organized logically**
- [ ] **terraform.tfvars.example in all example directories**
- [ ] **Scripts are executable and documented**
- [ ] **All necessary files are included**

## 🎯 Client Experience

- [ ] **Setup process takes < 10 minutes**
- [ ] **First successful terraform apply works**
- [ ] **Clear error messages for common issues**
- [ ] **Troubleshooting guide is helpful**
- [ ] **Examples demonstrate key functionality**

## 📞 Support Preparation

- [ ] **Support contact information is provided**
- [ ] **Known issues are documented**
- [ ] **Workarounds for common problems are included**
- [ ] **Feedback collection method is established**

## ✅ Final Validation Commands

Run these commands before sharing:

```bash
# 1. Clean the project
./scripts/clean_for_client.sh

# 2. Build the provider
go build -o terraform-provider-anypoint main.go

# 3. Test an example
cd examples/accessmanagement/rolegroup
terraform plan  # Should work with proper credentials

# 4. Verify no sensitive data
grep -r "542cc7e3\|68ef9520\|e5a776d9\|0a5E1fbf" . --exclude-dir=.git || echo "No hardcoded sensitive data found"

# 5. Check documentation
ls -la CLIENT_SETUP_GUIDE.md PRE_FLIGHT_CHECKLIST.md README.md
```

## 📋 Delivery Package

Include these items in your client delivery:

- [ ] **Source code repository or archive**
- [ ] **CLIENT_SETUP_GUIDE.md**
- [ ] **This PRE_FLIGHT_CHECKLIST.md**
- [ ] **Examples with terraform.tfvars.example files**
- [ ] **Any additional documentation or notes**

## 🚀 Post-Delivery

- [ ] **Schedule a setup call with the client**
- [ ] **Provide initial support during first use**
- [ ] **Collect feedback for improvements**
- [ ] **Document any issues found**

---

**✅ When all items are checked, the project is ready for client delivery!**

**📅 Checklist completed by:** _________________ **Date:** _________