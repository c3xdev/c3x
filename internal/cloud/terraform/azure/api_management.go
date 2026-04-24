package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAPIManagementRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_api_management",
		CoreRFunc: NewAPIManagement,
		ReferenceAttributes: []string{
			"certificate_id",
		},
	}
}
func NewAPIManagement(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.APIManagement{Address: d.Address, SKUName: d.Get("sku_name").String(), Region: d.Region}
	return r
}
