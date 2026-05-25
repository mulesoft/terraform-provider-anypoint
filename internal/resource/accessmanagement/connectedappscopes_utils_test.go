package accessmanagement

import (
	"testing"

	accessmgmt "github.com/mulesoft/terraform-provider-anypoint/internal/client/accessmanagement"
)

func TestScopeDiff(t *testing.T) {
	tests := []struct {
		name    string
		a       []accessmgmt.Scope
		b       []accessmgmt.Scope
		wantLen int
	}{
		{
			name:    "both empty",
			a:       []accessmgmt.Scope{},
			b:       []accessmgmt.Scope{},
			wantLen: 0,
		},
		{
			name: "a has items b is empty - all in diff",
			a: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{}},
				{Scope: "write:data", ContextParams: map[string]interface{}{}},
			},
			b:       []accessmgmt.Scope{},
			wantLen: 2,
		},
		{
			name: "b has same items - empty diff",
			a: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{}},
			},
			b: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{}},
			},
			wantLen: 0,
		},
		{
			name: "partial overlap",
			a: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{}},
				{Scope: "write:data", ContextParams: map[string]interface{}{}},
			},
			b: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{}},
			},
			wantLen: 1,
		},
		{
			name: "different context params = different scope",
			a: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{"org": "org-1"}},
			},
			b: []accessmgmt.Scope{
				{Scope: "read:data", ContextParams: map[string]interface{}{"org": "org-2"}},
			},
			wantLen: 1,
		},
		{
			name: "b has extra items - ignored",
			a: []accessmgmt.Scope{
				{Scope: "read:data"},
			},
			b: []accessmgmt.Scope{
				{Scope: "read:data"},
				{Scope: "write:data"},
			},
			wantLen: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := scopeDiff(tt.a, tt.b)
			if len(result) != tt.wantLen {
				t.Errorf("scopeDiff() len = %d, want %d (result: %v)", len(result), tt.wantLen, result)
			}
		})
	}
}
