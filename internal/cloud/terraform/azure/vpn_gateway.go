package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getVPNGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_vpn_gateway",
		CoreRFunc: newVPNGateway,
	}
}

func newVPNGateway(d *engine.ResourceSpec) engine.CatalogItem {
	v := &azure.VPNGateway{
		Address:    d.Address,
		Region:     d.Get("region").String(),
		ScaleUnits: d.GetInt64OrDefault("scale_unit", 1),
		Type:       "S2S",
	}

	return v
}
