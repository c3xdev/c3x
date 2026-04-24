package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSAAAARecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_aaaa_record",
		CoreRFunc: NewDNSAAAARecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSAAAARecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSAAAARecord{Address: d.Address, Region: d.Region}
	return r
}
