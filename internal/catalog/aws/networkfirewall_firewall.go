package aws

import (
	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// NetworkfirewallFirewall struct represents an AWS Network Firewall Firewall resource.
//
// Resource information: https://aws.amazon.com/network-firewall/
// Pricing information: https://aws.amazon.com/network-firewall/pricing/
type NetworkfirewallFirewall struct {
	Address string
	Region  string

	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

// NetworkfirewallFirewallUsageSchema defines a list which represents the usage schema of NetworkfirewallFirewall.
func (r *NetworkfirewallFirewall) CoreType() string {
	return "NetworkfirewallFirewall"
}

func (r *NetworkfirewallFirewall) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_data_processed_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the NetworkfirewallFirewall.
// It uses the `c3x_usage` struct tags to populate data into the NetworkfirewallFirewall.
func (r *NetworkfirewallFirewall) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid NetworkfirewallFirewall struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *NetworkfirewallFirewall) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{
		r.endpointCostComponent(),
		r.dataProcessedCostComponent(),
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *NetworkfirewallFirewall) endpointCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "Network Firewall Endpoint",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSNetworkFirewall"),
			ProductFamily: strPtr("AWS Firewall"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("^[A-Z0-9]*-Endpoint-Hour$")},
			},
		},
	}
}

func (r *NetworkfirewallFirewall) dataProcessedCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data Processed",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: floatPtrToDecimalPtr(r.MonthlyDataProcessedGB),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSNetworkFirewall"),
			ProductFamily: strPtr("AWS Firewall"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: regexPtr("^[A-Z0-9]*-Traffic-GB-Processed$")},
			},
		},
		UsageBased: true,
	}
}
