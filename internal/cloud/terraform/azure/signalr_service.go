package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSignalRServiceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_signalr_service",
		CoreRFunc: newSignalRService,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newSignalRService(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.SignalRService{
		Address:     d.Address,
		Region:      region,
		SkuName:     d.Get("sku.0.name").String(),
		SkuCapacity: d.Get("sku.0.capacity").Int(),
	}
}
