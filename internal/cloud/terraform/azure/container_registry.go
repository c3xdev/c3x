package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getContainerRegistryRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_container_registry",
		CoreRFunc: NewContainerRegistry,
	}
}
func NewContainerRegistry(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.ContainerRegistry{
		Address:                 d.Address,
		Region:                  d.Region,
		GeoReplicationLocations: len(d.Get("georeplications").Array()),
		SKU:                     d.Get("sku").String(),
	}
	return r
}
