package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type DNSRecordSet struct {
	Address        string
	MonthlyQueries *int64 `c3x_usage:"monthly_queries"`
}

func (r *DNSRecordSet) CoreType() string {
	return "DNSRecordSet"
}

func (r *DNSRecordSet) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_queries", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *DNSRecordSet) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DNSRecordSet) BuildResource() *engine.Estimate {
	var monthlyQueries *decimal.Decimal

	if r.MonthlyQueries != nil {
		monthlyQueries = decimalPtr(decimal.NewFromInt(*r.MonthlyQueries))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Queries",
				Unit:            "1M queries",
				UnitMultiplier:  decimal.NewFromInt(1000000),
				MonthlyQuantity: monthlyQueries,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr("global"),
					Service:       strPtr("Cloud DNS"),
					ProductFamily: strPtr("Network"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", Value: strPtr("DNS Query (port 53)")},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr("0"),
				},
				UsageBased: true,
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
