package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNewEKSClusterItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_eks_cluster",
		CoreRFunc: NewEKSCluster,
	}
}
func NewEKSCluster(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.EKSCluster{
		Address: d.Address,
		Region:  d.Get("region").String(),
		Version: d.Get("version").String(),
	}
	return r
}
