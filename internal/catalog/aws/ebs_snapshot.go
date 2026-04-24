package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type EBSSnapshot struct {
	Address                  string
	Region                   string
	SizeGB                   *float64
	MonthlyListBlockRequests *int64 `c3x_usage:"monthly_list_block_requests"`
	MonthlyGetBlockRequests  *int64 `c3x_usage:"monthly_get_block_requests"`
	MonthlyPutBlockRequests  *int64 `c3x_usage:"monthly_put_block_requests"`
	FastSnapshotRestoreHours *int64 `c3x_usage:"fast_snapshot_restore_hours"`
}

func (r *EBSSnapshot) CoreType() string {
	return "EBSSnapshot"
}

func (r *EBSSnapshot) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_list_block_requests", ValueType: engine.Int64, DefaultValue: 0}, {Key: "monthly_get_block_requests", ValueType: engine.Int64, DefaultValue: 0}, {Key: "monthly_put_block_requests", ValueType: engine.Int64, DefaultValue: 0}, {Key: "fast_snapshot_restore_hours", ValueType: engine.Int64, DefaultValue: 0}}
}

func (r *EBSSnapshot) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *EBSSnapshot) BuildResource() *engine.Estimate {
	region := r.Region

	gbVal := decimal.NewFromFloat(float64(defaultVolumeSize))

	if r.SizeGB != nil {
		gbVal = decimal.NewFromFloat(*r.SizeGB)
	}

	var listBlockRequests *decimal.Decimal
	if r.MonthlyListBlockRequests != nil {
		listBlockRequests = decimalPtr(decimal.NewFromInt(*r.MonthlyListBlockRequests))
	}

	var getSnapshotBlockRequests *decimal.Decimal
	if r.MonthlyGetBlockRequests != nil {
		getSnapshotBlockRequests = decimalPtr(decimal.NewFromInt(*r.MonthlyGetBlockRequests))
	}

	var putSnapshotBlockRequests *decimal.Decimal
	if r.MonthlyPutBlockRequests != nil {
		putSnapshotBlockRequests = decimalPtr(decimal.NewFromInt(*r.MonthlyPutBlockRequests))
	}

	var fastSnapshotRestoreHours *decimal.Decimal
	if r.MonthlyPutBlockRequests != nil {
		fastSnapshotRestoreHours = decimalPtr(decimal.NewFromInt(*r.FastSnapshotRestoreHours))
	}

	costComponents := []*engine.LineItem{
		ebsSnapshotCostComponent(region, gbVal),
		{
			Name:            "Fast snapshot restore",
			Unit:            "DSU-hours",
			UnitMultiplier:  decimal.NewFromInt(1),
			MonthlyQuantity: fastSnapshotRestoreHours,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonEC2"),
				ProductFamily: strPtr("Fast Snapshot Restore"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/EBS:FastSnapshotRestore$/")},
				},
			},
			UsageBased: true,
		},
		{
			Name:            "ListChangedBlocks & ListSnapshotBlocks API requests",
			Unit:            "1k requests",
			UnitMultiplier:  decimal.NewFromInt(1000),
			MonthlyQuantity: listBlockRequests,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonEC2"),
				ProductFamily: strPtr("EBS direct API Requests"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/EBS:directAPI.snapshot.List$/")},
				},
			},
			UsageBased: true,
		},
		{
			Name:            "GetSnapshotBlock API requests",
			Unit:            "1k SnapshotAPIUnits",
			UnitMultiplier:  decimal.NewFromInt(1000),
			MonthlyQuantity: getSnapshotBlockRequests,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonEC2"),
				ProductFamily: strPtr("EBS direct API Requests"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/EBS:directAPI.snapshot.Get$/")},
				},
			},
			UsageBased: true,
		},
		{
			Name:            "PutSnapshotBlock API requests",
			Unit:            "1k SnapshotAPIUnits",
			UnitMultiplier:  decimal.NewFromInt(1000),
			MonthlyQuantity: putSnapshotBlockRequests,
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(region),
				Service:       strPtr("AmazonEC2"),
				ProductFamily: strPtr("EBS direct API Requests"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/EBS:directAPI.snapshot.Put$/")},
				},
			},
			UsageBased: true,
		}}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents, UsageSchema: r.UsageSchema(),
	}
}

func ebsSnapshotCostComponent(region string, gbVal decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "EBS snapshot storage",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: &gbVal,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(region),
			Service:       strPtr("AmazonEC2"),
			ProductFamily: strPtr("Storage Snapshot"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/EBS:SnapshotUsage$/")},
			},
		},
		UsageBased: true,
	}
}
