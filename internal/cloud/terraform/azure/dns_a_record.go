package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSARecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_a_record",
		CoreRFunc: NewDNSARecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSARecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSARecord{Address: d.Address, Region: d.Region}
	return r
}
