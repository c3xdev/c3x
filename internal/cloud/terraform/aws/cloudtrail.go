package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudtrailRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudtrail",
		CoreRFunc: newCloudtrail,
	}
}

func newCloudtrail(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	r := &aws.Cloudtrail{
		Address:                 d.Address,
		Region:                  region,
		IncludeManagementEvents: d.GetBoolOrDefault("include_global_service_events", true),
		IncludeInsightEvents:    len(d.Get("insight_selector").Array()) > 0,
	}

	return r
}
