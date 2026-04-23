package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAutomationDSCConfigurationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_automation_dsc_configuration",
		CoreRFunc: NewAutomationDSCConfiguration,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewAutomationDSCConfiguration(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AutomationDSCConfiguration{Address: d.Address, Region: d.Region}
	return r
}
