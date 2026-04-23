package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeExternalVPNGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_external_vpn_gateway",
		CoreRFunc: NewComputeExternalVPNGateway,
	}
}
func NewComputeExternalVPNGateway(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ComputeExternalVPNGateway{Address: d.Address, Region: d.Get("region").String()}
	return r
}
