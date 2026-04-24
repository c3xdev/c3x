package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRoute53ZoneRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_route53_zone",
		CoreRFunc: NewRoute53Zone,
	}
}

func NewRoute53Zone(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.Route53Zone{
		Address: d.Address,
	}
	return r
}
