package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLoggingOrganizationSinkRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_logging_organization_sink",
		CoreRFunc: NewLoggingOrganizationSink,
	}
}

func NewLoggingOrganizationSink(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.Logging{
		Address: d.Address,
	}

	return r
}
