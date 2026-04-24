package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"fmt"

	"github.com/shopspring/decimal"
)

type NeptuneCluster struct {
	Address               string
	Region                string
	BackupRetentionPeriod int64
	StorageGB             *float64 `c3x_usage:"storage_gb"`
	MonthlyIORequests     *int64   `c3x_usage:"monthly_io_requests"`
	BackupStorageGB       *float64 `c3x_usage:"backup_storage_gb"`
}

func (r *NeptuneCluster) CoreType() string {
	return "NeptuneCluster"
}

func (r *NeptuneCluster) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
		{Key: "monthly_io_requests", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "backup_storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	}
}

func (r *NeptuneCluster) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *NeptuneCluster) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.storageCostComponent(),
		r.ioRequestsCostComponent(),
	}

	if r.BackupRetentionPeriod > 1 {
		costComponents = append(costComponents, r.backupStorageCostComponent())
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *NeptuneCluster) storageCostComponent() *engine.LineItem {
	var storageGB *decimal.Decimal
	if r.StorageGB != nil {
		storageGB = decimalPtr(decimal.NewFromFloat(*r.StorageGB))
	}

	return &engine.LineItem{
		Name:            "Storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: storageGB,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AmazonNeptune"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("^([A-Z]{3}\\d-|Global-|EU-)?StorageUsage$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *NeptuneCluster) ioRequestsCostComponent() *engine.LineItem {
	var monthlyIORequests *decimal.Decimal
	if r.MonthlyIORequests != nil {
		monthlyIORequests = decimalPtr(decimal.NewFromInt(*r.MonthlyIORequests))
	}

	return &engine.LineItem{
		Name:            "I/O requests",
		Unit:            "1M request",
		UnitMultiplier:  decimal.NewFromInt(int64(1000000)),
		MonthlyQuantity: monthlyIORequests,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AmazonNeptune"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr(fmt.Sprintf("/%s$/i", "StorageIOUsage"))},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}

func (r *NeptuneCluster) backupStorageCostComponent() *engine.LineItem {
	var backupStorageGB *decimal.Decimal
	if r.BackupStorageGB != nil {
		backupStorageGB = decimalPtr(decimal.NewFromFloat(*r.BackupStorageGB))
	}

	return &engine.LineItem{
		Name:            "Backup storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: backupStorageGB,
		ProductFilter: &engine.ProductSelector{
			VendorName: strPtr("aws"),
			Region:     strPtr(r.Region),
			Service:    strPtr("AmazonNeptune"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/BackupUsage$/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}
