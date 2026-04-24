package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// CloudHSMv2HSM struct represents an HSM module in a CloudHSM cluster.
//
// The HSM module is charged a hourly rate. Terraform allows you to specify the instance
// type of the HSM cluster, but at the moment AWS only supports one instance type, so
// each module has a set price depending on region.
//
// The cluster is counted as a free resource, but each cluster can have up to 32 HSM modules.
//
// Resource information: https://aws.amazon.com/cloudhsm/
// Pricing information: https://aws.amazon.com/cloudhsm/pricing/
type CloudHSMv2HSM struct {
	Address string
	Region  string

	MonthlyHours *float64 `c3x_usage:"monthly_hrs"`
}

// CoreType returns the name of this resource type
func (r *CloudHSMv2HSM) CoreType() string {
	return "CloudHSMv2HSM"
}

// UsageSchema defines a list which represents the usage schema of CloudHSMv2HSM.
func (r *CloudHSMv2HSM) UsageSchema() []*engine.ConsumptionField {
	hours, _ := engine.HourToMonthUnitMultiplier.Float64()

	return []*engine.ConsumptionField{
		{Key: "monthly_hrs", DefaultValue: hours, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the CloudHSMv2HSM.
// It uses the `c3x_usage` struct tags to populate data into the CloudHSMv2HSM.
func (r *CloudHSMv2HSM) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid CloudHSMv2HSM struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *CloudHSMv2HSM) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.hsmCostComponent(),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *CloudHSMv2HSM) hsmCostComponent() *engine.LineItem {
	quantity := engine.HourToMonthUnitMultiplier
	if r.MonthlyHours != nil {
		quantity = decimal.NewFromFloat(*r.MonthlyHours)
	}

	return &engine.LineItem{
		Name:            "HSM usage",
		Unit:            "hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(quantity),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("CloudHSM"),
			ProductFamily: strPtr("Dedicated-Host"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "instanceFamily", Value: strPtr("CloudHSM-v2")},
				{Key: "usagetype", ValueRegex: regexPtr("CloudHSMv2Usage$")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("on_demand"),
		},
		UsageBased: true,
	}
}
