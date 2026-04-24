package aws

import (
	"github.com/c3xdev/c3x/internal/catalog/aws"
	"github.com/c3xdev/c3x/internal/engine"
)

func getRDSClusterRegistryItem() *engine.CatalogEntry {
	return &engine.CatalogEntry{
		Name:      "aws_rds_cluster",
		CoreRFunc: NewRDSCluster,
	}
}

func NewRDSCluster(d *engine.ResourceSpec) engine.CatalogItem {
	engineMode := d.GetStringOrDefault("engine_mode", "provisioned")
	r := &aws.RDSCluster{
		Address:               d.Address,
		Region:                d.Get("region").String(),
		Engine:                d.GetStringOrDefault("engine", "aurora"),
		BackupRetentionPeriod: d.GetInt64OrDefault("backup_retention_period", 1),
		EngineMode:            engineMode,
		IOOptimized:           d.Get("storage_type").String() == "aurora-iopt1",
	}
	return r
}
