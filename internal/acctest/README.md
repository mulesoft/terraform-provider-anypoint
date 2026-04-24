# Acceptance Test Utilities

This directory contains shared utilities and helpers for acceptance testing of the Anypoint Terraform provider.

## Test Organization

Following Terraform provider best practices, our tests are organized as follows:

### **Test Locations**
- **Unit tests**: Located alongside their source files (e.g., `internal/client/*/`)  
- **Integration tests**: Located alongside resources (e.g., `internal/resource/*/`)
- **Acceptance tests**: Located alongside resources (e.g., `internal/resource/*/`)
- **Shared utilities**: Located in `internal/acctest/` (this directory)

### **Test Naming Conventions**
- **Unit tests**: `Test*` (e.g., `TestAnypointClient_GetUser`)
- **Integration tests**: `TestIntegration*` (e.g., `TestIntegrationEnvironmentResource_CRUD`)  
- **Acceptance tests**: `TestAcc*` (e.g., `TestAccEnvironmentResource_basic`)

### **File Naming Conventions**
- **Unit tests**: `*_test.go` alongside source files
- **Integration tests**: `*_integration_test.go` alongside resources
- **Acceptance tests**: `*_acc_test.go` alongside resources

## Shared Utilities

This package provides:

- **`TestAccProtoV6ProviderFactories`**: Provider factory for acceptance tests
- **`testAccPreCheck()`**: Pre-test validation of required environment variables
- **`testAccProviderConfig()`**: Standard provider configuration
- **`createTestClient()`**: Creates AnypointClient for testing
- **`createUserTestClient()`**: Creates UserAnypointClient for testing
- **Helper functions**: Resource naming, destroy checks, etc.

## Environment Variables

Set these environment variables for acceptance tests:

```bash
export ANYPOINT_CLIENT_ID="your-client-id"
export ANYPOINT_CLIENT_SECRET="your-client-secret"
export ANYPOINT_CONTROL_PLANE_URL="https://anypoint.mulesoft.com"
```

## Running Tests

### Unit Tests
```bash
# Run all unit tests
go test ./internal/client/... ./internal/datasource/... ./internal/resource/...

# Run specific unit tests  
go test ./internal/client/accessmanagement/ -v
```

### Integration Tests
```bash
# Run integration tests for specific resource
go test ./internal/resource/accessmanagement/ -run TestIntegration -v

# Run all integration tests
go test ./internal/resource/... -run TestIntegration -v
```

### Acceptance Tests  
```bash
# Run acceptance tests (requires environment variables)
TF_ACC=1 go test ./internal/resource/accessmanagement/ -run TestAcc -v

# Run specific acceptance test
TF_ACC=1 go test ./internal/resource/accessmanagement/ -run TestAccEnvironmentResource_basic -v
```

### Coverage
```bash
# Generate coverage report for all tests
go test -cover ./...

# Generate detailed coverage report
go test -coverprofile=coverage.out ./... && go tool cover -html=coverage.out
```

## Test Structure Examples

### Unit Test
```go
func TestAnypointClient_GetUser(t *testing.T) {
    // Test individual client methods with mocked HTTP responses
}
```

### Integration Test  
```go
func TestIntegrationEnvironmentResource_CRUD(t *testing.T) {
    // Test full resource lifecycle with mocked APIs
}
```

### Acceptance Test
```go
func TestAccEnvironmentResource_basic(t *testing.T) {
    resource.Test(t, resource.TestCase{
        PreCheck:                 func() { testAccPreCheck(t) },
        ProtoV6ProviderFactories: TestAccProtoV6ProviderFactories,
        Steps: []resource.TestStep{
            // Test against real APIs (when TF_ACC=1)
        },
    })
}
```