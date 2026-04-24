package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getVPNGatewayConnectionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_vpn_gateway_connection",
		CoreRFunc: newVPNGatewayConnection,
	}
}

func newVPNGatewayConnection(d *engine.ResourceSpec) engine.CatalogItem {
	v := &azure.VPNGatewayConnection{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}

	return v
}
