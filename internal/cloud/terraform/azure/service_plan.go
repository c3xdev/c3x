package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getServicePlanRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "azurerm_service_plan",
		CoreRFunc: func(d *engine.ResourceSpec) engine.CatalogItem {
			return &azure.ServicePlan{
				Address:     d.Address,
				Region:      d.Region,
				SKUName:     d.Get("sku_name").String(),
				WorkerCount: d.GetInt64OrDefault("worker_count", 1),
				OSType:      d.Get("os_type").String(),
				IsDevTest:   d.ProjectMetadata["isProduction"] == "false",
			}
		},
	}
}
