package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getVPCEndpointRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_vpc_endpoint",
		CoreRFunc: NewVPCEndpoint,
		ReferenceAttributes: []string{
			"subnet_ids",
			"vpc_id",
		},
	}
}

func NewVPCEndpoint(d *engine.ResourceSpec) engine.CatalogItem {
	subnetIDs := len(d.Get("subnet_ids").Array())

	// if the length of the subnet_ids attribute is zero this means that the attribute
	// has been modified with a subnet id that is yet to exist. In this instance we'll
	// use the reference attribute instead. In most cases this should have the accurate
	// number of subnet_ids.
	if subnetIDs == 0 {
		subnetIDs = len(d.References("subnet_ids"))
	}

	var interfaces int64 = 1
	if subnetIDs > 0 {
		interfaces = int64(subnetIDs)
	}

	r := &aws.VPCEndpoint{
		Address:    d.Address,
		Region:     d.Get("region").String(),
		Interfaces: intPtr(interfaces),
		Type:       d.Get("vpc_endpoint_type").String(),
	}
	return r
}
