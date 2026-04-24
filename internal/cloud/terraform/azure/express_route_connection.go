package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getExpressRouteConnectionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "azurerm_express_route_connection",
		RFunc: newExpressRouteConnection,
	}
}

func newExpressRouteConnection(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	e := &azure.ExpressRouteConnection{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}

	return e.BuildResource()
}
