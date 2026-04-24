package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLogicAppIntegrationAccountRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_logic_app_integration_account",
		CoreRFunc: newLogicAppIntegrationAccount,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newLogicAppIntegrationAccount(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	return azure.NewLogicAppIntegrationAccount(d.Address, region, d.GetStringOrDefault("sku_name", "free"))
}
