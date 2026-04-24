package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getKinesisAnalyticsApplicationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_kinesis_analytics_application",
		CoreRFunc: NewKinesisAnalyticsApplication,
	}
}

func NewKinesisAnalyticsApplication(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.KinesisAnalyticsApplication{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
