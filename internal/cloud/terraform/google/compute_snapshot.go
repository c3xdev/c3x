package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeSnapshotRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "google_compute_snapshot",
		CoreRFunc:           newComputeSnapshot,
		ReferenceAttributes: []string{"source_disk"},
	}
}

func newComputeSnapshot(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()

	size := computeSnapshotDiskSize(d)

	r := &google.ComputeSnapshot{
		Address:  d.Address,
		Region:   region,
		DiskSize: size,
	}
	return r
}
