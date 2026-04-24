package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLoggingFolderSinkRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_logging_folder_sink",
		CoreRFunc: NewLoggingFolderSink,
	}
}

func NewLoggingFolderSink(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.Logging{
		Address: d.Address,
	}

	return r
}
