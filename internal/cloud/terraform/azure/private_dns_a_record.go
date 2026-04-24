package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDNSARecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_a_record",
		CoreRFunc: NewPrivateDNSARecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewPrivateDNSARecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSARecord{Address: d.Address, Region: d.Region}
	return r
}
