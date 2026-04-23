package google

import (
	"github.com/c3xdev/c3x/internal/catalog/google"
	"github.com/c3xdev/c3x/internal/engine"
)

func getBigQueryTableRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "google_bigquery_table",
		CoreRFunc: NewBigQueryTable,
	}
}

func NewBigQueryTable(d *engine.ResourceSpec) engine.CatalogItem {
	r := &google.BigQueryTable{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}

	return r
}
