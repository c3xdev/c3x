package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSNSRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_ns_record",
		CoreRFunc: NewDNSNSRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSNSRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSNSRecord{Address: d.Address, Region: d.Region}
	return r
}
