package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getFederatedIdentityCredentialRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_federated_identity_credential",
		CoreRFunc: newFederatedIdentityCredential,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newFederatedIdentityCredential(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.FederatedIdentityCredential{
		Address: d.Address,
		Region:  region,
	}
}
