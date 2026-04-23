package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeMachineImageRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_compute_machine_image",
		CoreRFunc: newComputeMachineImage,
	}
}

func newComputeMachineImage(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()

	r := &google.ComputeMachineImage{
		Address: d.Address,
		Region:  region,
	}
	return r
}
