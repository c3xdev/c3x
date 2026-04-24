package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getDocDBClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_docdb_cluster",
		CoreRFunc: NewDocDBCluster,
	}

}
func NewDocDBCluster(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.DocDBCluster{
		Address:               d.Address,
		Region:                d.Get("region").String(),
		BackupRetentionPeriod: d.Get("backup_retention_period").Int(),
	}
	return r
}
