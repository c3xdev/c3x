package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

var (
	eipReferences = []string{
		"aws_nat_gateway.allocation_id",
		"aws_eip_association.allocation_id",
		"aws_lb.subnet_mapping.#.allocation_id",
	}
)

func getEIPRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:                "aws_eip",
		ReferenceAttributes: eipReferences,
		CoreRFunc:           NewEIP,
	}
}
func NewEIP(d *engine.ResourceSpec) engine.CatalogItem {
	var allocated bool
	if len(d.References(eipReferences...)) > 0 {
		allocated = true
	}

	if !d.IsEmpty("customer_owned_ipv4_pool") || !d.IsEmpty("instance") || !d.IsEmpty("network_interface") {
		allocated = true
	}

	r := &aws.EIP{
		Address:   d.Address,
		Region:    d.Get("region").String(),
		Allocated: allocated,
	}
	return r
}
