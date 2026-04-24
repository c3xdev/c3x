package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getBigQueryDatasetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_bigquery_dataset",
		CoreRFunc: NewBigQueryDataset,
	}
}

func NewBigQueryDataset(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.BigQueryDataset{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}

	return r
}
