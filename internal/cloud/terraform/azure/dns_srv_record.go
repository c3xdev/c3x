package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSSrvRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_srv_record",
		CoreRFunc: NewDNSSrvRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSSrvRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSSrvRecord{Address: d.Address, Region: d.Region}
	return r
}
