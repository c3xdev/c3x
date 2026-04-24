package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSecurityCenterSubscriptionPricingRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_security_center_subscription_pricing",
		CoreRFunc: newSecurityCenterSubscriptionPricing,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newSecurityCenterSubscriptionPricing(d *engine.ResourceSpec) engine.CatalogItem {
	region := "Global"

	return &azure.SecurityCenterSubscriptionPricing{
		Address:      d.Address,
		Region:       region,
		Tier:         d.GetStringOrDefault("tier", "Free"),
		ResourceType: d.GetStringOrDefault("resource_type", "VirtualMachines"),
	}
}
