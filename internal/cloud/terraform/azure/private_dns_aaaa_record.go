package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDNSAAAARecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_aaaa_record",
		CoreRFunc: NewPrivateDNSAAAARecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewPrivateDNSAAAARecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSAAAARecord{Address: d.Address, Region: d.Region}
	return r
}
