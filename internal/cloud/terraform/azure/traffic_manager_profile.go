package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
	"strings"
)

func getTrafficManagerProfileRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_traffic_manager_profile",
		CoreRFunc: newTrafficManagerProfile,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newTrafficManagerProfile(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	return &azure.TrafficManagerProfile{
		Address:            d.Address,
		Region:             region,
		Enabled:            trafficManagerProfileEnabled(d),
		TrafficViewEnabled: d.Get("traffic_view_enabled").Bool(),
	}
}

func trafficManagerProfileEnabled(d *engine.ResourceSpec) bool {
	return strings.ToLower(d.GetStringOrDefault("profile_status", "enabled")) == "enabled"
}
