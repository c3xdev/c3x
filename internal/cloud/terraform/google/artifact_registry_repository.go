package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getArtifactRegistryRepositoryRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_artifact_registry_repository",
		CoreRFunc: newArtifactRegistryRepository,
		GetRegion: func(defaultRegion string, d *engine.ResourceSpec) string {
			region := d.Get("region").String()

			zone := d.Get("zone").String()
			if zone != "" {
				region = zoneToRegion(zone)
			}

			location := d.Get("location").String()
			if location != "" {
				region = location
			}

			return region
		},
	}
}

func newArtifactRegistryRepository(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.ArtifactRegistryRepository{
		Address: d.Address,
		Region:  d.Region,
	}

	return r
}
