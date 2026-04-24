package aws

import (
	"github.com/c3xdev/c3x/internal/engine"
)

func getEIPAssociationRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:    "aws_eip_association",
		NoPrice: true,
		ReferenceAttributes: []string{
			"allocation_id",
		},
		Notes: []string{"Free resource."},
	}
}
