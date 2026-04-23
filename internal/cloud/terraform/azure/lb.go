package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLoadBalancerRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_lb",
		CoreRFunc: NewLB,
		ReferenceAttributes: []string{
			"resource_group_name",
		},
	}
}

func NewLB(d *engine.ResourceSpec) engine.CatalogItem {
	r := &azure.LB{
		Address: d.Address,
		Region:  d.Region,
		SKU:     d.Get("sku").String(),
	}
	return r
}
