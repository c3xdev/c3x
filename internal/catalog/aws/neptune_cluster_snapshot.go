package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

type NeptuneClusterSnapshot struct {
	Address               string
	Region                string
	BackupRetentionPeriod *int64   // This can be unknown since it's retrieved from the Neptune cluster
	BackupStorageGB       *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *NeptuneClusterSnapshot) CoreType() string {
	return "NeptuneClusterSnapshot"
}

func (r *NeptuneClusterSnapshot) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *NeptuneClusterSnapshot) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *NeptuneClusterSnapshot) BuildResource() *engine.Estimate {
	if r.BackupRetentionPeriod != nil && *r.BackupRetentionPeriod < 2 {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	cluster := &NeptuneCluster{
		Region:          r.Region,
		BackupStorageGB: r.BackupStorageGB,
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{cluster.backupStorageCostComponent()},
		UsageSchema:    r.UsageSchema(),
	}
}
