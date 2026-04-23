package azure

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"
	"github.com/shopspring/decimal"
)

// MonitorMetricAlert struct represents an Azure Monitor Metric Group.
//
// Resource information: https://learn.microsoft.com/en-us/azure/azure-monitor/alerts/alerts-overview
// Pricing information: https://azure.microsoft.com/en-us/pricing/details/monitor/
type MonitorMetricAlert struct {
	Address string
	Region  string

	Enabled                        bool
	ScopeCount                     int
	CriteriaDimensionsCount        int
	DynamicCriteriaDimensionsCount int
}

func (r *MonitorMetricAlert) CoreType() string {
	return "MonitorMetricAlert"
}

func (r *MonitorMetricAlert) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{}
}

// PopulateUsage parses the u engine.ConsumptionProfile
// It uses the `c3x_usage` struct tags to populate data.
func (r *MonitorMetricAlert) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

// BuildResource builds a engine.Estimate from the struct.
// This method is called after the resource is initialised by an IaC provider.
// See providers folder for more information.
func (r *MonitorMetricAlert) BuildResource() *engine.Estimate {
	if !r.Enabled {
		return &engine.Estimate{
			Name: r.Address,
		}
	}

	costComponents := []*engine.LineItem{}

	totalTimeSeries := int64(r.ScopeCount * (r.CriteriaDimensionsCount + r.DynamicCriteriaDimensionsCount))
	if totalTimeSeries > 0 {
		costComponents = append(costComponents, r.alertMetricMonitoringCostComponent(totalTimeSeries))
	}

	dynamicTimeSeries := int64(r.ScopeCount * r.DynamicCriteriaDimensionsCount)
	if dynamicTimeSeries > 0 {
		costComponents = append(costComponents, r.dynamicThresholdCostComponent(dynamicTimeSeries))
	}

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
	}
}

func (r *MonitorMetricAlert) alertMetricMonitoringCostComponent(quantity int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Metrics monitoring",
		Unit:            "time-series",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(quantity)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			Service:       strPtr("Azure Monitor"),
			ProductFamily: strPtr("Management and Governance"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", Value: strPtr("Alerts Metric Monitored")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("10"),
		},
	}
}

func (r *MonitorMetricAlert) dynamicThresholdCostComponent(quantity int64) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Dynamic threshold monitoring",
		Unit:            "time-series",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: decimalPtr(decimal.NewFromInt(quantity)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("azure"),
			Region:        r.normalizedRegion(),
			Service:       strPtr("Azure Monitor"),
			ProductFamily: strPtr("Management and Governance"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "meterName", Value: strPtr("Alerts Dynamic Threshold")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("0"),
		},
	}
}

func (r *MonitorMetricAlert) normalizedRegion() *string {
	if r.Region == "global" {
		return strPtr("Global")
	}
	return strPtr(r.Region)
}
