package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDNSTxtRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_dns_txt_record",
		CoreRFunc: NewDNSTxtRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewDNSTxtRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.DNSTxtRecord{Address: d.Address, Region: d.Region}
	return r
}
