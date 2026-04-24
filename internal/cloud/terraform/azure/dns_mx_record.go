package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSMXRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_mx_record",
		CoreRFunc: NewDNSMXRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSMXRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSMXRecord{Address: d.Address, Region: d.Region}
	return r
}
