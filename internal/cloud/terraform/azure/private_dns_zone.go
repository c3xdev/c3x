package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSPrivateZoneRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_zone",
		CoreRFunc: NewPrivateDNSZone,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
		Notes: []string{"Most expensive price tier is used."},
	}
}
func NewPrivateDNSZone(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSZone{Address: d.Address, Region: d.Region}
	return r
}
