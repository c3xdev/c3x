package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAppServicePlanRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_app_service_plan",
		CoreRFunc: NewAppServicePlan,
	}
}
func NewAppServicePlan(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AppServicePlan{
		Address:     d.Address,
		Region:      d.Region,
		SKUSize:     d.Get("sku.0.size").String(),
		SKUCapacity: d.Get("sku.0.capacity").Int(),
		Kind:        d.Get("kind").String(),
		IsDevTest:   d.ProjectMetadata["isProduction"] == "false",
	}
	return r
}
