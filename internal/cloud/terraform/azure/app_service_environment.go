package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAppServiceEnvironmentRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_app_service_environment",
		CoreRFunc: NewAppServiceEnvironment,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewAppServiceEnvironment(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AppServiceEnvironment{
		Address:     d.Address,
		Region:      d.Region,
		PricingTier: d.Get("pricing_tier").String(),
	}
	return r
}
