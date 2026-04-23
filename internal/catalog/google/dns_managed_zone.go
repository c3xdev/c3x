package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type DNSManagedZone struct {
	Address string
}

func (r *DNSManagedZone) CoreType() string {
	return "DNSManagedZone"
}

func (r *DNSManagedZone) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

func (r *DNSManagedZone) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *DNSManagedZone) BuildResource() *engine.Estimate {
	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			{
				Name:            "Managed zone",
				Unit:            "months",
				UnitMultiplier:  decimal.NewFromInt(1),
				MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr("gcp"),
					Region:        strPtr("global"),
					Service:       strPtr("Cloud DNS"),
					ProductFamily: strPtr("Network"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "description", Value: strPtr("ManagedZone")},
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
