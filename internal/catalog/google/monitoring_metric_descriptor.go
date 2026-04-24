package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"

	"github.com/c3xdev/c3x/internal/usage"
)

type MonitoringMetricDescriptor struct {
	Address                 string
	MonthlyMonitoringDataMB *int64 `c3x_usage:"monthly_monitoring_data_mb"`
	MonthlyAPICalls         *int64 `c3x_usage:"monthly_api_calls"`
}

func (r *MonitoringMetricDescriptor) CoreType() string {
	return "MonitoringMetricDescriptor"
}

func (r *MonitoringMetricDescriptor) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{
		{Key: "monthly_monitoring_data_mb", ValueType: engine.Int64, DefaultValue: 0},
		{Key: "monthly_api_calls", ValueType: engine.Int64, DefaultValue: 0},
	}
}

func (r *MonitoringMetricDescriptor) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *MonitoringMetricDescriptor) BuildResource() *engine.Estimate {
	costComponents := []*engine.LineItem{}

	var monitoringDataMB *decimal.Decimal
	if r.MonthlyMonitoringDataMB != nil {
		monitoringDataMB = decimalPtr(decimal.NewFromInt(*r.MonthlyMonitoringDataMB))

		monitoringDataTiers := usage.CalculateTierBuckets(*monitoringDataMB, []int{100000, 150000})

		if monitoringDataTiers[0].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.monitoringDataCostComponent("Monitoring data (first 100K)", "150", &monitoringDataTiers[0]))
		}

		if monitoringDataTiers[1].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.monitoringDataCostComponent("Monitoring data (next 150K)", "100000", &monitoringDataTiers[1]))
		}

		if monitoringDataTiers[2].GreaterThan(decimal.Zero) {
			costComponents = append(costComponents, r.monitoringDataCostComponent("Monitoring data (over 250K)", "250000", &monitoringDataTiers[2]))
		}
	} else {
		var unknown *decimal.Decimal
		costComponents = append(costComponents, r.monitoringDataCostComponent("Monitoring data (first 100K)", "150", unknown))
	}

	var apiCalls *decimal.Decimal
	if r.MonthlyAPICalls != nil {
		apiCalls = decimalPtr(decimal.NewFromInt(*r.MonthlyAPICalls))
	}

	costComponents = append(costComponents, r.apiCallsCostComponent(apiCalls))
	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *MonitoringMetricDescriptor) monitoringDataCostComponent(name string, usageTier string, monitoringDataMB *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            name,
		Unit:            "MB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: monitoringDataMB,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Service:       strPtr("Cloud Monitoring"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: strPtr("/Metric Volume/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr(usageTier),
		},
		UsageBased: true,
	}
}

func (r *MonitoringMetricDescriptor) apiCallsCostComponent(apiCalls *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "API calls",
		Unit:            "1k calls",
		UnitMultiplier:  decimal.NewFromInt(1000),
		MonthlyQuantity: apiCalls,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Service:       strPtr("Cloud Monitoring"),
			ProductFamily: strPtr("ApplicationServices"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: strPtr("/Monitoring API Requests/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			StartUsageAmount: strPtr("1000000"),
		},
		UsageBased: true,
	}
}
