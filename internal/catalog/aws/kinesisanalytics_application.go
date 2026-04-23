package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type KinesisAnalyticsApplication struct {
	Address                string
	Region                 string
	KinesisProcessingUnits *int64 `c3x_usage:"kinesis_processing_units"`
}

func (r *KinesisAnalyticsApplication) CoreType() string {
	return "KinesisAnalyticsApplication"
}

func (r *KinesisAnalyticsApplication) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "kinesis_processing_units", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *KinesisAnalyticsApplication) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *KinesisAnalyticsApplication) BuildResource() *engine.Estimate {
	var kinesisProcessingUnits *decimal.Decimal
	if r.KinesisProcessingUnits != nil {
		kinesisProcessingUnits = decimalPtr(decimal.NewFromInt(*r.KinesisProcessingUnits))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: []*engine.LineItem{r.processingStreamCostComponent(kinesisProcessingUnits)},
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *KinesisAnalyticsApplication) processingStreamCostComponent(kinesisProcessingUnits *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:           "Processing (stream)",
		Unit:           "KPU",
		UnitMultiplier: engine.HourToMonthUnitMultiplier,
		HourlyQuantity: kinesisProcessingUnits,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonKinesisAnalytics"),
			ProductFamily: strPtr("Kinesis Analytics"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/KPU-Hour-Java/i")},
			},
		},
		UsageBased: true,
	}
}
