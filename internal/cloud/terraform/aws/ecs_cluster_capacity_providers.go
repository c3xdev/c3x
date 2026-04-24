package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getECSClusterCapacityProvidersRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_ecs_cluster_capacity_providers",
		RFunc:               NewECSClusterCapacityProviders,
		ReferenceAttributes: []string{"cluster_name"},
	}
}

func NewECSClusterCapacityProviders(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
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
