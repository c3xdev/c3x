package azure

import (
	"github.com/c3xdev/c3x/internal/catalog/azure"
	"github.com/c3xdev/c3x/internal/engine"
)

func getPointToSiteVpnGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "azurerm_point_to_site_vpn_gateway",
		CoreRFunc: newPointToSiteVpnGateway,
	}
}

func newPointToSiteVpnGateway(d *engine.ResourceSpec) engine.CatalogItem {
	p := &azure.VPNGateway{
		Address:    d.Address,
		Region:     d.Get("region").String(),
		ScaleUnits: d.Get("scale_unit").Int(),
		Type:       "P2S",
	}

	return p
}
