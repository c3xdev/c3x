package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDXConnectionRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_dx_connection",
		CoreRFunc: NewDXConnection,
	}
}

func NewDXConnection(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.DXConnection{
		Address:   d.Address,
		Region:    d.Get("region").String(),
		Bandwidth: d.Get("bandwidth").String(),
		Location:  d.Get("location").String(),
	}
	return r
}
