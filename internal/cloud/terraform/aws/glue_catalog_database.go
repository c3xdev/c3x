package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getGlueCatalogDatabaseRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_glue_catalog_database",
		CoreRFunc: newGlueCatalogDatabase,
	}
}

func newGlueCatalogDatabase(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	r := &aws.GlueCatalogDatabase{
		Address: d.Address,
		Region:  region,
	}

	return r
}
