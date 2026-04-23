package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getStorageBucketRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "google_storage_bucket",
		CoreRFunc:           NewStorageBucket,
		ReferenceAttributes: []string{},
	}
}

func NewStorageBucket(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.StorageBucket{
		Address:      d.Address,
		Region:       d.Get("region").String(),
		Location:     d.Get("location").String(),
		StorageClass: d.Get("storage_class").String(),
	}
	return r
}
