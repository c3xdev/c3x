package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getVirtualNetworkPeeringRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_virtual_network_peering",
		CoreRFunc: newVirtualNetworkPeering,
		ReferenceAttributes: []string{
			"virtual_network_name",
			"remote_virtual_network_id",
			"resource_group_name",
		},
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			return lookupRegion(d, []string{"virtual_network_name"})
		},
	}
}

func newVirtualNetworkPeering(d *engine.ResourceSpec) engine.CatalogItem {
	sourceRegion := d.Region
	destinationRegion := lookupRegion(d, []string{"remote_virtual_network_id"})

	sourceZone := virtualNetworkPeeringConvertRegion(sourceRegion)
	destinationZone := virtualNetworkPeeringConvertRegion(destinationRegion)

	r := &azure.VirtualNetworkPeering{
		Address:           d.Address,
		DestinationRegion: destinationRegion,
		SourceRegion:      sourceRegion,
		DestinationZone:   destinationZone,
		SourceZone:        sourceZone,
	}
	return r
}

func virtualNetworkPeeringConvertRegion(region string) string {
	zone := regionToVNETZone(region)

	if strings.HasPrefix(strings.ToLower(region), "china") {
		zone = "CN Zone 1"
	}

	return zone
}
