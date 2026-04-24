package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getAzureRMFirewallPolicyRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "azurerm_firewall_policy",
		RFunc:               newAzureRMFirewallPolicy,
		ReferenceAttributes: []string{"azurerm_firewall_policy_rule_collection_group.firewall_policy_id"},
	}
}

func newAzureRMFirewallPolicy(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	return &engine.Estimate{
		Name:         d.Address,
		ResourceType: d.Type,
		Tags:         d.Tags,
		DefaultTags:  d.DefaultTags,
		IsSkipped:    true,
		NoPrice:      true,
		SkipMessage:  "Free resource.",
	}
}
