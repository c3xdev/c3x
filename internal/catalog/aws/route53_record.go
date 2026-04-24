package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"
)

type Route53Record struct {
	Address                    string
	IsAlias                    bool
	MonthlyLatencyBasedQueries *int64 `c3x_usage:"monthly_latency_based_queries"`
	MonthlyGeoQueries          *int64 `c3x_usage:"monthly_geo_queries"`
	MonthlyStandardQueries     *int64 `c3x_usage:"monthly_standard_queries"`
}

func (r *Route53Record) CoreType() string {
	return "Route53Record"
}

func (r *Route53Record) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_latency_based_queries", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_geo_queries", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_standard_queries", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *Route53Record) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *Route53Record) BuildResource() *engine.Estimate {
	if r.IsAlias {
		return &engine.Estimate{
			Name:        r.Address,
			NoPrice:     true,
			IsSkipped:   true,
			UsageSchema: r.UsageSchema(),
		}
	}

	costComponents := []*engine.LineItem{}
	limits := []int{1000000000}

	var numbOfStdQueries *decimal.Decimal
	if r.MonthlyStandardQueries != nil {
		numbOfStdQueries = decimalPtr(decimal.NewFromInt(*r.MonthlyStandardQueries))
		stdQueriesTiers := usage.CalculateTierBuckets(*numbOfStdQueries, limits)

		if stdQueriesTiers[0].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, queriesCostComponent("Standard queries (first 1B)", "DNS-Queries", "0", &stdQueriesTiers[0]))
		}

		if stdQueriesTiers[1].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, queriesCostComponent("Standard queries (over 1B)", "DNS-Queries", "1000000000", &stdQueriesTiers[1]))
		}
	} else {
		var unknown *decimal.Decimal

		costComponents = append(costComponents, queriesCostComponent("Standard queries (first 1B)", "DNS-Queries", "0", unknown))
	}

	var numbOfLBRQueries *decimal.Decimal
	if r.MonthlyLatencyBasedQueries != nil {
		numbOfLBRQueries = decimalPtr(decimal.NewFromInt(*r.MonthlyLatencyBasedQueries))
		lbrQueriesTiers := usage.CalculateTierBuckets(*numbOfLBRQueries, limits)

		if lbrQueriesTiers[0].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, queriesCostComponent("Latency based routing queries (first 1B)", "LBR-Queries", "0", &lbrQueriesTiers[0]))
		}

		if lbrQueriesTiers[1].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, queriesCostComponent("Latency based routing queries (over 1B)", "LBR-Queries", "1000000000", &lbrQueriesTiers[1]))
		}
	} else {
		var unknown *decimal.Decimal

		costComponents = append(costComponents, queriesCostComponent("Latency based routing queries (first 1B)", "LBR-Queries", "0", unknown))
	}

	var numbOfGeoQueries *decimal.Decimal
	if r.MonthlyGeoQueries != nil {
		numbOfGeoQueries = decimalPtr(decimal.NewFromInt(*r.MonthlyGeoQueries))
		geoQueriesTiers := usage.CalculateTierBuckets(*numbOfGeoQueries, limits)

		if geoQueriesTiers[0].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, queriesCostComponent("Geo DNS queries (first 1B)", "Geo-Queries", "0", &geoQueriesTiers[0]))
		}

		if geoQueriesTiers[1].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, queriesCostComponent("Geo DNS queries (over 1B)", "Geo-Queries", "1000000000", &geoQueriesTiers[1]))
		}
	} else {
		var unknown *decimal.Decimal

		costComponents = append(costComponents, queriesCostComponent("Geo DNS queries (first 1B)", "Geo-Queries", "0", unknown))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func queriesCostComponent(displayName string, usageType string, usageTier string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            displayName,
		Unit:            "1M queries",
		UnitMultiplier:  decimal.NewFromInt(1000000),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Service:       strPtr("AmazonRoute53"),
			ProductFamily: strPtr("DNS Query"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", Value: &usageType},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(usageTier),
		},
		UsageBased: true,
	}
}
