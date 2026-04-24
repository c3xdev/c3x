package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getSubnetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_subnet",
		RFunc:               NewSubnet,
		ReferenceAttributes: []string{"aws_nat_gateway.subnet_id"},
	}
}

func NewSubnet(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
	return &engine.Estimate{
		Name:         d.Address,
		ResourceType: d.Type,
		Tags:         d.Tags,
		DefaultTags:  d.DefaultTags,
		IsSkipped:    true,
		NoPrice:      true,
		SkipMessage:  "Free resource.",
	}
}
