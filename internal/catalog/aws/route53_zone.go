package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type Route53Zone struct {
	Address string
}

func (r *Route53Zone) CoreType() string {
	return "Route53Zone"
}

func (r *Route53Zone) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *Route53Zone) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *Route53Zone) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Hosted zone",
				Unit:            "months",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("aws"),
					Service:       strPtr("AmazonRoute53"),
					ProductFamily: strPtr("DNS Zone"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "usagetype", Value: strPtr("HostedZone")},
					},
				},
				PriceFilter: &engine.RateSelector{
					StartUsageAmount: strPtr("0"),
				},
			},
		},
		UsageSchema: r.UsageSchema(),
	}
}
