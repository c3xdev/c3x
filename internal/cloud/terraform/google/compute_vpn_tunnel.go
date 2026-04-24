package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeVPNTunnelRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_vpn_tunnel",
		CoreRFunc: NewComputeVPNTunnel,
	}
}

func NewComputeVPNTunnel(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ComputeVPNTunnel{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}

	return r
}
