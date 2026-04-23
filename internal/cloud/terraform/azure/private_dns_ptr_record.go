package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDNSPTRRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_ptr_record",
		CoreRFunc: NewPrivateDNSPTRRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewPrivateDNSPTRRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSPTRRecord{Address: d.Address, Region: d.Region}
	return r
}
