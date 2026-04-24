package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getBastionHostRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_bastion_host",
		CoreRFunc: NewBastionHost,
	}
}
func NewBastionHost(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.BastionHost{Address: d.Address, Region: d.Region}
	return r
}
