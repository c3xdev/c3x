package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAppConfigurationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_app_configuration",
		CoreRFunc: newAppConfiguration,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newAppConfiguration(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	sku := strings.ToLower(strings.TrimSpace(d.Get("sku").String()))
	if sku == "" {
		sku = "free"
	}
	array := d.Get("replica").Array()
	return &azure.AppConfiguration{
		Address:  d.Address,
		Region:   region,
		SKU:      sku,
		Replicas: int64(len(array)),
	}
}
