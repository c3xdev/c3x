package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudfrontFunctionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudfront_function",
		CoreRFunc: newCloudfrontFunction,
	}
}

func newCloudfrontFunction(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Region
	return &aws.CloudfrontFunction{
		Address: d.Address,
		Region:  region,
	}
}
