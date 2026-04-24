package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getGlobalacceleratorEndpointGroupRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_globalaccelerator_endpoint_group",
		CoreRFunc: newGlobalacceleratorEndpointGroup,
	}
}

func newGlobalacceleratorEndpointGroup(d *engine.ResourceSpec) engine.CatalogItem {
	region := d.Get("endpoint_group_region").String()
	r := &aws.GlobalacceleratorEndpointGroup{
		Address: d.Address,
		Region:  region,
	}

	return r
}
