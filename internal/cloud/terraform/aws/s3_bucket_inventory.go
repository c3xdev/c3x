package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getS3BucketInventoryRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_s3_bucket_inventory",
		CoreRFunc: NewS3BucketInventory,
	}
}

func NewS3BucketInventory(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.S3BucketInventory{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
