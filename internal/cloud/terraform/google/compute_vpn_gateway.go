package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeVPNGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_vpn_gateway",
		CoreRFunc: NewComputeVPNGateway,
	}
}
func NewComputeVPNGateway(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ComputeVPNGateway{Address: d.Address, Region: d.Get("region").String()}
	return r
}
