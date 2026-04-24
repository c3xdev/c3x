package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getApplicationInsightsStandardWebTestRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_application_insights_standard_web_test",
		CoreRFunc: newApplicationInsightsStandardWebTest,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newApplicationInsightsStandardWebTest(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.ApplicationInsightsStandardWebTest{
		Address:   d.Address,
		Region:    region,
		Enabled:   d.GetBoolOrDefault("enabled", true),
		Frequency: d.GetInt64OrDefault("frequency", 300),
	}
}
