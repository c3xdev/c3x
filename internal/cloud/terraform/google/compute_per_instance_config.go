package google

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getComputePerInstanceConfigRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "google_compute_per_instance_config",
		NoPrice:             true,
		ReferenceAttributes: []string{"instance_group_manager"},
		Notes:               []string{"Free resource."},
	}
}
