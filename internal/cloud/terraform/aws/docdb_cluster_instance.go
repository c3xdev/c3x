package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDocDBClusterInstanceRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_docdb_cluster_instance",
		CoreRFunc: NewDocDBClusterInstance,
	}
}
func NewDocDBClusterInstance(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.DocDBClusterInstance{
		Address:       d.Address,
		Region:        d.Get("region").String(),
		InstanceClass: d.Get("instance_class").String(),
	}
	return r
}
