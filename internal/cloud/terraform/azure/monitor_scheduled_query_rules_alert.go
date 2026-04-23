package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getMonitorScheduledQueryRulesAlertRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_monitor_scheduled_query_rules_alert",
		CoreRFunc: newMonitorScheduledQueryRulesAlert,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newMonitorScheduledQueryRulesAlert(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.MonitorScheduledQueryRulesAlert{
		Address:          d.Address,
		Region:           region,
		Enabled:          d.GetBoolOrDefault("enabled", true),
		TimeSeriesCount:  int64(1),
		FrequencyMinutes: d.Get("frequency").Int(),
	}
}
