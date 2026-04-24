package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getAzureRMFirewallPolicyRuleCollectionGroupRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "azurerm_firewall_policy_rule_collection_group",
		RFunc:               newAzureRMFirewallPolicyRuleCollectionGroup,
		ReferenceAttributes: []string{"firewall_policy_id"},
	}
}

func newAzureRMFirewallPolicyRuleCollectionGroup(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
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
