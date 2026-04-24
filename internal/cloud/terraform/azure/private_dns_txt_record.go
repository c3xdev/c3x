package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDNSTXTRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_txt_record",
		CoreRFunc: NewPrivateDNSTXTRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewPrivateDNSTXTRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSTXTRecord{Address: d.Address, Region: d.Region}
	return r
}
