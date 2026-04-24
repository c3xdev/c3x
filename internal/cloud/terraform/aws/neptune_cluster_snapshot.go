package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getNeptuneClusterSnapshotRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_neptune_cluster_snapshot",
		CoreRFunc: NewNeptuneClusterSnapshot,
		ReferenceAttributes: []string{
			"db_cluster_identifier",
		},
	}
}

func NewNeptuneClusterSnapshot(d *engine.ResourceSpec) engine.CatalogItem {
	var backupRetentionPeriod *int64

	dbClusterIdentifiers := d.References("db_cluster_identifier")
	if len(dbClusterIdentifiers) > 0 {
		cluster := dbClusterIdentifiers[0]
		backupRetentionPeriod = intPtr(cluster.GetInt64OrDefault("backup_retention_period", 1))
	}

	r := &aws.NeptuneClusterSnapshot{
		Address:               d.Address,
		Region:                d.Get("region").String(),
		BackupRetentionPeriod: backupRetentionPeriod,
	}
	return r
}
