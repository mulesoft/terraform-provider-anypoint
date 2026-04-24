package testutil

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/resource/schema"
)

// MockResourceContext creates a mock context for resource testing
func MockResourceContext() context.Context {
	return context.Background()
}

// MockProviderData creates mock provider data for resource tests
func MockProviderData(baseURL string) interface{} {
	// Return a generic map to avoid import cycle
	return map[string]interface{}{
		"base_url":      baseURL,
		"client_id":     "test-client-id",
		"client_secret": "test-client-secret",
		"timeout":       30,
	}
}

// TestResourceMetadata tests resource metadata implementation
func TestResourceMetadata(t *testing.T, res resource.Resource, expectedTypeName string) {
	t.Helper()

	ctx := MockResourceContext()
	req := resource.MetadataRequest{}
	resp := &resource.MetadataResponse{}

	res.Metadata(ctx, req, resp)

	if resp.TypeName != expectedTypeName {
		t.Errorf("Metadata() TypeName = %v, want %v", resp.TypeName, expectedTypeName)
	}
}

// TestResourceSchema tests resource schema implementation
func TestResourceSchema(t *testing.T, res resource.Resource, requiredAttrs, optionalAttrs, computedAttrs []string) {
	t.Helper()

	ctx := MockResourceContext()
	req := resource.SchemaRequest{}
	resp := &resource.SchemaResponse{}

	res.Schema(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Schema() has errors: %v", resp.Diagnostics.Errors())
	}

	// Check required attributes
	for _, attrName := range requiredAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsRequired() {
				t.Errorf("Schema() attribute %s should be required", attrName)
			}
		} else {
			t.Errorf("Schema() missing required attribute: %s", attrName)
		}
	}

	// Check optional attributes
	for _, attrName := range optionalAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsOptional() {
				t.Errorf("Schema() attribute %s should be optional", attrName)
			}
		} else {
			t.Errorf("Schema() missing optional attribute: %s", attrName)
		}
	}

	// Check computed attributes
	for _, attrName := range computedAttrs {
		if attr, exists := resp.Schema.Attributes[attrName]; exists {
			if !attr.IsComputed() {
				t.Errorf("Schema() attribute %s should be computed", attrName)
			}
		} else {
			t.Errorf("Schema() missing computed attribute: %s", attrName)
		}
	}
}

// MockResourceWithConfigure represents a resource that implements Configure
type MockResourceWithConfigure interface {
	resource.Resource
	resource.ResourceWithConfigure
}

// TestResourceConfigure tests resource configure implementation
func TestResourceConfigure(t *testing.T, res MockResourceWithConfigure, providerData interface{}) {
	t.Helper()

	ctx := MockResourceContext()
	req := resource.ConfigureRequest{
		ProviderData: providerData,
	}
	resp := &resource.ConfigureResponse{}

	res.Configure(ctx, req, resp)

	if resp.Diagnostics.HasError() {
		t.Errorf("Configure() has errors: %v", resp.Diagnostics.Errors())
	}
}

// MockResourceState represents a mock state for testing
type MockResourceState struct {
	ID   string
	Name string
}

// MockPlanModifier creates a mock plan modifier for testing
func MockPlanModifier() interface{} {
	// This would return a real plan modifier in practice
	return nil
}

// MockValidator creates a mock validator for testing
func MockValidator() []interface{} {
	// This would return real validators in practice
	return []interface{}{}
}

// AssertStringAttribute validates a string attribute in schema
func AssertStringAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, exists := attrs[name]
	if !exists {
		t.Errorf("Schema missing attribute: %s", name)
		return
	}

	stringAttr, ok := attr.(schema.StringAttribute)
	if !ok {
		t.Errorf("Attribute %s is not a StringAttribute", name)
		return
	}

	if required && !stringAttr.IsRequired() {
		t.Errorf("Attribute %s should be required", name)
	}
	if optional && !stringAttr.IsOptional() {
		t.Errorf("Attribute %s should be optional", name)
	}
	if computed && !stringAttr.IsComputed() {
		t.Errorf("Attribute %s should be computed", name)
	}
}

// AssertBoolAttribute validates a bool attribute in schema
func AssertBoolAttribute(t *testing.T, attrs map[string]schema.Attribute, name string, required, optional, computed bool) {
	t.Helper()

	attr, exists := attrs[name]
	if !exists {
		t.Errorf("Schema missing attribute: %s", name)
		return
	}

	boolAttr, ok := attr.(schema.BoolAttribute)
	if !ok {
		t.Errorf("Attribute %s is not a BoolAttribute", name)
		return
	}

	if required && !boolAttr.IsRequired() {
		t.Errorf("Attribute %s should be required", name)
	}
	if optional && !boolAttr.IsOptional() {
		t.Errorf("Attribute %s should be optional", name)
	}
	if computed && !boolAttr.IsComputed() {
		t.Errorf("Attribute %s should be computed", name)
	}
}

// CreateTestState creates a test state with common attributes
func CreateTestState(id, name string) map[string]interface{} {
	return map[string]interface{}{
		"id":   id,
		"name": name,
	}
}

// ValidateResourceImplementsInterfaces validates that a resource implements expected interfaces
func ValidateResourceImplementsInterfaces(t *testing.T, res interface{}) {
	t.Helper()

	if _, ok := res.(resource.Resource); !ok {
		t.Error("Resource does not implement resource.Resource interface")
	}
}

// MockCreateRequest creates a mock create request for testing
func MockCreateRequest(state map[string]interface{}) resource.CreateRequest {
	// This would create a proper request with state in practice
	return resource.CreateRequest{}
}

// MockUpdateRequest creates a mock update request for testing
func MockUpdateRequest(state, plan map[string]interface{}) resource.UpdateRequest {
	// This would create a proper request with state and plan in practice
	return resource.UpdateRequest{}
}

// MockDeleteRequest creates a mock delete request for testing
func MockDeleteRequest(state map[string]interface{}) resource.DeleteRequest {
	// This would create a proper request with state in practice
	return resource.DeleteRequest{}
}

// MockReadRequest creates a mock read request for testing
func MockReadRequest(state map[string]interface{}) resource.ReadRequest {
	// This would create a proper request with state in practice
	return resource.ReadRequest{}
}
