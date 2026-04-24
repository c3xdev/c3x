package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSCAARecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_caa_record",
		CoreRFunc: NewDNSCAARecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSCAARecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSCAARecord{Address: d.Address, Region: d.Region}
	return r
}
