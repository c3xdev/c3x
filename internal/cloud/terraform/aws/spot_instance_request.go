package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getSpotInstanceRequestRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name: "aws_spot_instance_request",
		Notes: []string{
			"Notes",
		},
		CoreRFunc: newSpotInstanceRequest,
	}
}

func newSpotInstanceRequest(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()

	var instanceType, ami string

	ami = d.GetStringOrDefault("ami", ami)
	instanceType = d.GetStringOrDefault("instance_type", instanceType)

	r := &aws.Instance{
		Address:        d.Address,
		Region:         region,
		PurchaseOption: "spot",
		InstanceType:   instanceType,
		AMI:            ami,
	}

	return r
}
