package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getApplicationInsightsRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_application_insights",
		CoreRFunc: NewApplicationInsights,
	}
}
func NewApplicationInsights(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.ApplicationInsights{
		Address:         d.Address,
		Region:          d.Region,
		RetentionInDays: d.Get("retention_in_days").Int(),
	}
	return r
}
