package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getActiveDirectoryDomainServiceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_active_directory_domain_service",
		CoreRFunc: NewActiveDirectoryDomainService,
	}
}
func NewActiveDirectoryDomainService(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.ActiveDirectoryDomainService{Address: d.Address, Region: d.Region, SKU: d.Get("sku").String()}
	return r
}
