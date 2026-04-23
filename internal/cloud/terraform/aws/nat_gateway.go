package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNATGatewayRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "aws_nat_gateway",
		ReferenceAttributes: []string{
			"allocation_id",
			"subnet_id",
		},
		CoreRFunc: NewNATGateway,
	}
}

func NewNATGateway(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()

	a := &aws.NATGateway{
		Address: d.Address,
		Region:  region,
	}

	return a
}
