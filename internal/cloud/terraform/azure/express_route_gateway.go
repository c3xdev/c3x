package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getExpressRouteGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_express_route_gateway",
		CoreRFunc: newExpressRouteGateway,
	}
}

func newExpressRouteGateway(d *engine.ResourceSpec) engine.CatalogItem {
	e := &azure.ExpressRouteGateway{
		Address:    d.Address,
		Region:     d.Get("region").String(),
		ScaleUnits: d.Get("scale_units").Int(),
	}

	return e
}
