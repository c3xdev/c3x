package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNeptuneClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_neptune_cluster",
		CoreRFunc: NewNeptuneCluster,
	}
}

func NewNeptuneCluster(d *engine.ResourceSpec) engine.CatalogItem {
	r := &aws.NeptuneCluster{
		Address:               d.Address,
		Region:                d.Get("region").String(),
		BackupRetentionPeriod: d.Get("backup_retention_period").Int(),
	}
	return r
}
