package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// GlueCrawler struct represents a serverless AWS Glue crawler. A Glue crawler crawls defined data sources and sends them
// into a Glue data catalog, ready for a Glue job to transform and merge into a main dataset/lake.
//
// GlueCrawler is just one resource of the wider AWS Glue service, which provides a number of different serverless services
// to build a robust data analytics pipeline.
//
// Resource information: https://aws.amazon.com/glue/
// Pricing information: https://aws.amazon.com/glue/pricing/
type GlueCrawler struct {
	Address string
	Region  string

	MonthlyHours *float64 `c3x_usage:"monthly_hours"`
}

func (r *GlueCrawler) CoreType() string {
	return "GlueCrawler"
}

func (r *GlueCrawler) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_hours", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the GlueCrawler.
// It uses the `c3x_usage` struct tags to populate data into the GlueCrawler.
func (r *GlueCrawler) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid GlueCrawler struct. GlueCrawler has just one engine.LineItem
// associated with it:
//
//  1. Hours - GlueCrawler is charged per hour that the crawler is run.
//
// This method is called after the resource is initialised by an IaC provider. See providers folder for more information.
func (r *GlueCrawler) BuildResource() *engine.Estimate {
	var quantity *decimal.Decimal
	if r.MonthlyHours != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyHours))
	}

	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:            "Duration",
				Unit:            "hours",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: quantity,
				ProductFilter: &engine.ProductSelector{
					VendorName:    vendorName,
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSGlue"),
					ProductFamily: strPtr("AWS Glue"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "operation", ValueRegex: strPtr("/^crawlerrun$/i")},
					},
				},
				UsageBased: true,
			},
		},
	}
}
