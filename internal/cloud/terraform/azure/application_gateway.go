package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getApplicationGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_application_gateway",
		CoreRFunc: NewApplicationGateway,
	}
}

func NewApplicationGateway(d *engine.ResourceSpec) engine.CatalogItem {
	var autoscalingMinCapacity *int64
	if d.Get("autoscale_configuration.0.min_capacity").Exists() {
		autoscalingMinCapacity = intPtr(d.Get("autoscale_configuration.0.min_capacity").Int())
	}

	r := &azure.ApplicationGateway{
		Address:                d.Address,
		SKUName:                d.Get("sku.0.name").String(),
		SKUCapacity:            d.Get("sku.0.capacity").Int(),
		AutoscalingMinCapacity: autoscalingMinCapacity,
		Region:                 d.Region,
	}

	return r
}
