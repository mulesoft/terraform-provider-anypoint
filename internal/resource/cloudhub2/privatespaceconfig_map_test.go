package cloudhub2

import (
	"context"
	"testing"

	"github.com/hashicorp/terraform-plugin-framework/resource"
	"github.com/hashicorp/terraform-plugin-framework/types"

	ch2client "github.com/mulesoft/terraform-provider-anypoint/internal/client/cloudhub2"
)

// --- networkBlockValidator ---

func TestNetworkBlockValidator_Description(t *testing.T) {
	v := &networkBlockValidator{}
	ctx := context.Background()
	desc := v.Description(ctx)
	if desc == "" {
		t.Error("Description() should return non-empty string")
	}
}

func TestNetworkBlockValidator_MarkdownDescription(t *testing.T) {
	v := &networkBlockValidator{}
	ctx := context.Background()
	md := v.MarkdownDescription(ctx)
	if md == "" {
		t.Error("MarkdownDescription() should return non-empty string")
	}
}

func TestPrivateSpaceConfigResource_ConfigValidators(t *testing.T) {
	r := NewPrivateSpaceConfigResource().(*PrivateSpaceConfigResource)
	ctx := context.Background()
	validators := r.ConfigValidators(ctx)
	if len(validators) == 0 {
		t.Error("ConfigValidators() should return at least one validator")
	}
}

// --- mapSpaceConfigToModel ---

func TestMapSpaceConfigToModel(t *testing.T) {
	ctx := context.Background()

	t.Run("nil spaceBase leaves name/status empty", func(t *testing.T) {
		data := &PrivateSpaceConfigResourceModel{}
		mapSpaceConfigToModel(ctx, data, "ps-1", "org-1", nil, nil, nil)
		if data.ID.ValueString() != "ps-1" {
			t.Errorf("ID = %q, want ps-1", data.ID.ValueString())
		}
		if data.OrganizationID.ValueString() != "org-1" {
			t.Errorf("OrganizationID = %q, want org-1", data.OrganizationID.ValueString())
		}
	})

	t.Run("spaceBase populates basic fields", func(t *testing.T) {
		spaceBase := &ch2client.PrivateSpace{
			Name:                    "my-space",
			Status:                  "Running",
			RootOrganizationID:      "root-org",
			MuleAppDeploymentCount:  5,
			DaysLeftForRelaxedQuota: 30,
			VPCMigrationInProgress:  false,
		}
		data := &PrivateSpaceConfigResourceModel{}
		mapSpaceConfigToModel(ctx, data, "ps-1", "org-1", spaceBase, nil, nil)
		if data.Name.ValueString() != "my-space" {
			t.Errorf("Name = %q, want my-space", data.Name.ValueString())
		}
		if data.Status.ValueString() != "Running" {
			t.Errorf("Status = %q, want Running", data.Status.ValueString())
		}
		if data.RootOrganizationID.ValueString() != "root-org" {
			t.Errorf("RootOrganizationID = %q, want root-org", data.RootOrganizationID.ValueString())
		}
		if data.MuleAppDeploymentCount.ValueInt64() != 5 {
			t.Errorf("MuleAppDeploymentCount = %d, want 5", data.MuleAppDeploymentCount.ValueInt64())
		}
		if data.DaysLeftForRelaxedQuota.ValueInt64() != 30 {
			t.Errorf("DaysLeftForRelaxedQuota = %d, want 30", data.DaysLeftForRelaxedQuota.ValueInt64())
		}
	})

	t.Run("networkSrc populates network when data.Network exists", func(t *testing.T) {
		spaceBase := &ch2client.PrivateSpace{Name: "ps", Status: "Running"}
		networkSrc := &ch2client.PrivateSpace{
			Network: ch2client.NetworkConfig{
				Region:   "us-east-1",
				CidrBlock: "10.0.0.0/16",
				DNSTarget: "dns.example.com",
			},
		}
		data := &PrivateSpaceConfigResourceModel{
			Network: &NetworkConfigModel{
				ReservedCIDRs: types.ListNull(types.StringType),
			},
		}
		mapSpaceConfigToModel(ctx, data, "ps-1", "org-1", spaceBase, networkSrc, nil)
		if data.Network == nil {
			t.Fatal("Network should not be nil")
		}
		if data.Network.Region.ValueString() != "us-east-1" {
			t.Errorf("Region = %q, want us-east-1", data.Network.Region.ValueString())
		}
		if data.Network.CidrBlock.ValueString() != "10.0.0.0/16" {
			t.Errorf("CidrBlock = %q, want 10.0.0.0/16", data.Network.CidrBlock.ValueString())
		}
	})

	t.Run("networkSrc auto-allocates network when API returns region and data.Network is nil", func(t *testing.T) {
		networkSrc := &ch2client.PrivateSpace{
			Network: ch2client.NetworkConfig{
				Region:   "eu-west-1",
				CidrBlock: "172.16.0.0/16",
			},
		}
		data := &PrivateSpaceConfigResourceModel{
			Network: nil, // should be auto-allocated
		}
		mapSpaceConfigToModel(ctx, data, "ps-2", "org-2", nil, networkSrc, nil)
		if data.Network == nil {
			t.Fatal("Network should have been auto-allocated")
		}
		if data.Network.Region.ValueString() != "eu-west-1" {
			t.Errorf("Region = %q, want eu-west-1", data.Network.Region.ValueString())
		}
	})

	t.Run("firewall rules synced only when data.FirewallRules is non-nil", func(t *testing.T) {
		firewallSrc := &ch2client.PrivateSpace{
			ManagedFirewallRules: []ch2client.FirewallRule{
				{CidrBlock: "10.0.0.0/8", Protocol: "TCP", FromPort: 443, ToPort: 443, Type: "INBOUND"},
			},
		}
		// Case 1: data.FirewallRules is nil (empty slice is non-nil in Go but len==0 is treated as nil pointer) — check actual semantics
		// Actually FirewallRules is []FirewallRuleModel, not a pointer.
		// The guard in mapSpaceConfigToModel checks `data.FirewallRules != nil` which in Go
		// means the slice itself must be non-nil (not just empty).
		data := &PrivateSpaceConfigResourceModel{FirewallRules: nil}
		mapSpaceConfigToModel(ctx, data, "ps-1", "org-1", nil, nil, firewallSrc)
		if data.FirewallRules != nil {
			t.Error("FirewallRules should remain nil when not managed")
		}

		// Case 2: data.FirewallRules is non-nil slice (even empty) — should sync
		data2 := &PrivateSpaceConfigResourceModel{
			FirewallRules: []FirewallRuleModel{},
		}
		mapSpaceConfigToModel(ctx, data2, "ps-1", "org-1", nil, nil, firewallSrc)
		if len(data2.FirewallRules) != 1 {
			t.Errorf("FirewallRules should be synced, got %v", data2.FirewallRules)
		}
	})
}

// --- PrivateSpaceConfigResource interface compliance ---

func TestPrivateSpaceConfigResource_ImplementsResourceWithConfigValidators(t *testing.T) {
	r := NewPrivateSpaceConfigResource()
	if _, ok := r.(resource.ResourceWithConfigValidators); !ok {
		t.Error("PrivateSpaceConfigResource should implement ResourceWithConfigValidators")
	}
}
