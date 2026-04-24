package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type CloudwatchLogGroup struct {
	Address               string
	Region                string
	MonthlyDataIngestedGB *float64 `c3x_usage:"monthly_data_ingested_gb"`
	StorageGB             *float64 `c3x_usage:"storage_gb"`
	MonthlyDataScannedGB  *float64 `c3x_usage:"monthly_data_scanned_gb"`
}

var CloudwatchLogGroupUsageSchema = []*engine.ConsumptionField{
	{Key: "monthly_data_ingested_gb", ValueType: engine.Float64, DefaultValue: 0},
	{Key: "storage_gb", ValueType: engine.Float64, DefaultValue: 0},
	{Key: "monthly_data_scanned_gb", ValueType: engine.Float64, DefaultValue: 0},
}

func (r *CloudwatchLogGroup) CoreType() string {
	return "CloudwatchLogGroup"
}

func (r *CloudwatchLogGroup) UsageSchema() []*engine.ConsumptionField {
	return CloudwatchLogGroupUsageSchema
}

func (r *CloudwatchLogGroup) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *CloudwatchLogGroup) BuildResource() *engine.Estimate {
	var gbDataIngestion *decimal.Decimal
	var gbDataStorage *decimal.Decimal
	var gbDataScanned *decimal.Decimal

	if r.MonthlyDataIngestedGB != nil {
		gbDataIngestion = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataIngestedGB))
	}

	if r.StorageGB != nil {
		gbDataStorage = decimalPtr(decimal.NewFromFloat(*r.StorageGB))
	}

	if r.MonthlyDataScannedGB != nil {
		gbDataScanned = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataScannedGB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Data ingested",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: gbDataIngestion,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonCloudWatch"),
					ProductFamily: strPtr("Data Payload"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/-DataProcessing-Bytes/")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Archival Storage",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: gbDataStorage,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonCloudWatch"),
					ProductFamily: strPtr("Storage Snapshot"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/-TimedStorage-ByteHrs/")},
					},
				},
				UsageBased: true,
			},
			{
				Name:            "Insights queries data scanned",
				Unit:            "GB",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: gbDataScanned,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Region:        strPtr(r.Region),
					Service:       strPtr("AmazonCloudWatch"),
					ProductFamily: strPtr("Data Payload"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/-DataScanned-Bytes/")},
					},
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
