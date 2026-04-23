package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNetworkWatcherFlowLogRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_network_watcher_flow_log",
		CoreRFunc: newNetworkWatcherFlowLog,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newNetworkWatcherFlowLog(d *engine.ResourceSpec) engine.CatalogItem {
	if !d.Get("enabled").Bool() {
		return engine.BlankCoreResource{
			Name: d.Address,
			Type: d.Type,
		}
	}

	trafficAnalyticsEnabled := false
	trafficAnalyticsAcceleratedProcessing := false

	if len(d.Get("traffic_analytics").Array()) > 0 {
		trafficAnalyticsEnabled = d.Get("traffic_analytics.0.enabled").Bool()
		trafficAnalyticsAcceleratedProcessing = d.Get("traffic_analytics.0.interval_in_minutes").Int() == int64(10)
	}

	region := d.Region
	return &azure.NetworkWatcherFlowLog{
		Address:                               d.Address,
		Region:                                region,
		TrafficAnalyticsEnabled:               trafficAnalyticsEnabled,
		TrafficAnalyticsAcceleratedProcessing: trafficAnalyticsAcceleratedProcessing,
	}
}
