package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudFunctionsRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_cloudfunctions_function",
		CoreRFunc: NewCloudFunctionsFunction,
	}
}

func NewCloudFunctionsFunction(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.CloudFunctionsFunction{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}

	if !d.IsEmpty("available_memory_mb") {
		r.AvailableMemoryMB = intPtr(d.Get("available_memory_mb").Int())
	}

	return r
}
