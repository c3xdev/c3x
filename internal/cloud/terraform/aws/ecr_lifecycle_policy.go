package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getECRLifecyclePolicy() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_ecr_lifecycle_policy",
		ReferenceAttributes: []string{"repository"},
		NoPrice:             true,
		Notes:               []string{"Free resource."},
	}
}
