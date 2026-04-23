package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeImageRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "google_compute_image",
		CoreRFunc:           newComputeImage,
		ReferenceAttributes: []string{"source_disk", "source_image", "source_snapshot"},
	}
}

func newComputeImage(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()

	storageSize := computeImageDiskSize(d)

	r := &google.ComputeImage{
		Address:     d.Address,
		Region:      region,
		StorageSize: storageSize,
	}
	return r
}
