package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getAutomationJobScheduleRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_automation_job_schedule",
		CoreRFunc: NewAutomationJobSchedule,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewAutomationJobSchedule(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.AutomationJobSchedule{Address: d.Address, Region: d.Region}
	return r
}
