package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDocDBClusterSnapshotRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_docdb_cluster_snapshot",
		CoreRFunc: NewDocDBClusterSnapshot,
	}

}
func NewDocDBClusterSnapshot(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.DocDBClusterSnapshot{
		Address: d.Address,
		Region:  d.Get("region").String(),
	}
	return r
}
