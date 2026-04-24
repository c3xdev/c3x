package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCodeBuildProjectRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_codebuild_project",
		CoreRFunc: NewCodeBuildProject,
	}
}
func NewCodeBuildProject(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.CodeBuildProject{
		Address:         d.Address,
		Region:          d.Get("region").String(),
		ComputeType:     d.Get("environment.0.compute_type").String(),
		EnvironmentType: d.Get("environment.0.type").String(),
	}
	return r
}
