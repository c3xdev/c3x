package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDNSCNameRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_cname_record",
		CoreRFunc: NewPrivateDNSCNameRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewPrivateDNSCNameRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSCNameRecord{Address: d.Address, Region: d.Region}
	return r
}
