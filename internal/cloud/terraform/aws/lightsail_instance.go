package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getLightsailInstanceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_lightsail_instance",
		CoreRFunc: NewLightsailInstance,
	}
}

func NewLightsailInstance(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.LightsailInstance{
		Address:  d.Address,
		BundleID: d.Get("bundle_id").String(),
		Region:   d.Get("region").String(),
	}
	return r
}
