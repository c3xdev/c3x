package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDataFactoryRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_data_factory",
		CoreRFunc: newDataFactory,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newDataFactory(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	r := &azure.DataFactory{
		Address: d.Address,
		Region:  region,
	}
	return r
}
