package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLoggingProjectSinkRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_logging_project_sink",
		CoreRFunc: NewLoggingProjectSink,
	}
}

func NewLoggingProjectSink(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.Logging{
		Address: d.Address,
	}

	return r
}
