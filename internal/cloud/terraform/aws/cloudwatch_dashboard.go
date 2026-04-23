package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudwatchDashboardRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudwatch_dashboard",
		CoreRFunc: NewCloudwatchDashboard,
	}
}
func NewCloudwatchDashboard(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.CloudwatchDashboard{
		Address: d.Address,
	}
	return r
}
