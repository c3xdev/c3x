package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRoute53RecordRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_route53_record",
		CoreRFunc:           NewRoute53Record,
		ReferenceAttributes: []string{"alias.0.name"},
	}
}
func NewRoute53Record(d *engine.ResourceSpec) engine.CatalogItem {
	isAlias := false

	aliasRefs := d.References("alias.0.name")
	if len(aliasRefs) > 0 && aliasRefs[0].Type != "aws_route53_record" {
		isAlias = true
	}

	r := &aws.Route53Record{
		Address: d.Address,
		IsAlias: isAlias,
	}
	return r
}
