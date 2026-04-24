package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRedshiftClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_redshift_cluster",
		CoreRFunc: NewRedshiftCluster,
	}
}

func NewRedshiftCluster(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.RedshiftCluster{
		Address:  d.Address,
		Region:   d.Get("region").String(),
		NodeType: d.Get("node_type").String(),
	}

	if !d.IsEmpty("number_of_nodes") {
		r.Nodes = intPtr(d.Get("number_of_nodes").Int())
	}
	return r
}
