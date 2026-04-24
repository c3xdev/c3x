package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getSchedulerScheduleRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_scheduler_schedule",
		ReferenceAttributes: []string{"ecs_parameters.0.task_definition_arn"},
		NoPrice:             true,
		Notes:               []string{"Free resource."},
	}
}
