package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getPipesPipeRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_pipes_pipe",
		ReferenceAttributes: []string{"target_parameters.0.ecs_task_parameters.0.task_definition_arn"},
		NoPrice:             true,
		Notes:               []string{"Free resource."},
	}
}
