package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getS3BucketAnalyticsConfigurationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_s3_bucket_analytics_configuration",
		CoreRFunc: NewS3BucketAnalyticsConfiguration,
	}
}

func NewS3BucketAnalyticsConfiguration(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.S3BucketAnalyticsConfiguration{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
