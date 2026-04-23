package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getActiveDirectoryDomainServiceReplicaSetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_active_directory_domain_service_replica_set",
		CoreRFunc: NewActiveDirectoryDomainServiceReplicaSet,
		ReferenceAttributes: []string{
			"domain_service_id",
		},
	}
}
func NewActiveDirectoryDomainServiceReplicaSet(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.ActiveDirectoryDomainServiceReplicaSet{
		Address: d.Address,
		Region:  d.Region,
	}
	if len(d.References("domain_service_id")) > 0 {
		r.DomainServiceIDSKU = d.References("domain_service_id")[0].Get("sku").String()
	}
	return r
}
