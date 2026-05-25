package cloudhub2

import (
	"testing"
)

func TestGetPSAssociationAttrTypes(t *testing.T) {
	attrTypes := getPSAssociationAttrTypes()
	required := []string{"id", "organization_id", "environment"}
	for _, key := range required {
		if _, ok := attrTypes[key]; !ok {
			t.Errorf("getPSAssociationAttrTypes() missing key %q", key)
		}
	}
	if len(attrTypes) != len(required) {
		t.Errorf("getPSAssociationAttrTypes() len = %d, want %d", len(attrTypes), len(required))
	}
}
