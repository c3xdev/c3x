package azure

import (
	"fmt"
	"strings"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/c3xdev/c3x/internal/usage"
)

// TrafficManagerProfile struct represents an Azure Traffic Manager profile.
//
// Resource information: https://learn.microsoft.com/en-us/azure/traffic-manager/traffic-manager-overview
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/traffic-manager/#pricing
type TrafficManagerProfile struct {
	Address string
	Region  string

	Enabled            bool
	TrafficViewEnabled bool

	MonthlyDNSQueries            *int64 `c3x_usage:"monthly_dns_queries"`
	MonthlyTrafficViewDataPoints *int64 `c3x_usage:"monthly_traffic_view_data_points"`
}

// CoreType returns the name of this resource type
func (r *TrafficManagerProfile) CoreType() string {
	return "TrafficManagerProfile"
}

// UsageSchema defines a list which represents the usage schema of TrafficManagerProfile.
func (r *TrafficManagerProfile) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_dns_queries", DefaultValue: 0, ValueType: engine.Int64},
		{Key: "monthly_traffic_view_data_points", DefaultValue: 0, ValueType: engine.Int64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the TrafficManagerProfile.
// It uses the `c3x_usage` struct tags to populate data into the TrafficManagerProfile.
func (r *TrafficManagerProfile) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid TrafficManagerProfile struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *TrafficManagerProfile) BuildResource() *engine.Estimate {
	if !r.Enabled {
		return &engine.Estimate{
			Name: r.Address,
		}
	}

	costComponents := []*engine.LineItem{}

	if r.MonthlyDNSQueries != nil {
		dnsQuantities := usage.CalculateTierBuckets(
			*decimalPtr(decimal.NewFromInt(*r.MonthlyDNSQueries)),
			[]int{1000000000},
		)
		costComponents = append(costComponents, r.dnsQueriesCostComponent(&dnsQuantities[0], "first 1B", "0"))

		if dnsQuantities[1].GreaterThan(decimal.NewFromInt(0)) {
			costComponents = append(costComponents, r.dnsQueriesCostComponent(&dnsQuantities[1], "over 1B", "1000"))
		}
	} else {
		costComponents = append(costComponents, r.dnsQueriesCostComponent(nil, "first 1B", "0"))
	}

	if r.TrafficViewEnabled {
		costComponents = append(costComponents, r.trafficViewCostComponent())
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *TrafficManagerProfile) dnsQueriesCostComponent(q *decimal.Decimal, tierName, startUsage string) *engine.LineItem {
	var millions *decimal.Decimal
	if q != nil {
		millions = decimalPtr(q.Div(decimal.NewFromInt(1000000)))
	}

	return &engine.LineItem{
		Name:            fmt.Sprintf("DNS queries (%s)", tierName),
		Unit:            "1M queries",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: millions,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(trafficManagerBillingRegion(r.Region)),
			Service:       strPtr("Traffic Manager"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr("Azure Endpoint")},
				{Key: "meterName", Value: strPtr("DNS Queries")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(startUsage),
		},
		UsageBased: true,
	}
}

func (r *TrafficManagerProfile) trafficViewCostComponent() *engine.LineItem {
	var millions *decimal.Decimal
	if r.MonthlyTrafficViewDataPoints != nil {
		millions = decimalPtr(decimal.NewFromInt(*r.MonthlyTrafficViewDataPoints).Div(decimal.NewFromInt(1000000)))
	}

	return &engine.LineItem{
		Name:            "Traffic view",
		Unit:            "1M data points",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: millions,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(trafficManagerBillingRegion(r.Region)),
			Service:       strPtr("Traffic Manager"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr("Traffic View")},
				{Key: "meterName", Value: strPtr("Traffic View Data Points Processed")},
			},
		},
		UsageBased: true,
	}
}

func trafficManagerBillingRegion(region string) string {
	switch {
	case strings.Contains(strings.ToLower(region), "usgov"):
		return "US Gov"
	case strings.Contains(strings.ToLower(region), "china"):
		return "China"
	case strings.Contains(strings.ToLower(region), "germany"):
		return "Germany"
	default:
		return "Global"
	}
}
