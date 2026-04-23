package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getVPCRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_vpc",
		RFunc:               NewVPC,
		ReferenceAttributes: []string{"aws_vpc_endpoint.vpc_id"},
	}
}

func NewVPC(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
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
