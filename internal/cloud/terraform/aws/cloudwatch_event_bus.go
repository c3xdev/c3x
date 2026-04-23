package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudwatchEventBusItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudwatch_event_bus",
		CoreRFunc: NewCloudwatchEventBus,
	}
}
func NewCloudwatchEventBus(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.CloudwatchEventBus{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
