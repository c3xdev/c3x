package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSZoneRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_zone",
		CoreRFunc: NewDNSZone,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
		Notes: []string{"Most expensive price tier is used."},
	}
}

func NewDNSZone(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSZone{
		Address: d.Address,
		Region:  d.Region,
	}

	return r
}
