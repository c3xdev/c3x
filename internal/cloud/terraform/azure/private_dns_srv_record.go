package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPrivateDNSSRVRecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_private_dns_srv_record",
		CoreRFunc: NewPrivateDNSSRVRecord,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}
func NewPrivateDNSSRVRecord(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.PrivateDNSSRVRecord{Address: d.Address, Region: d.Region}
	return r
}
