package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type DocDBCluster struct {
	Address               string
	Region                string
	BackupRetentionPeriod int64
	BackupStorageGB       *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *DocDBCluster) CoreType() string {
	return "DocDBCluster"
}

func (r *DocDBCluster) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *DocDBCluster) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DocDBCluster) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	if r.BackupRetentionPeriod > 1 {
		var backupStorage *decimal.Decimal
		if r.BackupStorageGB != nil {
			backupStorage = decimalPtr(decimal.NewFromFloat(*r.BackupStorageGB))
		}
		costComponents = append(costComponents, r.backupStorageCostComponent(backupStorage))
	} else {
		costComponents = append(costComponents, r.backupStorageCostComponent(nil))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *DocDBCluster) backupStorageCostComponent(backupStorage *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Backup storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: backupStorage,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonDocDB"),
			ProductFamily: strPtr("Storage Snapshot"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("(^|-)BackupUsage$")},
			},
		},
		UsageBased: true,
	}
}
