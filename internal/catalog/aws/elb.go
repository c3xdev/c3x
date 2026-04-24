package aws

import (
	"github.com/c3xdev/c3x/internal/catalog"
	"github.com/c3xdev/c3x/internal/engine"

	"github.com/shopspring/decimal"
)

type ELB struct {
	Address                string
	Region                 string
	MonthlyDataProcessedGB *float64 `c3x_usage:"monthly_data_processed_gb"`
}

var ELBUsageSchema = []*engine.ConsumptionField{
	{Key: "monthly_data_processed_gb", ValueType: engine.Float64, DefaultValue: 0},
}

func (r *ELB) CoreType() string {
	return "ELB"
}

func (r *ELB) UsageSchema() []*engine.ConsumptionField {
	return ELBUsageSchema
}

func (r *ELB) PopulateUsage(u *engine.ConsumptionProfile) {
	catalog.PopulateArgsWithUsage(r, u)
}

func (r *ELB) BuildResource() *engine.Estimate {
	var dataProcessed *decimal.Decimal
	if r.MonthlyDataProcessedGB != nil {
		dataProcessed = decimalPtr(decimal.NewFromFloat(*r.MonthlyDataProcessedGB))
	}

	return &engine.Estimate{
		Name: r.Address,
		CostComponents: []*engine.LineItem{
			r.lbCostComponent(),
			r.dataProcessedCostComponent(dataProcessed),
		},
		UsageSchema: r.UsageSchema(),
	}
}

func (r *ELB) lbCostComponent() *engine.LineItem {
	return &engine.LineItem{
		Name:           "Classic load balancer",
		Unit:           "hours",
		HourlyQuantity: decimalPtr(decimal.NewFromInt(1)),
		UnitMultiplier: decimal.NewFromInt(1),
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSELB"),
			ProductFamily: strPtr("Load Balancer"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "locationType", Value: strPtr("AWS Region")},
				{Key: "usagetype", ValueRegex: strPtr("/LoadBalancerUsage/")},
			},
		},
	}
}

func (r *ELB) dataProcessedCostComponent(dataProcessed *decimal.Decimal) *engine.LineItem {
	return &engine.LineItem{
		Name:            "Data processed",
		Unit:            "GB",
		UnitMultiplier:  decimal.NewFromInt(1),
		MonthlyQuantity: dataProcessed,
		ProductFilter: &engine.ProductSelector{
			VendorName:    strPtr("aws"),
			Region:        strPtr(r.Region),
			Service:       strPtr("AWSELB"),
			ProductFamily: strPtr("Load Balancer"),
			AttributeFilters: []*engine.AttributeMatch{
				{Key: "usagetype", ValueRegex: strPtr("/DataProcessing-Bytes/")},
			},
		},
		UsageBased: true,
	}
}
