package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getECSTaskSet() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "aws_ecs_task_set",
		RFunc: func(d *engine.ResourceSpec, _ *engine.ConsumptionProfile) *engine.Estimate {
			return &engine.Estimate{
				Name:         d.Address,
				ResourceType: d.Type,
				Tags:         d.Tags,
				DefaultTags:  d.DefaultTags,
				IsSkipped:    true,
				NoPrice:      true,
				SkipMessage:  "Free resource.",
			}
		},
		ReferenceAttributes: []string{"service", "cluster", "task_definition"},
	}
}
