package cloudhub2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/types"

	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

func TestMapFirewallRulesToAPI(t *testing.T) {
	tests := []struct {
		name     string
		input    []FirewallRuleModel
		wantLen  int
		validate func(t *testing.T, rules []ch2client.FirewallRule)
	}{
		{
			name:    "empty list",
			input:   []FirewallRuleModel{},
			wantLen: 0,
		},
		{
			name: "single rule",
			input: []FirewallRuleModel{
				{
					CidrBlock: types.StringValue("10.0.0.0/8"),
					Protocol:  types.StringValue("TCP"),
					FromPort:  types.Int64Value(80),
					ToPort:    types.Int64Value(80),
					Type:      types.StringValue("INBOUND"),
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, rules []ch2client.FirewallRule) {
				if rules[0].CidrBlock != "10.0.0.0/8" {
					t.Errorf("CidrBlock = %s, want 10.0.0.0/8", rules[0].CidrBlock)
				}
				if rules[0].Protocol != "TCP" {
					t.Errorf("Protocol = %s, want TCP", rules[0].Protocol)
				}
				if rules[0].FromPort != 80 {
					t.Errorf("FromPort = %d, want 80", rules[0].FromPort)
				}
				if rules[0].ToPort != 80 {
					t.Errorf("ToPort = %d, want 80", rules[0].ToPort)
				}
				if rules[0].Type != "INBOUND" {
					t.Errorf("Type = %s, want INBOUND", rules[0].Type)
				}
			},
		},
		{
			name: "multiple rules",
			input: []FirewallRuleModel{
				{
					CidrBlock: types.StringValue("10.0.0.0/8"),
					Protocol:  types.StringValue("TCP"),
					FromPort:  types.Int64Value(443),
					ToPort:    types.Int64Value(443),
					Type:      types.StringValue("INBOUND"),
				},
				{
					CidrBlock: types.StringValue("192.168.0.0/16"),
					Protocol:  types.StringValue("UDP"),
					FromPort:  types.Int64Value(53),
					ToPort:    types.Int64Value(53),
					Type:      types.StringValue("OUTBOUND"),
				},
			},
			wantLen: 2,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapFirewallRulesToAPI(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("mapFirewallRulesToAPI() len = %d, want %d", len(result), tt.wantLen)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestMapFirewallRulesFromAPI(t *testing.T) {
	tests := []struct {
		name     string
		input    []ch2client.FirewallRule
		wantLen  int
		validate func(t *testing.T, rules []FirewallRuleModel)
	}{
		{
			name:    "empty list",
			input:   []ch2client.FirewallRule{},
			wantLen: 0,
		},
		{
			name: "single rule",
			input: []ch2client.FirewallRule{
				{
					CidrBlock: "10.0.0.0/8",
					Protocol:  "TCP",
					FromPort:  443,
					ToPort:    443,
					Type:      "INBOUND",
				},
			},
			wantLen: 1,
			validate: func(t *testing.T, rules []FirewallRuleModel) {
				if rules[0].CidrBlock.ValueString() != "10.0.0.0/8" {
					t.Errorf("CidrBlock = %s, want 10.0.0.0/8", rules[0].CidrBlock.ValueString())
				}
				if rules[0].Protocol.ValueString() != "TCP" {
					t.Errorf("Protocol = %s, want TCP", rules[0].Protocol.ValueString())
				}
				if rules[0].FromPort.ValueInt64() != 443 {
					t.Errorf("FromPort = %d, want 443", rules[0].FromPort.ValueInt64())
				}
				if rules[0].Type.ValueString() != "INBOUND" {
					t.Errorf("Type = %s, want INBOUND", rules[0].Type.ValueString())
				}
			},
		},
		{
			name: "rules are sorted canonically",
			input: []ch2client.FirewallRule{
				{CidrBlock: "10.0.0.0/8", Protocol: "TCP", FromPort: 443, ToPort: 443, Type: "OUTBOUND"},
				{CidrBlock: "10.0.0.0/8", Protocol: "TCP", FromPort: 80, ToPort: 80, Type: "INBOUND"},
			},
			wantLen: 2,
			validate: func(t *testing.T, rules []FirewallRuleModel) {
				// INBOUND < OUTBOUND alphabetically
				if rules[0].Type.ValueString() != "INBOUND" {
					t.Errorf("Expected first rule to be INBOUND after sort, got %s", rules[0].Type.ValueString())
				}
				if rules[1].Type.ValueString() != "OUTBOUND" {
					t.Errorf("Expected second rule to be OUTBOUND after sort, got %s", rules[1].Type.ValueString())
				}
			},
		},
		{
			name: "sort by protocol within same type",
			input: []ch2client.FirewallRule{
				{CidrBlock: "10.0.0.0/8", Protocol: "UDP", FromPort: 53, ToPort: 53, Type: "INBOUND"},
				{CidrBlock: "10.0.0.0/8", Protocol: "TCP", FromPort: 80, ToPort: 80, Type: "INBOUND"},
			},
			wantLen: 2,
			validate: func(t *testing.T, rules []FirewallRuleModel) {
				if rules[0].Protocol.ValueString() != "TCP" {
					t.Errorf("Expected TCP before UDP, got %s", rules[0].Protocol.ValueString())
				}
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mapFirewallRulesFromAPI(tt.input)
			if len(result) != tt.wantLen {
				t.Errorf("mapFirewallRulesFromAPI() len = %d, want %d", len(result), tt.wantLen)
			}
			if tt.validate != nil {
				tt.validate(t, result)
			}
		})
	}
}

func TestFirewallRulesEqual(t *testing.T) {
	rule1 := FirewallRuleModel{
		CidrBlock: types.StringValue("10.0.0.0/8"),
		Protocol:  types.StringValue("TCP"),
		FromPort:  types.Int64Value(80),
		ToPort:    types.Int64Value(80),
		Type:      types.StringValue("INBOUND"),
	}
	rule2 := FirewallRuleModel{
		CidrBlock: types.StringValue("192.168.0.0/16"),
		Protocol:  types.StringValue("TCP"),
		FromPort:  types.Int64Value(443),
		ToPort:    types.Int64Value(443),
		Type:      types.StringValue("INBOUND"),
	}

	tests := []struct {
		name  string
		a, b  []FirewallRuleModel
		equal bool
	}{
		{
			name:  "both empty",
			a:     []FirewallRuleModel{},
			b:     []FirewallRuleModel{},
			equal: true,
		},
		{
			name:  "both nil",
			a:     nil,
			b:     nil,
			equal: true,
		},
		{
			name:  "different lengths",
			a:     []FirewallRuleModel{rule1},
			b:     []FirewallRuleModel{},
			equal: false,
		},
		{
			name:  "same rules",
			a:     []FirewallRuleModel{rule1},
			b:     []FirewallRuleModel{rule1},
			equal: true,
		},
		{
			name:  "different CidrBlock",
			a:     []FirewallRuleModel{rule1},
			b:     []FirewallRuleModel{rule2},
			equal: false,
		},
		{
			name:  "same two rules same order",
			a:     []FirewallRuleModel{rule1, rule2},
			b:     []FirewallRuleModel{rule1, rule2},
			equal: true,
		},
		{
			name:  "same two rules different order",
			a:     []FirewallRuleModel{rule1, rule2},
			b:     []FirewallRuleModel{rule2, rule1},
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := firewallRulesEqual(tt.a, tt.b)
			if result != tt.equal {
				t.Errorf("firewallRulesEqual() = %v, want %v", result, tt.equal)
			}
		})
	}
}

func TestReservedCIDRsEqual(t *testing.T) {
	ctx := context.Background()

	tests := []struct {
		name  string
		a, b  types.List
		equal bool
	}{
		{
			name:  "both null",
			a:     types.ListNull(types.StringType),
			b:     types.ListNull(types.StringType),
			equal: true,
		},
		{
			name: "null vs empty list",
			a:    types.ListNull(types.StringType),
			b:    func() types.List {
				v, _ := types.ListValueFrom(ctx, types.StringType, []string{})
				return v
			}(),
			equal: true,
		},
		{
			name: "same single element",
			a: func() types.List {
				v, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/8"})
				return v
			}(),
			b: func() types.List {
				v, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/8"})
				return v
			}(),
			equal: true,
		},
		{
			name: "different elements",
			a: func() types.List {
				v, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/8"})
				return v
			}(),
			b: func() types.List {
				v, _ := types.ListValueFrom(ctx, types.StringType, []string{"192.168.0.0/16"})
				return v
			}(),
			equal: false,
		},
		{
			name: "different lengths",
			a: func() types.List {
				v, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/8", "192.168.0.0/16"})
				return v
			}(),
			b: func() types.List {
				v, _ := types.ListValueFrom(ctx, types.StringType, []string{"10.0.0.0/8"})
				return v
			}(),
			equal: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := reservedCIDRsEqual(ctx, tt.a, tt.b)
			if result != tt.equal {
				t.Errorf("reservedCIDRsEqual() = %v, want %v", result, tt.equal)
			}
		})
	}
}
