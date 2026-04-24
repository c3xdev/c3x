package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getContainerRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "google_container_registry",
		CoreRFunc:           NewContainerRegistry,
		ReferenceAttributes: []string{},
	}
}
func NewContainerRegistry(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ContainerRegistry{
		Address:      d.Address,
		Region:       d.Get("region").String(),
		Location:     d.Get("location").String(),
		StorageClass: d.Get("storage_class").String(),
	}
	return r
}
