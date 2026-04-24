package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getGlobalAcceleratorRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_globalaccelerator_accelerator",
		CoreRFunc: newGlobalAccelerator,
	}
}

func newGlobalAccelerator(d *engine.ResourceSpec) engine.CatalogItem {

	r := &aws.GlobalAccelerator{
		Address: d.Address,
	}

	return r
}
