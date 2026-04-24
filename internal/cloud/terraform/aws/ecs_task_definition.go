package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

// This is a free resource but needs it's own custom registry item to specify the custom ID lookup function.
func getECSTaskDefinitionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:    "aws_ecs_task_definition",
		NoPrice: true,
		Notes:   []string{"Free resource."},
		CustomRefIDFunc: func(d *engine.ResourceSpec) []string {
			refs := []string{d.Get("arn").String()}

			family := d.Get("family").String()
			if family != "" {
				refs = append(refs, family)
			}

			return refs
		},
	}
}
