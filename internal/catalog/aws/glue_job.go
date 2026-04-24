package aws

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// GlueJob struct represents a serverless AWS Glue job. A Glue job is designed to clean, transform and merge data
// into a data lake so that it is easy to analyze.
//
// GlueJob is just one resource of the wider AWS Glue service, which provides a number of different serverless services
// to build a robust data analytics pipeline.
//
// Resource information: https://aws.amazon.com/glue/
// Pricing information: https://aws.amazon.com/glue/pricing/
type GlueJob struct {
	Address string
	Region  string
	DPUs    float64

	MonthlyHours *float64 `c3x_usage:"monthly_hours"`
}

func (r *GlueJob) CoreType() string {
	return "GlueJob"
}

func (r *GlueJob) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_hours", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the GlueJob.
// It uses the `c3x_usage` struct tags to populate data into the GlueJob.
func (r *GlueJob) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid GlueJob struct. GlueJob has just one engine.LineItem
// associated with it:
//
//  1. DPU hours - GlueJob is charged per hour that the job is run. Users are charged based on the number of DPU
//     units they use in that time.
//
// This method is called after the resource is initialised by an IaC provider. See providers folder for more information.
func (r *GlueJob) BuildResource() *engine.Estimate {
	var quantity *decimal.Decimal
	if r.MonthlyHours != nil {
		quantity = decimalPtr(decimal.NewFromFloat(*r.MonthlyHours * r.DPUs))
	}

	suffix := "DPUs"
	if r.DPUs == 1 {
		suffix = "DPU"
	}

	unit := fmt.Sprintf("hours (%d %s)", int(r.DPUs), suffix)

	if r.DPUs > 0 && r.DPUs < 1 {
		unit = fmt.Sprintf("hours (%.4f %s)", r.DPUs, suffix)
	}

	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:            "Duration",
				Unit:            unit,
				UnitMultiplier:  decimal.NewFromFloat(r.DPUs),
				MonthlyQuantity: quantity,
				ProductFilter: &engine.ProductSelector{
					VendorName:    vendorName,
					Region:        strPtr(r.Region),
					Service:       strPtr("AWSGlue"),
					ProductFamily: strPtr("AWS Glue"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "group", Value: strPtr("ETL Job run")},
						{Key: "operation", ValueRegex: strPtr("/^jobrun$/i")},
					},
				},
				UsageBased: true,
			},
		},
	}
}
