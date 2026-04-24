package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSPtrRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_ptr_record",
		CoreRFunc: NewDNSPtrRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSPtrRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSPtrRecord{Address: d.Address, Region: d.Region}
	return r
}
