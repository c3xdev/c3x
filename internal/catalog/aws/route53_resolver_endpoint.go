package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/c3xdev/c3x/internal/usage"

	"github.com/shopspring/decimal"
)

type Route53ResolverEndpoint struct {
	Address           string
	Region            string
	ResolverEndpoints int64
	MonthlyQueries    *int64 `c3x_usage:"monthly_queries"`
}

func (r *Route53ResolverEndpoint) CoreType() string {
	return "Route53ResolverEndpoint"
}

func (r *Route53ResolverEndpoint) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_queries", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *Route53ResolverEndpoint) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *Route53ResolverEndpoint) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		{
			Name:           "Resolver endpoints",
			Unit:           "hours",
			UnitMultiplier: decimal.NewFromInt(1),
			HourlyQuantity: decimalPtr(decimal.NewFromInt(r.ResolverEndpoints)),
			ProductFilter: &engine.ProductSelector{
				VendorName:    strPtr("aws"),
				Region:        strPtr(r.Region),
				Service:       strPtr("AmazonRoute53"),
				ProductFamily: strPtr("DNS Query"),
				AttributeFilters: []*engine.AttributeMatch{
					{Key: "usagetype", ValueRegex: strPtr("/ResolverNetworkInterface$/")},
				},
			},
		},
	}

	queryTierLimits := []int{1000000000}

	if r.MonthlyQueries != nil {
		monthlyQueries := decimal.NewFromInt(*r.MonthlyQueries)
		dnsQueriesTier := usage.CalculateTierBuckets(monthlyQueries, queryTierLimits)
		tierOne := dnsQueriesTier[0]
		tierTwo := dnsQueriesTier[1]

		if tierOne.GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.queriesCostComponent("DNS queries (first 1B)", "0", &tierOne))
		}

		if tierTwo.GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.queriesCostComponent("DNS queries (over 1B)", "1000000000", &tierTwo))
		}

	} else {
		var unknown *decimal.Decimal
		costComponents = append(costComponents, r.queriesCostComponent("DNS queries (first 1B)", "0", unknown))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *Route53ResolverEndpoint) queriesCostComponent(displayName string, usageTier string, monthlyQueries *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            displayName,
		Unit:            "1M queries",
		UnitMultiplier:  decimal.NewFromInt(1000000),
		MonthlyQuantity: monthlyQueries,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AmazonRoute53"),
			ProductFamily: strPtr("DNS Query"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/DNS-Queries/")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(usageTier),
		},
		UsageBased: true,
	}
}
