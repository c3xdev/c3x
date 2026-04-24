package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getCloudFormationStackSetRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_cloudformation_stack_set",
		CoreRFunc: NewCloudFormationStackSet,
	}
}
func NewCloudFormationStackSet(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.CloudFormationStackSet{
		Address:      d.Address,
		Region:       d.Get("region").String(),
		TemplateBody: d.Get("template_body").String(),
	}
	return r
}
