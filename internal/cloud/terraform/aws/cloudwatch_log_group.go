package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudwatchLogGroupItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudwatch_log_group",
		CoreRFunc: NewCloudwatchLogGroup,
	}
}
func NewCloudwatchLogGroup(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.CloudwatchLogGroup{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
