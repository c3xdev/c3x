package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getECSClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:  "aws_ecs_cluster",
		RFunc: NewECSCluster,
		// this is a reverse reference, it depends on the aws_ecs_cluster_capacity_provider RegistryItem
		// defining "cluster_name" as a ReferenceAttribute
		ReferenceAttributes: []string{"aws_ecs_cluster_capacity_providers.cluster_name"},
		CustomRefIDFunc: func(d *engine.ResourceSpec) []string {
			name := d.Get("name").String()
			if name != "" {
				return []string{name}
			}

			return nil
		},
	}
}

func NewECSCluster(d *engine.ResourceSpec, u *engine.ConsumptionProfile) *engine.Estimate {
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
