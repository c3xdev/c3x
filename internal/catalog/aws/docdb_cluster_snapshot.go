package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type DocDBClusterSnapshot struct {
	Address         string
	Region          string
	BackupStorageGB *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *DocDBClusterSnapshot) CoreType() string {
	return "DocDBClusterSnapshot"
}

func (r *DocDBClusterSnapshot) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *DocDBClusterSnapshot) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DocDBClusterSnapshot) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	var backupStorage *decimal.Decimal
	if r.BackupStorageGB != nil {
		backupStorage = decimalPtr(decimal.NewFromFloat(*r.BackupStorageGB))
	}

	cluster := &DocDBCluster{
		Region: r.Region,
	}

	costComponents = append(costComponents, cluster.backupStorageCostComponent(backupStorage))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}
