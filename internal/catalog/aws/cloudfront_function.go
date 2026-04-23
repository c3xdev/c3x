package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// CloudfrontFunction struct represents an AWS CloudFront Function. With
// CloudFront Functions, you can write lightweight functions in JavaScript
// for high-scale, latency-sensitive CDN customizations.
//
// Resource information: https://docs.aws.amazon.com/AmazonCloudFront/latest/DeveloperGuide/cloudfront-functions.html
// Pricing information: https://aws.amazon.com/cloudfront/pricing/
type CloudfrontFunction struct {
	Address string
	Region  string

	MonthlyRequests *int64 `c3x_usage:"monthly_requests"`
}

// CoreType returns the name of this resource type
func (r *CloudfrontFunction) CoreType() string {
	return "CloudfrontFunction"
}

// UsageSchema defines a list which represents the usage schema of CloudfrontFunction.
func (r *CloudfrontFunction) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "MonthlyRequests", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the CloudfrontFunction.
// It uses the `c3x_usage` struct tags to populate data into the CloudfrontFunction.
func (r *CloudfrontFunction) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid CloudfrontFunction struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *CloudfrontFunction) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	costComponents = append(costComponents, r.monthlyRequestsCostComponent())

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *CloudfrontFunction) monthlyRequestsCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Total number of invocations",
		Unit:            "1M invocations",
		UnitMultiplier:  decimal.NewFromInt(1000000),
		MonthlyQuantity: intPtrToDecimalPtr(r.MonthlyRequests),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Service:       strPtr("AmazonCloudFront"),
			ProductFamily: strPtr("Request"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", Value: strPtr("Executions-CloudFrontFunctions")},
				{Key: "groupDescription", ValueRegex: regexPtr("CloudFront Function")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption:   strPtr("on_demand"),
			StartUsageAmount: strPtr("2000000"),
		},
		UsageBased: true,
	}
}
