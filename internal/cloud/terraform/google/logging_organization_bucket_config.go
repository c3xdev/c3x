package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLoggingOrganizationBucketConfigRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_logging_organization_bucket_config",
		CoreRFunc: NewLoggingOrganizationBucketConfig,
	}
}

func NewLoggingOrganizationBucketConfig(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.Logging{
		Address: d.Address,
	}

	return r
}
