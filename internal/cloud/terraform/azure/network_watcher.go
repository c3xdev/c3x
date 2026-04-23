package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNetworkWatcherRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_network_watcher",
		CoreRFunc: newNetworkWatcher,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newNetworkWatcher(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.NetworkWatcher{
		Address: d.Address,
		Region:  region,
	}
}
