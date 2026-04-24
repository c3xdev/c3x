package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ComputeTargetGRPCProxy struct {
	Address                string
	Region                 string
	MonthlyProxyInstances  *float64 `c3x_usage:"monthly_proxy_instances"`
	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

func (r *ComputeTargetGRPCProxy) CoreType() string {
	return "ComputeTargetGRPCProxy"
}

func (r *ComputeTargetGRPCProxy) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_proxy_instances", ValueType: engine.Float64, DefaultValue: 0.000000}, {Key: "monthly_data_processed_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *ComputeTargetGRPCProxy) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ComputeTargetGRPCProxy) BuildResource() *engine.Estimate {
	var monthlyProxyInstances, monthlyDataProcessedGb *decimal.Decimal
	region := r.Region
	costComponents := make([]*engine.LineItem, 0)

	if r.MonthlyProxyInstances != nil {
		monthlyProxyInstances = decimalPtr(decimal.NewFromFloat(*r.MonthlyProxyInstances))
	}

	costComponents = append(costComponents, r.proxyInstanceCostComponent(monthlyProxyInstances))

	if r.MonthlyDataProcessedGB != nil {
		monthlyDataProcessedGb = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataProcessedGB))
	}

	costComponents = append(costComponents, dataProcessedCostComponent(region, monthlyDataProcessedGb))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *ComputeTargetGRPCProxy) proxyInstanceCostComponent(instanceCount *decimal.Decimal) *engine.LineItem {
	var quantity *decimal.Decimal
	if instanceCount != nil {
		instanceHours := engine.HourToMonthUnitMultiplier.Mul(*instanceCount)
		quantity = &instanceHours
	}

	return &engine.LineItem{
		Name:            "Proxy instance",
		Unit:            "hours",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: strPtr("/^Network Load Balancing: Forwarding Rule Minimum/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("OnDemand"),
		},
		UsageBased: true,
	}
}

func dataProcessedCostComponent(region string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data processed",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(region),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: strPtr("/^Network Internal Load Balancing: Data Processing/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("OnDemand"),
		},
		UsageBased: true,
	}
}
