package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeRouterNATRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_router_nat",
		CoreRFunc: NewComputeRouterNAT,
	}
}

func NewComputeRouterNAT(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ComputeRouterNAT{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}

	return r
}
