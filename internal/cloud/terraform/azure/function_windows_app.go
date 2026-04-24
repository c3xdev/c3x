package azure

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func geWindowsFunctionAppRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "azurerm_windows_function_app",
		ReferenceAttributes: []string{
			"service_plan_id",
		},
		CoreRFunc: func(d *engine.ResourceSpec) engine.CatalogItem {
			return newFunctionApp(d)
		},
	}
}
