package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDNSMXRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_mx_record",
		CoreRFunc: NewPrivateDNSMXRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewPrivateDNSMXRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSMXRecord{Address: d.Address, Region: d.Region}
	return r
}
