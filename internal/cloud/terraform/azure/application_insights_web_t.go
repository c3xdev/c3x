package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getApplicationInsightsWebTestRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_application_insights_web_test",
		CoreRFunc: NewApplicationInsightsWebTest,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewApplicationInsightsWebTest(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.ApplicationInsightsWebTest{
		Address: d.Address,
		Region:  d.Region,
		Enabled: d.Get("enabled").Bool(),
		Kind:    d.Get("kind").String(),
	}
	return r
}
