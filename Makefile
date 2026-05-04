.PHONY: help build clean test testacc fmt docs build-all package-all

GO ?= /Users/ankit.sarda/go/pkg/mod/golang.org/toolchain@v0.0.1-go1.25.5.darwin-arm64/bin/go

# Default target
help: ## Show this help message
	@echo 'Usage: make [target]'
	@echo ''
	@echo 'Targets:'
	@awk 'BEGIN {FS = ":.*?## "} /^[a-zA-Z_-]+:.*?## / {printf "  %-15s %s\n", $$1, $$2}' $(MAKEFILE_LIST)

build: ## Build the provider for current platform
	$(GO) build -o terraform-provider-anypoint

build-all: ## Build the provider for all platforms
	@echo "Building for all platforms..."
	@mkdir -p dist
	# Windows AMD64
	CGO_ENABLED=0 GOOS=windows GOARCH=amd64 $(GO) build -o dist/terraform-provider-anypoint_windows_amd64.exe
	# Linux AMD64
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 $(GO) build -o dist/terraform-provider-anypoint_linux_amd64
	# Linux ARM64
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 $(GO) build -o dist/terraform-provider-anypoint_linux_arm64
	# macOS AMD64 (Intel)
	CGO_ENABLED=0 GOOS=darwin GOARCH=amd64 $(GO) build -o dist/terraform-provider-anypoint_darwin_amd64
	# macOS ARM64 (Apple Silicon)
	CGO_ENABLED=0 GOOS=darwin GOARCH=arm64 $(GO) build -o dist/terraform-provider-anypoint_darwin_arm64
	@echo "✅ All platforms built successfully in dist/ directory"

package-all: build-all ## Package providers for distribution
	@echo "Packaging providers for distribution..."
	@mkdir -p dist/packages
	# Create structured directories for each platform
	@for platform in windows_amd64 linux_amd64 linux_arm64 darwin_amd64 darwin_arm64; do \
		echo "Packaging $$platform..."; \
		mkdir -p "dist/packages/$$platform/sfprod.com/mulesoft/anypoint/0.1.0/$$platform"; \
		if [ "$$platform" = "windows_amd64" ]; then \
			cp "dist/terraform-provider-anypoint_$$platform.exe" "dist/packages/$$platform/sfprod.com/mulesoft/anypoint/0.1.0/$$platform/terraform-provider-anypoint_v0.1.0.exe"; \
		else \
			cp "dist/terraform-provider-anypoint_$$platform" "dist/packages/$$platform/sfprod.com/mulesoft/anypoint/0.1.0/$$platform/terraform-provider-anypoint_v0.1.0"; \
		fi; \
		cd "dist/packages/$$platform" && tar -czf "../anypoint-terraform-provider-$$platform.tar.gz" .; \
		cd ../../../..; \
	done
	@echo "✅ Packages created in dist/packages/"

clean: ## Clean build artifacts
	rm -f terraform-provider-anypoint
	rm -rf .terraform
	rm -f .terraform.lock.hcl
	rm -rf dist

test: ## Run unit tests
	$(GO) test -v ./...

test-coverage: ## Run unit tests with coverage
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -html=coverage.out -o coverage.html
	$(GO) tool cover -func=coverage.out

test-coverage-ci: ## Run unit tests with coverage for CI
	$(GO) test -v -race -coverprofile=coverage.out -covermode=atomic ./...
	$(GO) tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

testacc: ## Run acceptance tests
	TF_ACC=1 $(GO) test -v ./... -timeout 120m

fmt: ## Format code
	$(GO) fmt ./...
	terraform fmt -recursive ./examples/

docs: ## Generate documentation
	go generate

install: build ## Build and install the provider locally
	mkdir -p ~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/darwin_arm64
	cp terraform-provider-anypoint ~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/darwin_arm64/terraform-provider-anypoint_v0.1.0
	cp terraform-provider-anypoint ~/.terraform.d/plugins/sfprod.com/mulesoft/anypoint/0.1.0/darwin_arm64/terraform-provider-anypoint

deps: ## Download dependencies
	$(GO) mod download
	$(GO) mod tidy

lint: ## Run linter
	golangci-lint run

# Development helpers
dev-setup: deps fmt docs ## Setup development environment

example-team: install ## Run the team example
	cd examples/team && terraform init && terraform plan

example-team-apply: install ## Apply the team example 