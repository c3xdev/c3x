package azure

import (
	"fmt"
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// TrafficManagerEndpoint struct represents Azure Traffic Manager Endpoints.
//
// Resource information: https://learn.microsoft.com/en-us/azure/traffic-manager/traffic-manager-endpoint-types
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/traffic-manager/#pricing
type TrafficManagerEndpoint struct {
	Address string
	Region  string

	ProfileEnabled      bool
	External            bool
	HealthCheckInterval int64
}

// CoreType returns the name of this resource type
func (r *TrafficManagerEndpoint) CoreType() string {
	return "TrafficManagerEndpoint"
}

// UsageSchema defines a list which represents the usage schema of TrafficManagerEndpoint.
func (r *TrafficManagerEndpoint) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the TrafficManagerEndpoint.
// It uses the `c3x_usage` struct tags to populate data into the TrafficManagerEndpoint.
func (r *TrafficManagerEndpoint) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from a valid TrafficManagerEndpoint struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *TrafficManagerEndpoint) BuildResource() *engine.Estimate {
	if !r.ProfileEnabled {
		return &engine.Estimate{
			Name: r.Address,
		}
	}

	costComponents := []*engine.LineItem{
		r.healthCheckCostComponent(),
	}

	if r.HealthCheckInterval < 30 {
		costComponents = append(costComponents, r.fastHealthCheckCostComponent())
	}

	return &engine.Estimate{
		Name:           r.Address,
		UsageSchema:    r.UsageSchema(),
		CostComponents: costComponents,
	}
}

func (r *TrafficManagerEndpoint) sku() string {
	if r.External {
		return "Non-Azure Endpoint"
	} else {
		return "Azure Endpoint"
	}
}

func (r *TrafficManagerEndpoint) healthCheckCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            fmt.Sprintf("Basic health check (%s)", trafficManagerBillingRegion(r.Region)),
		Unit:            "hours",
		UnitMultiplier:  engine.MonthToHourUnitMultiplier,
		UnitRounding:    int32Ptr(0),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),

		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(trafficManagerBillingRegion(r.Region)),
			Service:       strPtr("Traffic Manager"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(r.sku())},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Health Checks", r.sku()))},
			},
		},
	}
}

func (r *TrafficManagerEndpoint) fastHealthCheckCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:            fmt.Sprintf("Fast interval health checks add-on (%s)", trafficManagerBillingRegion(r.Region)),
		Unit:            "hours",
		UnitMultiplier:  engine.MonthToHourUnitMultiplier,
		UnitRounding:    int32Ptr(0),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(trafficManagerBillingRegion(r.Region)),
			Service:       strPtr("Traffic Manager"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: strPtr(r.sku())},
				{Key: "meterName", Value: strPtr(fmt.Sprintf("%s Fast Interval Health Check Add-ons", r.sku()))},
			},
		},
	}
}
