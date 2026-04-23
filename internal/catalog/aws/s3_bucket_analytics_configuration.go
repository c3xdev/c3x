package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type S3BucketAnalyticsConfiguration struct {
	Address                 string
	Region                  string
	MonthlyMonitoredObjects *int64 `c3x_usage:"monthly_monitored_objects"`
}

func (r *S3BucketAnalyticsConfiguration) CoreType() string {
	return "S3BucketAnalyticsConfiguration"
}

func (r *S3BucketAnalyticsConfiguration) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_monitored_objects", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *S3BucketAnalyticsConfiguration) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *S3BucketAnalyticsConfiguration) BuildResource() *engine.Estimate {
	var monitObj *decimal.Decimal
	if r.MonthlyMonitoredObjects != nil {
		monitObj = decimalPtr(decimal.NewFromInt(*r.MonthlyMonitoredObjects))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Objects monitored",
				Unit:            "1M objects",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: monitObj,
				ProductFilter: &engine.ProductSelector{
					VendorName: strPtr("aws"),
					Region:     strPtr(r.Region),
					Service:    strPtr("AmazonS3"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", ValueRegex: strPtr("/StorageAnalytics-ObjCount/")},
					},
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
