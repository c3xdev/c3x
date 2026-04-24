package google

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputeRegionPerInstanceConfigRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "google_compute_region_per_instance_config",
		NoPrice:             true,
		ReferenceAttributes: []string{"region_instance_group_manager"},
		Notes:               []string{"Free resource."},
	}
}
