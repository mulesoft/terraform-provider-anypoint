package accessmanagement

import (
	"testing"

	accessmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

func TestVCoreEntitlementToModel(t *testing.T) {
	t.Run("nil returns null object", func(t *testing.T) {
		result := vCoreEntitlementToModel(nil)
		if !result.IsNull() {
			t.Errorf("vCoreEntitlementToModel(nil) should return null object")
		}
	})

	t.Run("valid entitlement is converted", func(t *testing.T) {
		vcore := &accessmgmt.VCoreEntitlement{
			Assigned:   10.5,
			Reassigned: 2.5,
		}
		result := vCoreEntitlementToModel(vcore)
		if result.IsNull() {
			t.Fatal("vCoreEntitlementToModel() returned null for valid input")
		}
		attrs := result.Attributes()
		assigned, ok := attrs["assigned"]
		if !ok {
			t.Fatal("Missing 'assigned' attribute")
		}
		_ = assigned
	})

	t.Run("zero values are preserved", func(t *testing.T) {
		vcore := &accessmgmt.VCoreEntitlement{Assigned: 0, Reassigned: 0}
		result := vCoreEntitlementToModel(vcore)
		if result.IsNull() {
			t.Fatal("vCoreEntitlementToModel() returned null for zero values")
		}
	})
}

func TestMqEntitlementToModel(t *testing.T) {
	t.Run("nil returns default object", func(t *testing.T) {
		result := mqEntitlementToModel(nil)
		if result.IsNull() {
			t.Fatal("mqEntitlementToModel(nil) should return default object, not null")
		}
		attrs := result.Attributes()
		if _, hasBase := attrs["base"]; !hasBase {
			t.Error("Result should have 'base' attribute")
		}
		if _, hasAddOn := attrs["add_on"]; !hasAddOn {
			t.Error("Result should have 'add_on' attribute")
		}
	})

	t.Run("valid entitlement is converted", func(t *testing.T) {
		mq := &accessmgmt.MqEntitlement{
			Base:  100,
			AddOn: 50,
		}
		result := mqEntitlementToModel(mq)
		if result.IsNull() {
			t.Fatal("mqEntitlementToModel() returned null for valid input")
		}
	})
}

func TestEnvironmentToModel(t *testing.T) {
	t.Run("nil returns null object", func(t *testing.T) {
		result := environmentToModel(nil)
		if !result.IsNull() {
			t.Error("environmentToModel(nil) should return null object")
		}
	})

	t.Run("valid environment with ArcNamespace is converted", func(t *testing.T) {
		ns := "test-namespace"
		env := &accessmgmt.OrgEnvironment{
			ID:             "env-123",
			Name:           "Production",
			OrganizationID: "org-456",
			IsProduction:   true,
			Type:           "production",
			ClientID:       "client-789",
			ArcNamespace:   &ns,
		}
		result := environmentToModel(env)
		if result.IsNull() {
			t.Fatal("environmentToModel() returned null for valid input")
		}
		attrs := result.Attributes()
		if _, ok := attrs["id"]; !ok {
			t.Error("Missing 'id' attribute")
		}
		if _, ok := attrs["arc_namespace"]; !ok {
			t.Error("Missing 'arc_namespace' attribute")
		}
	})

	t.Run("valid environment without ArcNamespace is converted", func(t *testing.T) {
		env := &accessmgmt.OrgEnvironment{
			ID:             "env-123",
			Name:           "Sandbox",
			OrganizationID: "org-456",
			IsProduction:   false,
			Type:           "sandbox",
			ClientID:       "client-789",
			ArcNamespace:   nil,
		}
		result := environmentToModel(env)
		if result.IsNull() {
			t.Fatal("environmentToModel() returned null for valid input")
		}
		attrs := result.Attributes()
		arcNS, ok := attrs["arc_namespace"]
		if !ok {
			t.Fatal("Missing 'arc_namespace' attribute")
		}
		if !arcNS.IsNull() {
			t.Error("arc_namespace should be null when not provided")
		}
	})
}

func TestVcoreOrZero(t *testing.T) {
	t.Run("nil returns zero vcore object", func(t *testing.T) {
		result := vcoreOrZero(nil)
		if result.IsNull() {
			t.Fatal("vcoreOrZero(nil) should return non-null zero object")
		}
	})

	t.Run("non-nil is passed through", func(t *testing.T) {
		vcore := &accessmgmt.VCoreEntitlement{Assigned: 5, Reassigned: 1}
		result := vcoreOrZero(vcore)
		if result.IsNull() {
			t.Fatal("vcoreOrZero() returned null for valid input")
		}
	})
}
