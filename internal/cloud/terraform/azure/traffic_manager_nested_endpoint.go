package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getTrafficManagerNestedEndpointRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_traffic_manager_nested_endpoint",
		CoreRFunc: newTrafficManagerNestedEndpoint,
		ReferenceAttributes: []string{
			"profile_id",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			if len(d.References("profile_id")) > 0 {
				profile := d.References("profile_id")[0]
				return lookupRegion(profile, []string{"resource_group_name"})
			}

			return ""
		},
	}
}

func newTrafficManagerNestedEndpoint(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	healthCheckInterval := int64(30)
	profileEnabled := false

	if len(d.References("profile_id")) > 0 {
		profile := d.References("profile_id")[0]
		healthCheckInterval = profile.GetInt64OrDefault("monitor_config.0.interval_in_seconds", 30)
		profileEnabled = trafficManagerProfileEnabled(profile)
	}

	return &azure.TrafficManagerEndpoint{
		Address:             d.Address,
		Region:              region,
		ProfileEnabled:      profileEnabled,
		External:            false,
		HealthCheckInterval: healthCheckInterval,
	}
}
