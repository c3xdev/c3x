package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMonitorDiagnosticSettingRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_monitor_diagnostic_setting",
		CoreRFunc: newMonitorDiagnosticSetting,
		ReferenceAttributes: []string{
			"target_resource_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"target_resource_id"})
		},
	}
}

func newMonitorDiagnosticSetting(d *engine.ResourceSpec) engine.CatalogItem {
	return &azure.MonitorDiagnosticSetting{
		Address: d.Address,
		Region:  d.Region,

		EventHubTarget:        !d.IsEmpty("eventhub_authorization_rule_id"),
		PartnerSolutionTarget: !d.IsEmpty("partner_solution_id"),
		StorageAccountTarget:  !d.IsEmpty("storage_account_id"),
	}
}
