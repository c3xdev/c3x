package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getELBRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_elb",
		CoreRFunc: NewELB,
	}
}
func NewELB(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.ELB{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
