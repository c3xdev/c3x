package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMonitorDataCollectionRuleRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_monitor_data_collection_rule",
		CoreRFunc: newMonitorDataCollectionRule,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newMonitorDataCollectionRule(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.MonitorDataCollectionRule{
		Address: d.Address,
		Region:  region,
	}
}
