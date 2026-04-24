package azure

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// NetworkDdosProtectionPlan struct represents Azure DDoS Protection Plan.
// DDoS Protection Plan is a resource that provides DDoS protection for virtual networks and IPs.
//
// Resource information: https://azure.microsoft.com/en-us/products/ddos-protection/
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/ddos-protection/#pricing
type NetworkDdosProtectionPlan struct {
	Address       string
	Region        string
	OverageAmount *int64 `c3x_usage:"overage_amount"`
}

// CoreType returns the name of this resource type.
func (r *NetworkDdosProtectionPlan) CoreType() string {
	return "NetworkDdosProtectionPlan"
}

// UsageSchema defines a list which represents the usage schema of
// NetworkDdosProtectionPlan. There is only one usage item, `overage_amount`,
// which represents the number of resources that fall outside the base ddos
// coverage.
func (r *NetworkDdosProtectionPlan) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "overage_amount", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the
// NetworkDdosProtectionPlan. It uses the `c3x_usage` struct tags to
// populate data into the NetworkDdosProtectionPlan.
func (r *NetworkDdosProtectionPlan) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid NetworkDdosProtectionPlan
// struct. This method is called after the resource is initialised by an IaC
// provider.
//
// BuildResource returns two cost components:
//
//  1. DDoS Protection Plan: The cost of the DDoS Protection Plan.
//  2. Overage charges: The cost of the overage charges for the DDoS Protection Plan.
//     This is the number of resources that fall outside the base coverage offered by
//     the protection plan (100). This amount is defined in the usage file as it is
//     difficult to infer the number of resources that fall outside the base coverage
//     from the IaC.
func (r *NetworkDdosProtectionPlan) BuildResource() *engine.Estimate {
	var overageAmount *decimal.Decimal
	overageUnit := "resource"
	if r.OverageAmount != nil {
		overageAmount = decimalPtr(decimal.NewFromInt(*r.OverageAmount))
		if overageAmount.GreaterThan(decimal.NewFromInt(1)) {
			overageUnit = "resources"
		}
	}

	return &engine.Estimate{
		Name:        r.Address,
		UsageSchema: r.UsageSchema(),
		CostComponents: []*engine.LineItem{
			{
				Name:           "DDoS Protection Plan",
				Unit:           "hours",
				UnitMultiplier: decimal.NewFromInt(1),
				HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr(vendorName),
					Region:        strPtr(r.Region),
					Service:       strPtr("Azure DDOS Protection"),
					ProductFamily: strPtr("Networking"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "skuName", Value: strPtr("Network Protection")},
						{Key: "meterName", Value: strPtr("Network Protection Plan")},
					},
				},
			},
			{
				Name:           "Overage charges",
				Unit:           overageUnit,
				UnitMultiplier: engine.HourToMonthUnitMultiplier,
				HourlyQuantity: overageAmount,
				ProductFilter: &engine.ProductSelector{
					VendorName:    strPtr(vendorName),
					Region:        strPtr(r.Region),
					Service:       strPtr("Azure DDOS Protection"),
					ProductFamily: strPtr("Networking"),
					AttributeFilters: []*engine.AttributeMatch{
						{Key: "skuName", Value: strPtr("Network Protection")},
						{Key: "meterName", Value: strPtr("Network Protection Resource")},
					},
				},
				UsageBased: true,
			},
		},
	}
}
