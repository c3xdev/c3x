package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAutomationDSCNodeConfigurationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_automation_dsc_nodeconfiguration",
		CoreRFunc: NewAutomationDSCNodeConfiguration,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewAutomationDSCNodeConfiguration(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AutomationDSCNodeConfiguration{Address: d.Address, Region: d.Region}
	return r
}
