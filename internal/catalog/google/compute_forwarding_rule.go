package google

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ComputeForwardingRule struct {
	Address              string
	Region               string
	MonthlyIngressDataGB *float64 `c3x_usage:"monthly_ingress_data_gb"`
}

func (r *ComputeForwardingRule) CoreType() string {
	return "ComputeForwardingRule"
}

func (r *ComputeForwardingRule) UsageSchema() []*engine.ConsumptionField {
	return []*engine.ConsumptionField{{Key: "monthly_ingress_data_gb", ValueType: engine.Float64, DefaultValue: 0}}
}

func (r *ComputeForwardingRule) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ComputeForwardingRule) BuildResource() *engine.Estimate {
	var monthlyIngressDataGb *decimal.Decimal
	region := r.Region
	costComponents := make([]*engine.LineItem, 0)

	costComponents = append(costComponents, r.computeForwardingCostComponent())

	if r.MonthlyIngressDataGB != nil {
		monthlyIngressDataGb = decimalPtr(decimal.NewFromFloat(*r.MonthlyIngressDataGB))
	}

	costComponents = append(costComponents, computeIngressDataCostComponent(region, monthlyIngressDataGb))

	return &engine.Estimate{
		Name:           r.Address,
		CostComponents: costComponents,
		UsageSchema:    r.UsageSchema(),
	}
}

func (r *ComputeForwardingRule) computeForwardingCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "Forwarding rules",
		Unit:           "hours",
		UnitMultiplier: decimal.NewFromInt(1),
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(r.Region),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: strPtr("/Forwarding Rule Additional/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("OnDemand"),
		},
	}
}

func computeIngressDataCostComponent(region string, quantity *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Ingress data",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: quantity,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("gcp"),
			Region:        strPtr(region),
			Service:       strPtr("Compute Engine"),
			ProductFamily: strPtr("Network"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "description", ValueRegex: strPtr("/^Network Load Balancing: Data Processing Charge/i")},
			},
		},
		PriceFilter: &engine.RateSelector{
			PurchaseOption: strPtr("OnDemand"),
		},
		UsageBased: true,
	}
}
