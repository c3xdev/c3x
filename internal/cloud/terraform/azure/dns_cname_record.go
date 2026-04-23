package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSCNameRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_cname_record",
		CoreRFunc: NewDNSCNameRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSCNameRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSCNameRecord{Address: d.Address, Region: d.Region}
	return r
}
