package accessmanagement

import (
	"testing"
)

// These helpers are package-private functions that build attribute type maps.
// A simple smoke-test verifying the expected keys are present is enough to
// lift them from 0% coverage without duplicating schema tests.

func TestGetSubscriptionAttributeTypes(t *testing.T) {
	at := getSubscriptionAttributeTypes()
	for _, key := range []string{"category", "type", "expiration", "justification"} {
		if _, ok := at[key]; !ok {
			t.Errorf("getSubscriptionAttributeTypes() missing key %q", key)
		}
	}
}

func TestGetOwnerAttributeTypes(t *testing.T) {
	at := getOwnerAttributeTypes()
	required := []string{
		"id", "first_name", "last_name", "email", "username",
		"enabled", "created_at", "updated_at", "organization_id",
		"deleted", "type",
	}
	for _, key := range required {
		if _, ok := at[key]; !ok {
			t.Errorf("getOwnerAttributeTypes() missing key %q", key)
		}
	}
}

func TestGetEnvironmentsAttributeTypes(t *testing.T) {
	at := getEnvironmentsAttributeTypes()
	for _, key := range []string{"id", "name", "organization_id", "is_production", "type", "client_id", "arc_namespace"} {
		if _, ok := at[key]; !ok {
			t.Errorf("getEnvironmentsAttributeTypes() missing key %q", key)
		}
	}
}
