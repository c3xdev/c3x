package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getVirtualHubRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_virtual_hub",
		CoreRFunc: newVirtualHub,
	}
}

func newVirtualHub(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	sku := "Basic"
	s := d.Get("sku").String()
	if s != "" {
		sku = s
	}

	v := &azure.VirtualHub{
		Address: d.Address,
		Region:  region,
		SKU:     sku,
	}

	return v
}
