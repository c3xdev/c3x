package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getECRRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_ecr_repository",
		CoreRFunc:           NewECRRepository,
		ReferenceAttributes: []string{"aws_ecr_lifecycle_policy.repository"},
	}
}
func NewECRRepository(d *engine.ResourceSpec) engine.CatalogItem {
	return &aws.ECRRepository{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
}
