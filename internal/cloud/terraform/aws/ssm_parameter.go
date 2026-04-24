package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSSMParameterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ssm_parameter",
		CoreRFunc: NewSSMParameter,
	}
}

func NewSSMParameter(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.SSMParameter{
		Address: d.Address,
		Region:  d.Get("region").String(),
		Tier:    d.Get("tier").String(),
	}
	return r
}
