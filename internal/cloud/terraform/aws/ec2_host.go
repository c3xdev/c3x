package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getEC2HostRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_ec2_host",
		CoreRFunc: newEC2Host,
	}
}

func newEC2Host(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("region").String()
	r := &aws.EC2Host{
		Address:        d.Address,
		Region:         region,
		InstanceType:   d.Get("instance_type").String(),
		InstanceFamily: d.Get("instance_family").String(),
	}
	return r
}
