package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudwatchEventTargetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_cloudwatch_event_target",
		ReferenceAttributes: []string{"ecs_target.0.task_definition_arn"},
		NoPrice:             true,
		Notes:               []string{"Free resource."},
	}
}
