package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getServiceBusNamespaceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_servicebus_namespace",
		CoreRFunc: newServiceBusNamespace,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func newServiceBusNamespace(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &azure.ServiceBusNamespace{
		Address:  d.Address,
		Region:   region,
		SKU:      d.Get("sku").String(),
		Capacity: d.Get("capacity").Int(),
	}
}
