package azure

import (
	"fmt"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
)

// VirtualHub is the central hub in the "hub and spoke architecture" of Azure Virtual WAN.
// It enables transitive connectivity between endpoints that may be distributed across different types of 'spokes'.
//
// More resource information here: https://docs.microsoft.com/en-us/azure/virtual-wan/virtual-wan-about
// Pricing information here: https://azure.microsoft.com/en-us/pricing/details/virtual-wan/
type VirtualHub struct {
	// Address is the unique name of the resource in the IAC language.
	Address string
	// Region is the azure region the VirtualHub is provisioned within.
	Region string
	// SKU is the VirtualHub hub type. It can be one of: Basic|Standard.
	SKU string

	// MonthlyDataProcessedGB represents a usage cost for the amount of gb of data that is processed
	// through the hub on a monthly basis. It is a float to allow users to specify values whole GBs.
	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

// CoreType returns the name of this resource type
func (v *VirtualHub) CoreType() string {
	return "VirtualHub"
}

// UsageSchema defines a list which represents the usage schema of VirtualHubUsageSchema.
func (v *VirtualHub) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_data_processed_gb", DefaultValue: 0, ValueType: engine.Float64},
	}
}

// PopulateUsage parses the u engine.ConsumptionProfile into the VirtualHub.
// It uses the `c3x_usage` struct tags to populate data into the VirtualHub.
func (v *VirtualHub) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(v, u)
}

// BuildResource builds a engine.Estimate from a valid VirtualHub.
// It returns VirtualHub as a *engine.Estimate with 2 cost components provided.
// These cost components are only applicable if the VirtualHub is type Standard.
// The Basic hub is provided free by azure.
// See here: https://azure.microsoft.com/en-us/pricing/details/virtual-wan/ for more information.
//
// This method is called after the resource is initialised by an iac provider.
// See providers folder for more information.
func (v *VirtualHub) BuildResource() *engine.Estimate {
	if v.SKU == "Basic" {
		return &engine.Estimate{
			Name:        v.Address,
			UsageSchema: v.UsageSchema(),
			NoPrice:     true,
			IsSkipped:   true,
		}
	}

	components := []*engine.LineItem{
		v.deploymentHours(),
	}

	if v.MonthlyDataProcessedGB != nil {
		components = append(components, v.dataProcessed())
	}

	return &engine.Estimate{
		Name:           v.Address,
		UsageSchema:    v.UsageSchema(),
		CostComponents: components,
	}
}

func (v VirtualHub) dataProcessed() *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data processed",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromFloat(*v.MonthlyDataProcessedGB)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(v.Region),
			Service:       strPtr("Virtual WAN"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: v.hubTypeFilter()},
				{Key: "meterName", Value: v.hubTypeDataFilter()},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
		UsageBased: true,
	}
}

func (v VirtualHub) deploymentHours() *engine.LineItem {
	return &engine.LineItem{
		Name:           "Deployment",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        strPtr(v.Region),
			Service:       strPtr("Virtual WAN"),
			ProductFamily: strPtr("Networking"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "skuName", Value: v.hubTypeFilter()},
				{Key: "meterName", Value: v.hubTypeUnitFilter()},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("Consumption"),
		},
	}
}

func (v VirtualHub) hubTypeFilter() *string {
	return strPtr(fmt.Sprintf("%s Hub", v.SKU))
}

func (v VirtualHub) hubTypeUnitFilter() *string {
	return strPtr(fmt.Sprintf("%s Hub Unit", v.SKU))
}

func (v VirtualHub) hubTypeDataFilter() *string {
	return strPtr(fmt.Sprintf("%s Hub Data Processed", v.SKU))
}
