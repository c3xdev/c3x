package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPowerBIEmbeddedRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_powerbi_embedded",
		CoreRFunc: newPowerBIEmbedded,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newPowerBIEmbedded(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.PowerBIEmbedded{
		Address: d.Address,
		Region:  region,
		SKU:     d.Get("sku_name").String(),
	}
}
