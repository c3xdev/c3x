package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getKinesisAnalyticsV2ApplicationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_kinesisanalyticsv2_application",
		CoreRFunc: NewKinesisAnalyticsV2Application,
		Notes: []string{
			"Terraform doesn’t currently support Analytics Studio, but when it does they will require 2 orchestration KPUs.",
		},
	}
}

func NewKinesisAnalyticsV2Application(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.KinesisAnalyticsV2Application{
		Address:            d.Address,
		Region:             d.Get("region").String(),
		RuntimeEnvironment: d.Get("runtime_environment").String(),
	}
	return r
}
