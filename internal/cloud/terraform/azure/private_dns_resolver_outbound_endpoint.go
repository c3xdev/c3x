package azure

import (
	"strings"

	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDnsResolverOutboundEndpointRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_resolver_outbound_endpoint",
		CoreRFunc: newPrivateDnsResolverOutboundEndpoint,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newPrivateDnsResolverOutboundEndpoint(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region

	if strings.HasPrefix(strings.ToLower(region), "usgov") {
		region = "US Gov Zone 1"
	} else if strings.HasPrefix(strings.ToLower(region), "germany") {
		region = "DE Zone 1"
	} else if strings.HasPrefix(strings.ToLower(region), "china") {
		region = "Zone 1 (China)"
	} else {
		region = "Zone 1"
	}

	return &azure.PrivateDnsResolverOutboundEndpoint{
		Address: d.Address,
		Region:  region,
	}
}
