package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNetworkDdosProtectionPlanRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_network_ddos_protection_plan",
		CoreRFunc: newNetworkDdosProtectionPlan,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newNetworkDdosProtectionPlan(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.NetworkDdosProtectionPlan{
		Address: d.Address,
		Region:  region,
	}
}
