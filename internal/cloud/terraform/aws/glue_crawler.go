package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getGlueCrawlerRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_glue_crawler",
		CoreRFunc: newGlueCrawler,
	}
}

func newGlueCrawler(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	r := &aws.GlueCrawler{
		Address: d.Address,
		Region:  region,
	}

	return r
}
